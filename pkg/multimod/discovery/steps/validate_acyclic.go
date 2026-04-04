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
	"errors"
	"fmt"
	"strings"

	"github.com/thumbrise/resilience/pkg/multimod"
	"github.com/thumbrise/resilience/pkg/multimod/graph"
)

// ErrCyclicDependency is returned when the internal dependency graph has a cycle.
var ErrCyclicDependency = errors.New("cyclic dependency")

// ErrRootRequiresSub is returned when the root module requires an internal sub-module.
// Root must be the zero-deps core — subs depend on root, not reverse.
var ErrRootRequiresSub = errors.New("root module must not require internal sub-modules")

// ValidateAcyclic checks that the internal dependency graph has no cycles.
// Builds adjacency from sub-module Requires only, filtering to internal modules.
// Root is intentionally excluded from the graph: in a well-structured monorepo,
// root is the core module with zero internal deps — subs depend on root, not reverse.
// Returns error if a cycle is found — desired State cannot be built.
func ValidateAcyclic(state multimod.State) (multimod.State, error) {
	internal := make(map[string]bool, 1+len(state.Subs))
	internal[state.Root.Path] = true

	for _, sub := range state.Subs {
		internal[sub.Path] = true
	}

	// Root must not require any internal sub-module.
	for _, req := range state.Root.Requires {
		if internal[req] {
			return state, fmt.Errorf(
				"%w: root requires %s — root must be the zero-deps core",
				ErrRootRequiresSub,
				req,
			)
		}
	}

	adjacency := make(map[string][]string)

	for _, sub := range state.Subs {
		var deps []string

		for _, req := range sub.Requires {
			if internal[req] && req != sub.Path {
				deps = append(deps, req)
			}
		}

		if len(deps) > 0 {
			adjacency[sub.Path] = deps
		}
	}

	cycle := graph.DetectCycle(adjacency)
	if cycle != nil {
		return state, fmt.Errorf(
			"%w: %s — extract one module into a separate repository",
			ErrCyclicDependency,
			strings.Join(cycle, " → "),
		)
	}

	return state, nil
}
