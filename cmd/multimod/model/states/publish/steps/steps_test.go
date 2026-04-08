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

package steps_test

import (
	"testing"

	"github.com/thumbrise/resilience/cmd/multimod/model"
	"github.com/thumbrise/resilience/cmd/multimod/model/states/publish/steps"
)

func TestStripReplaces_RemovesAll(t *testing.T) {
	state := devState()

	result, err := steps.StripReplaces(state)
	if err != nil {
		t.Fatalf("StripReplaces failed: %v", err)
	}

	for _, sub := range result.Subs {
		if len(sub.Replaces) != 0 {
			t.Errorf("sub %s still has replaces: %v", sub.Path, sub.Replaces)
		}
	}
}

func TestStripReplaces_DoesNotModifyInput(t *testing.T) {
	state := devState()

	_, _ = steps.StripReplaces(state)

	if len(state.Subs[0].Replaces) == 0 {
		t.Error("input state was modified — Replaces should still be populated")
	}
}

func TestPinRequires_PinsInternalRequires(t *testing.T) {
	state := devState()

	step := steps.PinRequires("v1.2.3")

	result, err := step(state)
	if err != nil {
		t.Fatalf("PinRequires failed: %v", err)
	}

	sub := result.Subs[0]

	version, ok := sub.RequireVersions["example.com/root"]
	if !ok {
		t.Fatal("RequireVersions missing example.com/root")
	}

	if version != "v1.2.3" {
		t.Errorf("got %s, want v1.2.3", version)
	}
}

func TestPinRequires_IgnoresExternalRequires(t *testing.T) {
	state := devState()
	state.Subs[0].Requires = append(state.Subs[0].Requires, "example.com/external")

	step := steps.PinRequires("v1.2.3")

	result, err := step(state)
	if err != nil {
		t.Fatalf("PinRequires failed: %v", err)
	}

	sub := result.Subs[0]

	if _, ok := sub.RequireVersions["example.com/external"]; ok {
		t.Error("external module should not be pinned")
	}
}

func TestPinRequires_DoesNotModifyInput(t *testing.T) {
	state := devState()

	step := steps.PinRequires("v1.2.3")
	_, _ = step(state)

	if state.Subs[0].RequireVersions != nil {
		t.Error("input state was modified — RequireVersions should still be nil")
	}
}

// devState returns a minimal dev-state for testing.
func devState() model.State {
	return model.State{
		Root: model.Module{
			Path: "example.com/root",
			Dir:  "/project",
		},
		Subs: []model.Module{
			{
				Path:     "example.com/root/sub",
				Dir:      "/project/sub",
				Requires: []string{"example.com/root"},
				Replaces: []string{"example.com/root"},
			},
		},
	}
}
