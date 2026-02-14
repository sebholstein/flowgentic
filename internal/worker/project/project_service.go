package project

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/fsnotify/fsnotify"
	ignore "github.com/sabhiram/go-gitignore"
)

// FileEntry represents a single file or directory in the tree.
type FileEntry struct {
	Name  string
	Path  string // relative to cwd
	IsDir bool
	Size  int64
}

// FileEventType describes a filesystem change.
type FileEventType int

const (
	FileEventCreated FileEventType = iota + 1
	FileEventRemoved
)

// FileEvent is a single filesystem change notification.
type FileEvent struct {
	Type  FileEventType
	Entry FileEntry
}

// ProjectService provides recursive directory listing and filesystem watching
// with gitignore filtering.
type ProjectService struct{}

// NewProjectService creates a new ProjectService.
func NewProjectService() *ProjectService {
	return &ProjectService{}
}

// ListTree returns the full recursive file tree under cwd, filtered by
// gitignore rules. Entries are sorted directories-first then alphabetically
// at each level, with children listed immediately after their parent directory.
func (s *ProjectService) ListTree(cwd string) ([]FileEntry, error) {
	cwd = filepath.Clean(cwd)
	matchers := loadGitignoreMatchers(cwd, "")
	return listTreeRecursive(cwd, cwd, "", matchers)
}

// WatchTree watches cwd recursively for filesystem changes and sends events
// on the returned channel. The watcher stops when ctx is cancelled.
func (s *ProjectService) WatchTree(ctx context.Context, cwd string) (<-chan FileEvent, error) {
	cwd = filepath.Clean(cwd)
	matchers := loadGitignoreMatchers(cwd, "")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Walk the tree and add all visible directories to the watcher.
	if err := addWatchDirsRecursive(watcher, cwd, cwd, "", matchers); err != nil {
		watcher.Close()
		return nil, err
	}

	ch := make(chan FileEvent, 64)

	go func() {
		defer close(ch)
		defer watcher.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-watcher.Events:
				if !ok {
					return
				}
				fe := processWatchEvent(ev, cwd, watcher, matchers)
				if fe != nil {
					select {
					case ch <- *fe:
					case <-ctx.Done():
						return
					}
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	return ch, nil
}

// listTreeRecursive builds the tree for a single directory level, recursing into subdirs.
func listTreeRecursive(cwd, absDir, relDir string, matchers []*ignore.GitIgnore) ([]FileEntry, error) {
	dirEntries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, err
	}

	// Load any .gitignore in this directory and append to matchers for children.
	childMatchers := loadLocalGitignore(absDir, matchers)

	// Separate dirs and files, filter ignored entries.
	var dirs, files []fs.DirEntry
	for _, de := range dirEntries {
		entryRel := joinRel(relDir, de.Name())
		if shouldIgnore(de.Name(), entryRel, de.IsDir(), childMatchers) {
			continue
		}
		if de.IsDir() {
			dirs = append(dirs, de)
		} else {
			files = append(files, de)
		}
	}

	// Sort each group alphabetically.
	slices.SortFunc(dirs, func(a, b fs.DirEntry) int {
		return strings.Compare(strings.ToLower(a.Name()), strings.ToLower(b.Name()))
	})
	slices.SortFunc(files, func(a, b fs.DirEntry) int {
		return strings.Compare(strings.ToLower(a.Name()), strings.ToLower(b.Name()))
	})

	var entries []FileEntry

	// Dirs first, each followed by its recursive children.
	for _, de := range dirs {
		entryRel := joinRel(relDir, de.Name())
		entries = append(entries, FileEntry{
			Name:  de.Name(),
			Path:  entryRel,
			IsDir: true,
		})
		children, err := listTreeRecursive(cwd, filepath.Join(absDir, de.Name()), entryRel, childMatchers)
		if err != nil {
			return nil, err
		}
		entries = append(entries, children...)
	}

	// Then files.
	for _, de := range files {
		entryRel := joinRel(relDir, de.Name())
		entry := FileEntry{
			Name:  de.Name(),
			Path:  entryRel,
			IsDir: false,
		}
		if info, err := de.Info(); err == nil {
			entry.Size = info.Size()
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// addWatchDirsRecursive adds absDir and all visible subdirectories to the watcher.
func addWatchDirsRecursive(watcher *fsnotify.Watcher, cwd, absDir, relDir string, matchers []*ignore.GitIgnore) error {
	if err := watcher.Add(absDir); err != nil {
		return err
	}

	dirEntries, err := os.ReadDir(absDir)
	if err != nil {
		return err
	}

	childMatchers := loadLocalGitignore(absDir, matchers)

	for _, de := range dirEntries {
		if !de.IsDir() {
			continue
		}
		entryRel := joinRel(relDir, de.Name())
		if shouldIgnore(de.Name(), entryRel, true, childMatchers) {
			continue
		}
		if err := addWatchDirsRecursive(watcher, cwd, filepath.Join(absDir, de.Name()), entryRel, childMatchers); err != nil {
			return err
		}
	}

	return nil
}

// processWatchEvent converts an fsnotify event into a FileEvent, or nil if ignored.
// For new directories it also adds them to the watcher.
func processWatchEvent(ev fsnotify.Event, cwd string, watcher *fsnotify.Watcher, matchers []*ignore.GitIgnore) *FileEvent {
	absPath := ev.Name
	relPath, err := filepath.Rel(cwd, absPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return nil
	}
	name := filepath.Base(absPath)

	// Build matchers chain for the parent directory.
	parentRel := filepath.Dir(relPath)
	if parentRel == "." {
		parentRel = ""
	}
	allMatchers := loadGitignoreMatchers(cwd, parentRel)

	switch {
	case ev.Has(fsnotify.Create):
		info, err := os.Stat(absPath)
		if err != nil {
			return nil
		}
		isDir := info.IsDir()
		if shouldIgnore(name, relPath, isDir, allMatchers) {
			return nil
		}
		// If a new directory is created, add it (and children) to the watcher.
		if isDir {
			_ = addWatchDirsRecursive(watcher, cwd, absPath, relPath, allMatchers)
		}
		return &FileEvent{
			Type: FileEventCreated,
			Entry: FileEntry{
				Name:  name,
				Path:  relPath,
				IsDir: isDir,
				Size:  info.Size(),
			},
		}

	case ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename):
		if shouldIgnore(name, relPath, false, allMatchers) {
			return nil
		}
		return &FileEvent{
			Type:  FileEventRemoved,
			Entry: FileEntry{Name: name, Path: relPath},
		}

	default:
		return nil
	}
}

// loadGitignoreMatchers collects .gitignore files from cwd down to cwd/relPath.
func loadGitignoreMatchers(cwd, relPath string) []*ignore.GitIgnore {
	var matchers []*ignore.GitIgnore

	parts := splitPath(relPath)
	dirs := make([]string, 0, len(parts)+1)
	dirs = append(dirs, cwd)
	cur := cwd
	for _, p := range parts {
		cur = filepath.Join(cur, p)
		dirs = append(dirs, cur)
	}

	for _, dir := range dirs {
		gi := filepath.Join(dir, ".gitignore")
		if m, err := ignore.CompileIgnoreFile(gi); err == nil {
			matchers = append(matchers, m)
		}
	}

	return matchers
}

// loadLocalGitignore appends the .gitignore in absDir (if present) to the existing matchers.
func loadLocalGitignore(absDir string, matchers []*ignore.GitIgnore) []*ignore.GitIgnore {
	gi := filepath.Join(absDir, ".gitignore")
	m, err := ignore.CompileIgnoreFile(gi)
	if err != nil {
		return matchers
	}
	out := make([]*ignore.GitIgnore, len(matchers)+1)
	copy(out, matchers)
	out[len(matchers)] = m
	return out
}

// shouldIgnore returns true if the entry should be hidden.
func shouldIgnore(name, relPath string, isDir bool, matchers []*ignore.GitIgnore) bool {
	if name == ".git" {
		return true
	}
	matchPath := relPath
	if isDir {
		matchPath += "/"
	}
	for _, m := range matchers {
		if m.MatchesPath(matchPath) {
			return true
		}
	}
	return false
}

// joinRel joins a relative directory path with a filename, handling the root case.
func joinRel(relDir, name string) string {
	if relDir == "" {
		return name
	}
	return filepath.Join(relDir, name)
}

// splitPath splits a relative path into its components, skipping empty parts.
func splitPath(p string) []string {
	p = filepath.Clean(p)
	if p == "." || p == "" {
		return nil
	}
	return strings.Split(p, string(filepath.Separator))
}
