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

// PinRequires returns a Step that pins internal require versions to the given version.
// Publish-state: users get go.mod with correct version tags, not v0.0.0.
func PinRequires(version string) model.Step {
	return func(state model.State) (model.State, error) {
		internalPaths := state.InternalPaths()

		subs := make([]model.Module, len(state.Subs))
		copy(subs, state.Subs)

		for i, sub := range subs {
			subs[i].RequireVersions = pinVersions(sub.Requires, internalPaths, version)
		}

		state.Subs = subs

		return state, nil
	}
}

// pinVersions builds RequireVersions for internal requires.
func pinVersions(requires []string, internalPaths map[string]bool, version string) map[string]string {
	if len(requires) == 0 {
		return nil
	}

	pins := make(map[string]string)

	for _, req := range requires {
		if internalPaths[req] {
			pins[req] = version
		}
	}

	if len(pins) == 0 {
		return nil
	}

	return pins
}
