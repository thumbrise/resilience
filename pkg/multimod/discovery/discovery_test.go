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

package discovery_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/thumbrise/resilience/pkg/multimod/discovery"
)

func newDiscovery() *discovery.Discovery {
	return discovery.NewDefaultDiscovery()
}

const testGoVersion = "1.25.0"

func TestDiscover_ValidProject(t *testing.T) {
	root := scaffold(t, "valid_project")
	d := newDiscovery()

	state, err := d.Discover(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if state.Root.Path != "example.com/root" {
		t.Errorf("Root.Path = %q, want %q", state.Root.Path, "example.com/root")
	}

	if state.Root.GoVersion != testGoVersion {
		t.Errorf("Root.GoVersion = %q, want %q", state.Root.GoVersion, testGoVersion)
	}

	if len(state.Subs) != 1 {
		t.Fatalf("len(Subs) = %d, want 1", len(state.Subs))
	}

	sub := state.Subs[0]
	if sub.Path != "example.com/root/sub" {
		t.Errorf("Sub.Path = %q, want %q", sub.Path, "example.com/root/sub")
	}

	// Enriched: go version synced to root.
	if sub.GoVersion != testGoVersion {
		t.Errorf("Sub.GoVersion = %q, want %q (synced to root)", sub.GoVersion, testGoVersion)
	}

	// Enriched: replaces computed from internal deps.
	if !slices.Contains(sub.Replaces, "example.com/root") {
		t.Errorf("Sub.Replaces = %v, want example.com/root", sub.Replaces)
	}

	// Enriched: workspace = root + sub.
	if len(state.Workspace) != 2 {
		t.Fatalf("len(Workspace) = %d, want 2", len(state.Workspace))
	}
}

func TestDiscover_CrossDeps(t *testing.T) {
	root := scaffold(t, "cross_deps")
	d := newDiscovery()

	state, err := d.Discover(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Subs) != 2 {
		t.Fatalf("len(Subs) = %d, want 2", len(state.Subs))
	}

	// Find beta — it requires root + alpha, so replaces should contain both.
	var beta *struct{ Replaces []string }

	for _, sub := range state.Subs {
		if sub.Path == "example.com/root/beta" {
			beta = &struct{ Replaces []string }{sub.Replaces}
		}
	}

	if beta == nil {
		t.Fatal("beta not found in Subs")
	}

	if !slices.Contains(beta.Replaces, "example.com/root") {
		t.Errorf("beta.Replaces missing root: %v", beta.Replaces)
	}

	if !slices.Contains(beta.Replaces, "example.com/root/alpha") {
		t.Errorf("beta.Replaces missing alpha: %v", beta.Replaces)
	}
}

func TestDiscover_GoVersionMismatch_Enriched(t *testing.T) {
	root := scaffold(t, "go_version_mismatch")
	d := newDiscovery()

	state, err := d.Discover(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Sub had go 1.26.0, but enricher should set it to root's 1.25.0.
	for _, sub := range state.Subs {
		if sub.GoVersion != testGoVersion {
			t.Errorf("%s GoVersion = %q, want %q (enriched to root)", sub.Path, sub.GoVersion, testGoVersion)
		}
	}
}

func TestDiscover_CyclicDeps_Error(t *testing.T) {
	root := scaffold(t, "cyclic_deps")
	d := newDiscovery()

	_, err := d.Discover(root)
	if err == nil {
		t.Fatal("expected error for cyclic deps, got nil")
	}

	if !strings.Contains(err.Error(), "cyclic") {
		t.Errorf("error should mention cyclic: %v", err)
	}
}

func TestDiscover_CorruptedGoMod_Error(t *testing.T) {
	root := scaffold(t, "corrupted_gomod")
	d := newDiscovery()

	_, err := d.Discover(root)
	if err == nil {
		t.Fatal("expected error for corrupted go.mod, got nil")
	}
}

func TestDiscover_ExcludedDirs(t *testing.T) {
	root := scaffold(t, "excluded_dirs")
	d := newDiscovery()

	state, err := d.Discover(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Subs) != 0 {
		t.Errorf("len(Subs) = %d, want 0 (excluded dirs)", len(state.Subs))
	}
}

func TestDiscover_NestedSub(t *testing.T) {
	root := scaffold(t, "nested_sub")
	d := newDiscovery()

	state, err := d.Discover(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false

	for _, sub := range state.Subs {
		if sub.Path == "example.com/root/a/b" {
			found = true
		}
	}

	if !found {
		t.Error("nested sub example.com/root/a/b not found")
	}
}

func TestDiscover_EmptyGoMod_Error(t *testing.T) {
	root := scaffold(t, "empty_gomod")
	d := newDiscovery()

	_, err := d.Discover(root)
	if err == nil {
		t.Fatal("expected error for go.mod with no module directive, got nil")
	}

	if !strings.Contains(err.Error(), "no module directive") {
		t.Errorf("error should mention 'no module directive': %v", err)
	}
}

func TestDiscover_SubsDeterministicOrder(t *testing.T) {
	root := scaffold(t, "cross_deps")
	d := newDiscovery()

	state, err := d.Discover(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(state.Subs) < 2 {
		t.Fatalf("need at least 2 subs, got %d", len(state.Subs))
	}

	for i := 1; i < len(state.Subs); i++ {
		if state.Subs[i].Path < state.Subs[i-1].Path {
			t.Errorf("subs not sorted: %q before %q", state.Subs[i-1].Path, state.Subs[i].Path)
		}
	}
}
