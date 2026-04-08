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

import "github.com/thumbrise/resilience/cmd/multimod/model"

// Discovery builds the desired dev-state by running a pipeline of Steps.
// Thin wrapper: seeds State from rootDir, delegates to Pipeline.Run.
type Discovery struct {
	pipeline *model.Pipeline
}

// NewDiscovery creates a Discovery with the given pipeline steps.
func NewDiscovery(steps []model.Step) *Discovery {
	return &Discovery{pipeline: model.NewPipeline(steps)}
}

// Discover runs the pipeline starting from rootDir.
// Returns the desired State or the first error encountered.
func (d *Discovery) Discover(rootDir model.AbsDir) (model.State, error) {
	seed := model.State{
		Root: model.Module{Dir: rootDir},
	}

	return d.pipeline.Run(seed)
}
