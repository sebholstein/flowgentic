package project

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDir creates a temp directory with a realistic file tree for testing.
// Structure:
//
//	root/
//	  .git/            (always hidden)
//	    config
//	  .gitignore       (ignores: node_modules/, *.log)
//	  src/
//	    main.go
//	    sub/
//	      .gitignore   (ignores: tmp/)
//	      helper.go
//	      tmp/         (ignored by nested .gitignore)
//	        junk.txt
//	  node_modules/    (ignored by root .gitignore)
//	    pkg/
//	  README.md
//	  build.log        (ignored by root .gitignore)
//	  empty_dir/       (no children)
func setupTestDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	dirs := []string{
		".git",
		"src/sub/tmp",
		"node_modules/pkg",
		"empty_dir",
	}
	for _, d := range dirs {
		require.NoError(t, os.MkdirAll(filepath.Join(root, d), 0o755))
	}

	files := map[string]string{
		".git/config":              "[core]",
		".gitignore":               "node_modules/\n*.log\n",
		"src/main.go":              "package main",
		"src/sub/.gitignore":       "tmp/\n",
		"src/sub/helper.go":        "package sub",
		"src/sub/tmp/junk.txt":     "junk",
		"node_modules/pkg/index.js": "module.exports = {}",
		"README.md":                "# test",
		"build.log":                "some log output",
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(content), 0o644))
	}

	return root
}

func TestListTree_FullRecursiveTree(t *testing.T) {
	root := setupTestDir(t)
	svc := NewProjectService()

	entries, err := svc.ListTree(root)
	require.NoError(t, err)

	paths := entryPaths(entries)

	// Expected visible entries (dirs-first at each level, then alpha):
	// empty_dir, src, src/sub, src/sub/.gitignore, src/sub/helper.go, src/main.go, .gitignore, README.md
	// Filtered out: .git/*, node_modules/*, build.log, src/sub/tmp/*
	assert.Contains(t, paths, "src")
	assert.Contains(t, paths, "src/sub")
	assert.Contains(t, paths, "src/sub/helper.go")
	assert.Contains(t, paths, "src/main.go")
	assert.Contains(t, paths, ".gitignore")
	assert.Contains(t, paths, "README.md")
	assert.Contains(t, paths, "empty_dir")

	// Filtered out.
	assert.NotContains(t, paths, ".git")
	assert.NotContains(t, paths, ".git/config")
	assert.NotContains(t, paths, "node_modules")
	assert.NotContains(t, paths, "node_modules/pkg")
	assert.NotContains(t, paths, "build.log")
	assert.NotContains(t, paths, "src/sub/tmp")
	assert.NotContains(t, paths, "src/sub/tmp/junk.txt")
}

func TestListTree_DirsFirstOrdering(t *testing.T) {
	root := setupTestDir(t)
	svc := NewProjectService()

	entries, err := svc.ListTree(root)
	require.NoError(t, err)

	// At root level, dirs should come before files.
	// Find the first file at root level (path has no separator).
	firstFileIdx := -1
	lastDirIdx := -1
	for i, e := range entries {
		if !isRootLevel(e.Path) {
			continue
		}
		if e.IsDir {
			lastDirIdx = i
		} else if firstFileIdx == -1 {
			firstFileIdx = i
		}
	}
	if firstFileIdx != -1 && lastDirIdx != -1 {
		assert.Less(t, lastDirIdx, firstFileIdx, "root-level dirs should come before files")
	}
}

func TestListTree_ChildrenFollowParent(t *testing.T) {
	root := setupTestDir(t)
	svc := NewProjectService()

	entries, err := svc.ListTree(root)
	require.NoError(t, err)

	// Find "src" and verify "src/sub" appears after it.
	srcIdx := -1
	srcSubIdx := -1
	for i, e := range entries {
		if e.Path == "src" {
			srcIdx = i
		}
		if e.Path == "src/sub" {
			srcSubIdx = i
		}
	}
	require.NotEqual(t, -1, srcIdx, "src should be in tree")
	require.NotEqual(t, -1, srcSubIdx, "src/sub should be in tree")
	assert.Less(t, srcIdx, srcSubIdx, "src/sub should come after src")
}

func TestListTree_NestedGitignore(t *testing.T) {
	root := setupTestDir(t)
	svc := NewProjectService()

	entries, err := svc.ListTree(root)
	require.NoError(t, err)

	paths := entryPaths(entries)
	// src/sub/.gitignore ignores tmp/
	assert.NotContains(t, paths, "src/sub/tmp")
	assert.NotContains(t, paths, "src/sub/tmp/junk.txt")
	// But src/sub/helper.go is visible.
	assert.Contains(t, paths, "src/sub/helper.go")
}

func TestListTree_GitAlwaysHidden(t *testing.T) {
	root := setupTestDir(t)
	svc := NewProjectService()

	entries, err := svc.ListTree(root)
	require.NoError(t, err)

	for _, e := range entries {
		assert.NotEqual(t, ".git", e.Name, "no entry should be named .git")
	}
}

func TestListTree_FileSize(t *testing.T) {
	root := setupTestDir(t)
	svc := NewProjectService()

	entries, err := svc.ListTree(root)
	require.NoError(t, err)

	m := entryMap(entries)
	assert.Greater(t, m["src/main.go"].Size, int64(0), "file should have non-zero size")
	assert.Equal(t, int64(0), m["src"].Size, "dir should have zero size")
}

func TestWatchTree_CreateFileInSubdir(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "sub"), 0o755))

	svc := NewProjectService()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := svc.WatchTree(ctx, root)
	require.NoError(t, err)

	// Create a file in a subdirectory — should be caught by recursive watcher.
	newFile := filepath.Join(root, "sub", "hello.txt")
	require.NoError(t, os.WriteFile(newFile, []byte("hello"), 0o644))

	ev := waitForEvent(t, ch, 2*time.Second)
	require.NotNil(t, ev, "expected create event")
	assert.Equal(t, FileEventCreated, ev.Type)
	assert.Equal(t, filepath.Join("sub", "hello.txt"), ev.Entry.Path)
}

func TestWatchTree_CreateAndRemoveAtRoot(t *testing.T) {
	root := t.TempDir()
	svc := NewProjectService()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := svc.WatchTree(ctx, root)
	require.NoError(t, err)

	// Create a file at root.
	newFile := filepath.Join(root, "hello.txt")
	require.NoError(t, os.WriteFile(newFile, []byte("hello"), 0o644))

	ev := waitForEvent(t, ch, 2*time.Second)
	require.NotNil(t, ev, "expected create event")
	assert.Equal(t, FileEventCreated, ev.Type)
	assert.Equal(t, "hello.txt", ev.Entry.Name)

	// Remove it.
	require.NoError(t, os.Remove(newFile))

	ev = waitForEvent(t, ch, 2*time.Second)
	require.NotNil(t, ev, "expected remove event")
	assert.Equal(t, FileEventRemoved, ev.Type)
	assert.Equal(t, "hello.txt", ev.Entry.Name)
}

func TestWatchTree_NewDirGetsWatched(t *testing.T) {
	root := t.TempDir()
	svc := NewProjectService()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := svc.WatchTree(ctx, root)
	require.NoError(t, err)

	// Create a new directory.
	newDir := filepath.Join(root, "newdir")
	require.NoError(t, os.Mkdir(newDir, 0o755))

	ev := waitForEvent(t, ch, 2*time.Second)
	require.NotNil(t, ev, "expected create event for dir")
	assert.Equal(t, FileEventCreated, ev.Type)
	assert.Equal(t, "newdir", ev.Entry.Name)
	assert.True(t, ev.Entry.IsDir)

	// Now create a file inside the new directory — it should also be watched.
	require.NoError(t, os.WriteFile(filepath.Join(newDir, "file.txt"), []byte("hi"), 0o644))

	ev = waitForEvent(t, ch, 2*time.Second)
	require.NotNil(t, ev, "expected create event for file in new dir")
	assert.Equal(t, FileEventCreated, ev.Type)
	assert.Equal(t, filepath.Join("newdir", "file.txt"), ev.Entry.Path)
}

func TestWatchTree_IgnoredFilesNoEvents(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, ".gitignore"), []byte("*.log\n"), 0o644))

	svc := NewProjectService()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := svc.WatchTree(ctx, root)
	require.NoError(t, err)

	// Create an ignored file.
	require.NoError(t, os.WriteFile(filepath.Join(root, "debug.log"), []byte("log"), 0o644))

	// Create a visible file.
	require.NoError(t, os.WriteFile(filepath.Join(root, "visible.txt"), []byte("hi"), 0o644))

	ev := waitForEvent(t, ch, 2*time.Second)
	require.NotNil(t, ev)
	assert.Equal(t, "visible.txt", ev.Entry.Name, "should only see visible file event")
}

// helpers

func entryPaths(entries []FileEntry) []string {
	paths := make([]string, len(entries))
	for i, e := range entries {
		paths[i] = e.Path
	}
	return paths
}

func entryMap(entries []FileEntry) map[string]FileEntry {
	m := make(map[string]FileEntry, len(entries))
	for _, e := range entries {
		m[e.Path] = e
	}
	return m
}

func isRootLevel(path string) bool {
	return !contains(path, string(filepath.Separator))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsStr(s, substr)))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func waitForEvent(t *testing.T, ch <-chan FileEvent, timeout time.Duration) *FileEvent {
	t.Helper()
	select {
	case ev, ok := <-ch:
		if !ok {
			return nil
		}
		return &ev
	case <-time.After(timeout):
		t.Log("timed out waiting for event")
		return nil
	}
}
