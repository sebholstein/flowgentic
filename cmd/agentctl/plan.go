package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
	"gopkg.in/yaml.v3"
)

var (
	taskFilePattern = regexp.MustCompile(`^\d{2}-[a-z0-9]+(-[a-z0-9]+)*\.md$`)
	taskIDPattern   = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

type planFileFrontmatter struct {
	Title string `yaml:"title"`
}

type taskFileFrontmatter struct {
	ID        string   `yaml:"id"`
	DependsOn []string `yaml:"depends_on"`
	Agent     string   `yaml:"agent"`
	Subtasks  []string `yaml:"subtasks"`
}

type submittedPlanPayload struct {
	Version int                 `json:"version"`
	Plans   []submittedPlanUnit `json:"plans"`
}

type submittedPlanUnit struct {
	ThreadID string               `json:"thread_id"`
	PlanDir  string               `json:"plan_dir"`
	Plan     submittedPlanContent `json:"plan"`
	Tasks    []submittedTask      `json:"tasks"`
}

type submittedPlanContent struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type submittedTask struct {
	Filename    string   `json:"filename"`
	ID          string   `json:"id"`
	DependsOn   []string `json:"depends_on,omitempty"`
	Agent       string   `json:"agent,omitempty"`
	Subtasks    []string `json:"subtasks,omitempty"`
	Description string   `json:"description"`
}

func planGetCurrentDir() (string, error) {
	st, err := loadOrInitPlanState()
	if err != nil {
		return "", err
	}
	return st.Current.PlanDir, nil
}

func planRequestThreadDir() (planAllocation, error) {
	st, err := loadOrInitPlanState()
	if err != nil {
		return planAllocation{}, err
	}
	return allocateAdditionalPlanDir(st)
}

func planRemoveThread(threadID string) error {
	if strings.TrimSpace(threadID) == "" {
		return fmt.Errorf("thread id is required")
	}

	st, err := loadOrInitPlanState()
	if err != nil {
		return err
	}

	idx := -1
	for i, a := range st.Additional {
		if a.ThreadID == threadID {
			idx = i
			if err := os.RemoveAll(a.PlanDir); err != nil {
				return fmt.Errorf("remove plan dir %q: %w", a.PlanDir, err)
			}
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("unknown thread id %q", threadID)
	}

	st.Additional = append(st.Additional[:idx], st.Additional[idx+1:]...)
	return savePlanState(st)
}

func planClearCurrent() error {
	st, err := loadOrInitPlanState()
	if err != nil {
		return err
	}
	if err := os.RemoveAll(st.Current.PlanDir); err != nil {
		return fmt.Errorf("clear current plan dir: %w", err)
	}
	return ensurePlanDir(st.Current.PlanDir)
}

func planCommit(ctx context.Context, agentName string) (int, error) {
	st, err := loadOrInitPlanState()
	if err != nil {
		return 0, err
	}

	allocs := make([]planAllocation, 0, 1+len(st.Additional))
	allocs = append(allocs, st.Current)
	allocs = append(allocs, st.Additional...)

	payload := submittedPlanPayload{Version: 1}
	var allErrs []string
	for _, a := range allocs {
		unit, errs := validateAndBuildPlanUnit(a)
		if len(errs) > 0 {
			allErrs = append(allErrs, errs...)
			continue
		}
		payload.Plans = append(payload.Plans, unit)
	}
	if len(allErrs) > 0 {
		return 0, errors.New("plan validation failed:\n- " + strings.Join(allErrs, "\n- "))
	}

	sessionID := os.Getenv(agentCtlSessionIDEnv)
	if sessionID == "" {
		return 0, fmt.Errorf("%s env not set", agentCtlSessionIDEnv)
	}
	if strings.TrimSpace(agentName) == "" {
		agentName = "codex"
	}
	agentProto, err := driver.ParseProtoAgent(agentName)
	if err != nil {
		return 0, fmt.Errorf("invalid --agent %q: %w", agentName, err)
	}

	blob, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("encode plan payload: %w", err)
	}

	client := newAgentCtlClient()
	_, err = client.SubmitPlan(ctx, connect.NewRequest(&workerv1.SubmitPlanRequest{
		SessionId: sessionID,
		Agent:     agentProto,
		Plan:      blob,
	}))
	if err != nil {
		return 0, fmt.Errorf("submit plan: %w", err)
	}
	return len(payload.Plans), nil
}

func validateAndBuildPlanUnit(a planAllocation) (submittedPlanUnit, []string) {
	var errs []string
	unit := submittedPlanUnit{ThreadID: a.ThreadID, PlanDir: a.PlanDir}

	planPath := filepath.Join(a.PlanDir, "plan.md")
	planRaw, err := os.ReadFile(planPath)
	if err != nil {
		return unit, []string{fmt.Sprintf("%s: read plan.md: %v", a.PlanDir, err)}
	}
	var pfm planFileFrontmatter
	planBody, err := decodeFrontmatter(planRaw, &pfm)
	if err != nil {
		return unit, []string{fmt.Sprintf("%s/plan.md: %v", a.PlanDir, err)}
	}
	if strings.TrimSpace(pfm.Title) == "" {
		errs = append(errs, fmt.Sprintf("%s/plan.md: title is required", a.PlanDir))
	}
	unit.Plan = submittedPlanContent{Title: pfm.Title, Body: strings.TrimSpace(planBody)}

	tasksDir := filepath.Join(a.PlanDir, "tasks")
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		errs = append(errs, fmt.Sprintf("%s: read tasks dir: %v", a.PlanDir, err))
		return unit, errs
	}

	type taskMeta struct {
		File string
		FM   taskFileFrontmatter
		Body string
	}
	var metas []taskMeta
	sortIndices := map[string]struct{}{}
	ids := map[string]string{} // id -> file

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !taskFilePattern.MatchString(name) {
			continue
		}
		sortKey := name[:2]
		if _, exists := sortIndices[sortKey]; exists {
			errs = append(errs, fmt.Sprintf("%s/tasks/%s: duplicate sort index %s", a.PlanDir, name, sortKey))
		}
		sortIndices[sortKey] = struct{}{}

		raw, err := os.ReadFile(filepath.Join(tasksDir, name))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s/tasks/%s: read: %v", a.PlanDir, name, err))
			continue
		}
		var tfm taskFileFrontmatter
		body, err := decodeFrontmatter(raw, &tfm)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s/tasks/%s: %v", a.PlanDir, name, err))
			continue
		}
		if !taskIDPattern.MatchString(tfm.ID) {
			errs = append(errs, fmt.Sprintf("%s/tasks/%s: invalid id %q", a.PlanDir, name, tfm.ID))
		}
		if tfm.ID == "" {
			errs = append(errs, fmt.Sprintf("%s/tasks/%s: id is required", a.PlanDir, name))
		}
		if prev, ok := ids[tfm.ID]; ok {
			errs = append(errs, fmt.Sprintf("%s/tasks/%s: duplicate id %q (already in %s)", a.PlanDir, name, tfm.ID, prev))
		}
		ids[tfm.ID] = name
		if strings.TrimSpace(body) == "" {
			errs = append(errs, fmt.Sprintf("%s/tasks/%s: task body is required", a.PlanDir, name))
		}

		metas = append(metas, taskMeta{
			File: name,
			FM:   tfm,
			Body: strings.TrimSpace(body),
		})
	}

	// depends_on references.
	graph := map[string][]string{}
	for _, m := range metas {
		graph[m.FM.ID] = append([]string(nil), m.FM.DependsOn...)
		for _, dep := range m.FM.DependsOn {
			if _, ok := ids[dep]; !ok {
				errs = append(errs, fmt.Sprintf("%s/tasks/%s: depends_on references unknown id %q", a.PlanDir, m.File, dep))
			}
		}
	}
	if cycle := findCycle(graph); len(cycle) > 0 {
		errs = append(errs, fmt.Sprintf("%s/tasks: circular dependency detected: %s", a.PlanDir, strings.Join(cycle, " -> ")))
	}

	slices.SortFunc(metas, func(a, b taskMeta) int { return strings.Compare(a.File, b.File) })
	for _, m := range metas {
		unit.Tasks = append(unit.Tasks, submittedTask{
			Filename:    m.File,
			ID:          m.FM.ID,
			DependsOn:   m.FM.DependsOn,
			Agent:       m.FM.Agent,
			Subtasks:    m.FM.Subtasks,
			Description: m.Body,
		})
	}
	return unit, errs
}

func decodeFrontmatter[T any](raw []byte, out *T) (string, error) {
	s := string(raw)
	if !strings.HasPrefix(s, "---\n") {
		return "", fmt.Errorf("missing YAML frontmatter start")
	}
	rest := s[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return "", fmt.Errorf("missing YAML frontmatter end")
	}

	yml := rest[:end]
	body := rest[end+len("\n---\n"):]

	dec := yaml.NewDecoder(strings.NewReader(yml))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		return "", fmt.Errorf("invalid frontmatter: %w", err)
	}
	return body, nil
}

func findCycle(graph map[string][]string) []string {
	const (
		unvisited = 0
		visiting  = 1
		visited   = 2
	)
	state := map[string]int{}
	stack := []string{}

	var dfs func(string) []string
	dfs = func(node string) []string {
		state[node] = visiting
		stack = append(stack, node)
		for _, dep := range graph[node] {
			switch state[dep] {
			case visiting:
				idx := 0
				for i := range stack {
					if stack[i] == dep {
						idx = i
						break
					}
				}
				cycle := append([]string{}, stack[idx:]...)
				cycle = append(cycle, dep)
				return cycle
			case unvisited:
				if c := dfs(dep); len(c) > 0 {
					return c
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[node] = visited
		return nil
	}

	for node := range graph {
		if state[node] == unvisited {
			if c := dfs(node); len(c) > 0 {
				return c
			}
		}
	}
	return nil
}
