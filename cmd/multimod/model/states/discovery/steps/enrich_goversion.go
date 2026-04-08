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

// EnrichGoVersion sets GoVersion of every sub-module to match Root.GoVersion.
// Desired state: all modules in the project use the same go directive.
func EnrichGoVersion(state model.State) (model.State, error) {
	if state.Root.GoVersion == "" {
		return state, nil
	}

	subs := make([]model.Module, len(state.Subs))
	copy(subs, state.Subs)

	for i := range subs {
		subs[i].GoVersion = state.Root.GoVersion
	}

	state.Subs = subs

	return state, nil
}
