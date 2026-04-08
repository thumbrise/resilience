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

package release_test

import (
	"log/slog"
	"slices"
	"testing"

	"github.com/thumbrise/resilience/cmd/multimod/model"
	"github.com/thumbrise/resilience/cmd/multimod/release"
)

func TestDryRun_UnderscorePrefixNotTagged(t *testing.T) {
	state := model.State{
		Root: model.Module{
			Path: "example.com/root",
			Dir:  "/project",
		},
		Subs: []model.Module{
			{
				Path: "example.com/root/otel",
				Dir:  "/project/otel",
			},
			{
				Path: "example.com/root/tools",
				Dir:  "/project/_tools",
			},
		},
	}

	r := release.NewReleaser(state, slog.Default())
	plan := r.DryRun("v1.2.3")

	// otel should be tagged.
	if !slices.Contains(plan.SubTags, "otel/v1.2.3") {
		t.Errorf("SubTags missing otel/v1.2.3: %v", plan.SubTags)
	}

	// _tools should NOT be tagged.
	for _, tag := range plan.SubTags {
		if tag == "_tools/v1.2.3" {
			t.Errorf("_tools should not be tagged, but found: %s", tag)
		}
	}

	// Both should have modified files (go.mod transform applies to all).
	if len(plan.ModifiedFiles) != 2 {
		t.Errorf("ModifiedFiles = %d, want 2 (both subs get go.mod transform)", len(plan.ModifiedFiles))
	}
}
