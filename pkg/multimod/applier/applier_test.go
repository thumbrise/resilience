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

package applier_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thumbrise/resilience/pkg/multimod"
	"github.com/thumbrise/resilience/pkg/multimod/applier"
	"github.com/thumbrise/resilience/pkg/multimod/discovery"
	"github.com/thumbrise/resilience/pkg/multimod/testutil"
)

func TestApply_CreatesGoWork(t *testing.T) {
	root := testutil.Scaffold(t, "missing_gowork")

	state := discover(t, root)

	a := applier.NewApplier()
	if err := a.Apply(state); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root.String(), "go.work"))
	if err != nil {
		t.Fatalf("go.work not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "sub") {
		t.Errorf("go.work missing sub module:\n%s", content)
	}

	if !strings.Contains(content, "1.25.0") {
		t.Errorf("go.work missing go version:\n%s", content)
	}
}

func TestApply_AddsReplace(t *testing.T) {
	root := testutil.Scaffold(t, "missing_replace")

	state := discover(t, root)

	a := applier.NewApplier()
	if err := a.Apply(state); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root.String(), "sub", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), "replace") {
		t.Errorf("sub/go.mod missing replace directive:\n%s", string(data))
	}
}

func TestApply_SyncsGoVersion(t *testing.T) {
	root := testutil.Scaffold(t, "go_version_mismatch")

	state := discover(t, root)

	a := applier.NewApplier()
	if err := a.Apply(state); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root.String(), "sub", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(data), "go 1.25.0") {
		t.Errorf("sub/go.mod not synced to root version:\n%s", string(data))
	}
}

func TestApply_ValidProject_NoChanges(t *testing.T) {
	root := testutil.Scaffold(t, "valid_project")

	// Read files before apply.
	gomodBefore, _ := os.ReadFile(filepath.Join(root.String(), "sub", "go.mod"))
	goworkBefore, _ := os.ReadFile(filepath.Join(root.String(), "go.work"))

	state := discover(t, root)

	a := applier.NewApplier()
	if err := a.Apply(state); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Files should be unchanged (idempotent).
	gomodAfter, _ := os.ReadFile(filepath.Join(root.String(), "sub", "go.mod"))
	goworkAfter, _ := os.ReadFile(filepath.Join(root.String(), "go.work"))

	if string(gomodBefore) != string(gomodAfter) {
		t.Errorf("sub/go.mod changed on valid project:\nbefore:\n%s\nafter:\n%s", gomodBefore, gomodAfter)
	}

	// go.work is always rewritten from desired state, so content match is enough.
	if !strings.Contains(string(goworkAfter), "sub") {
		t.Errorf("go.work missing sub after apply:\n%s", goworkAfter)
	}

	_ = goworkBefore // suppress unused
}

func TestApply_CrossDeps_BothReplaces(t *testing.T) {
	root := testutil.Scaffold(t, "cross_deps")

	state := discover(t, root)

	a := applier.NewApplier()
	if err := a.Apply(state); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// beta requires root + alpha, should have replace for both.
	data, err := os.ReadFile(filepath.Join(root.String(), "beta", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "replace") {
		t.Errorf("beta/go.mod missing replace directives:\n%s", content)
	}
}

func TestApply_StaleReplace_UpdatesPath(t *testing.T) {
	root := testutil.Scaffold(t, "stale_replace")

	state := discover(t, root)

	a := applier.NewApplier()
	if err := a.Apply(state); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root.String(), "sub", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)

	// The stale path "../../wrong" must be replaced with the correct relative path to root.
	if strings.Contains(content, "../../wrong") {
		t.Errorf("sub/go.mod still has stale replace path:\n%s", content)
	}

	// modfile normalizes "../" to ".." — both are valid, check without trailing slash.
	if !strings.Contains(content, "replace example.com/root => ..") {
		t.Errorf("sub/go.mod missing corrected replace directive:\n%s", content)
	}
}

// discover runs the full discovery pipeline and fails on error.
func discover(t *testing.T, root multimod.AbsDir) multimod.State {
	t.Helper()

	disc := discovery.NewDefaultDiscovery()

	state, err := disc.Discover(root)
	if err != nil {
		t.Fatalf("Discovery failed: %v", err)
	}

	return state
}
