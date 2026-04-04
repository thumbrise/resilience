// Copyright 2026 thumbrise
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/thumbrise/resilience/pkg/multimod/testutil"
)

// chdir changes cwd to dir for the duration of the test.
func chdir(t *testing.T, dir string) {
	t.Helper()

	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = os.Chdir(prev) })
}

func TestBoot_MultiModule(t *testing.T) {
	root := testutil.Scaffold(t, "valid_project")
	chdir(t, root.String())

	result, err := NewBootloader().Boot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !sameDir(t, result.RootDir.String(), root.String()) {
		t.Errorf("RootDir = %q, want %q", result.RootDir, root)
	}

	if !result.MultiModule {
		t.Error("MultiModule = false, want true")
	}
}

func TestBoot_SingleModule(t *testing.T) {
	root := testutil.Scaffold(t, "single_module")
	chdir(t, root.String())

	result, err := NewBootloader().Boot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.MultiModule {
		t.Error("MultiModule = true, want false")
	}
}

func TestBoot_NoGoMod(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	_, err := NewBootloader().Boot()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrNoGoMod) {
		t.Errorf("error = %v, want ErrNoGoMod", err)
	}
}

func TestBoot_FromSubdir_Fails(t *testing.T) {
	root := testutil.Scaffold(t, "valid_project")

	subDir := filepath.Join(root.String(), "sub")
	chdir(t, subDir)

	// Boot from sub-directory should fail — cwd has go.mod but it's a sub-module,
	// not the project root. The sub's go.mod exists so Boot succeeds,
	// but MultiModule will be false (no sub-modules under sub/).
	result, err := NewBootloader().Boot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Sub-directory has go.mod but no sub-modules — not multi-module from this cwd.
	if result.MultiModule {
		t.Error("MultiModule = true from sub-directory, want false")
	}
}

func TestBoot_NoGitWarning(t *testing.T) {
	root := testutil.Scaffold(t, "valid_project")
	chdir(t, root.String())

	result, err := NewBootloader().Boot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No .git in scaffold — Boot should still work but warn.
	if !result.NoGit {
		t.Error("NoGit = false, want true (no .git in scaffolded dir)")
	}
}

func TestBoot_WithGit_NoWarning(t *testing.T) {
	root := testutil.Scaffold(t, "valid_project")

	// Create .git directory.
	if err := os.Mkdir(filepath.Join(root.String(), ".git"), 0o750); err != nil {
		t.Fatal(err)
	}

	chdir(t, root.String())

	result, err := NewBootloader().Boot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.NoGit {
		t.Error("NoGit = true, want false (.git exists)")
	}
}

func TestBoot_ExcludedDirs_NotMultiModule(t *testing.T) {
	root := testutil.Scaffold(t, "excluded_dirs")
	chdir(t, root.String())

	result, err := NewBootloader().Boot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.MultiModule {
		t.Error("MultiModule = true, want false (all sub go.mods in excluded dirs)")
	}
}

func TestBoot_CwdIsRoot(t *testing.T) {
	root := testutil.Scaffold(t, "multi_module")
	chdir(t, root.String())

	result, err := NewBootloader().Boot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !sameDir(t, result.RootDir.String(), root.String()) {
		t.Errorf("RootDir = %q, want cwd %q", result.RootDir, root)
	}
}

// sameDir compares two directory paths using os.SameFile to handle symlinks
// (e.g. macOS /var → /private/var).
func sameDir(t *testing.T, a, b string) bool {
	t.Helper()

	infoA, err := os.Stat(a)
	if err != nil {
		t.Fatalf("stat %q: %v", a, err)
	}

	infoB, err := os.Stat(b)
	if err != nil {
		t.Fatalf("stat %q: %v", b, err)
	}

	return os.SameFile(infoA, infoB)
}
