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

package discovery

import "github.com/thumbrise/resilience/pkg/multimod"

// Discovery builds the desired State by running a pipeline of Steps.
// Returns validated, enriched State or error.
// modfile and FS knowledge stays inside Steps — Discovery is a pure pipeline runner.
type Discovery struct {
	steps []Step
}

// NewDiscovery creates a Discovery with the given pipeline steps.
// Panics if steps is empty — programmer error.
func NewDiscovery(steps []Step) *Discovery {
	if len(steps) == 0 {
		panic("multimod: NewDiscovery requires at least one Step")
	}

	return &Discovery{steps: steps}
}

// Discover runs the pipeline starting from rootDir.
// Returns the desired State or the first error encountered.
func (d *Discovery) Discover(rootDir multimod.AbsDir) (multimod.State, error) {
	state := multimod.State{
		Root: multimod.Module{Dir: rootDir},
	}

	var err error

	for _, step := range d.steps {
		state, err = step(state)
		if err != nil {
			return multimod.State{}, err
		}
	}

	return state, nil
}
