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

package model

// Step is a single unit of a state pipeline.
// Receives State by value, returns new State or error.
// Pure function contract: input State is not modified.
//
// Used by both discovery (FS → dev-state) and publish (dev-state → publish-state).
//

type Step func(State) (State, error)

// Pipeline runs Steps sequentially, threading State through each.
type Pipeline struct {
	steps []Step
}

// NewPipeline creates a Pipeline from the given steps.
// Panics if steps is empty — programmer error.
func NewPipeline(steps []Step) *Pipeline {
	if len(steps) == 0 {
		panic("multimod: NewPipeline requires at least one Step")
	}

	return &Pipeline{steps: steps}
}

// Run executes the pipeline starting from the given state.
// Returns the final State or the first error encountered.
func (p *Pipeline) Run(state State) (State, error) {
	var err error

	for _, step := range p.steps {
		state, err = step(state)
		if err != nil {
			return State{}, err
		}
	}

	return state, nil
}
