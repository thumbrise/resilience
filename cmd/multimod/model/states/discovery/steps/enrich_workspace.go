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

// EnrichWorkspace sets Workspace to root + all subs.
// Desired state: go.work contains every module in the project.
func EnrichWorkspace(state model.State) (model.State, error) {
	workspace := make([]model.Module, 0, 1+len(state.Subs))
	workspace = append(workspace, state.Root)
	workspace = append(workspace, state.Subs...)

	state.Workspace = workspace

	return state, nil
}
