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

// Package goworkapplier syncs go.work to match the desired State.
// Simple flat package — no internal details needed.
package goworkapplier

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"

	"github.com/thumbrise/resilience/cmd/multimod/model"
)

// GoWorkApplier syncs go.work to match the desired State.
// Generates go.work from scratch, writes only if content differs.
type GoWorkApplier struct{}

// NewGoWorkApplier creates a GoWorkApplier.
func NewGoWorkApplier() *GoWorkApplier {
	return &GoWorkApplier{}
}

// Apply generates go.work from desired state and writes it only if changed.
func (a *GoWorkApplier) Apply(state model.State) error {
	if len(state.Workspace) == 0 {
		return nil
	}

	work, err := modfile.ParseWork("go.work", []byte("go "+state.Root.GoVersion+"\n"), nil)
	if err != nil {
		return fmt.Errorf("creating go.work: %w", err)
	}

	for _, mod := range state.Workspace {
		rel, err := state.Root.Dir.Rel(mod.Dir)
		if err != nil {
			return fmt.Errorf("computing relative path for %s: %w", mod.Path, err)
		}

		if err := work.AddUse(rel, ""); err != nil {
			return fmt.Errorf("adding use %s: %w", rel, err)
		}
	}

	work.SortBlocks()
	work.Cleanup()

	desiredData := modfile.Format(work.Syntax)

	goworkPath := filepath.Join(state.Root.Dir.String(), model.WorkspaceMarker)

	existingData, err := os.ReadFile(goworkPath) //nolint:gosec // path from model
	if err == nil && string(existingData) == string(desiredData) {
		return nil
	}

	if err := os.WriteFile(goworkPath, desiredData, 0o644); err != nil { //nolint:gosec // go.work must be world-readable
		return fmt.Errorf("writing %s: %w", goworkPath, err)
	}

	return nil
}
