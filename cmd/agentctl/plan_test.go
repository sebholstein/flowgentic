package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAndBuildPlanUnit_DetectsDependencyCycle(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "plan.md"), `---
title: "Cycle test"
---

Plan body.
`)
	mustWriteFile(t, filepath.Join(dir, "tasks", "01-a.md"), `---
id: a
depends_on: [b]
---

Task A body.
`)
	mustWriteFile(t, filepath.Join(dir, "tasks", "02-b.md"), `---
id: b
depends_on: [a]
---

Task B body.
`)

	_, errs := validateAndBuildPlanUnit(planAllocation{
		ThreadID: "thread-1",
		PlanDir:  dir,
	})

	if len(errs) == 0 {
		t.Fatalf("expected validation errors, got none")
	}
	joined := strings.Join(errs, "\n")
	if !strings.Contains(joined, "circular dependency detected") {
		t.Fatalf("expected cycle error, got: %s", joined)
	}
}

func TestValidateAndBuildPlanUnit_AllowsAcyclicDependencies(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "plan.md"), `---
title: "Acyclic test"
---

Plan body.
`)
	mustWriteFile(t, filepath.Join(dir, "tasks", "01-a.md"), `---
id: a
depends_on: []
---

Task A body.
`)
	mustWriteFile(t, filepath.Join(dir, "tasks", "02-b.md"), `---
id: b
depends_on: [a]
---

Task B body.
`)

	_, errs := validateAndBuildPlanUnit(planAllocation{
		ThreadID: "thread-1",
		PlanDir:  dir,
	})
	if len(errs) > 0 {
		t.Fatalf("expected no validation errors, got: %v", errs)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

