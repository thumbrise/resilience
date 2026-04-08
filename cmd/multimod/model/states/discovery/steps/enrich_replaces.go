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

package steps

import (
	"github.com/thumbrise/resilience/cmd/multimod/model"
)

// EnrichReplaces sets Replaces of every sub-module to all other modules in the project.
// Unconditional: every sub gets replace for root + every other sub.
// This guarantees that `go mod tidy` never fetches internal modules from registry —
// replace is already in place before require appears.
// Unused replaces (no matching require) are harmless — Go ignores them.
func EnrichReplaces(state model.State) (model.State, error) {
	// Collect all module paths except self.
	allPaths := make([]string, 0, 1+len(state.Subs))
	allPaths = append(allPaths, state.Root.Path)

	for _, sub := range state.Subs {
		allPaths = append(allPaths, sub.Path)
	}

	subs := make([]model.Module, len(state.Subs))
	copy(subs, state.Subs)

	for i := range subs {
		var replaces []string

		for _, path := range allPaths {
			if path != subs[i].Path {
				replaces = append(replaces, path)
			}
		}

		subs[i].Replaces = replaces
	}

	state.Subs = subs

	return state, nil
}
