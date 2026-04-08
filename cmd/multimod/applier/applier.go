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

// Package applier makes the filesystem match the desired State.
// Orchestrates go.work (via internal GoWorkApplier) and go.mod (directly).
// modfile is an internal implementation detail — types do not escape this package.
package applier

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"

	"github.com/thumbrise/resilience/cmd/multimod/applier/internal/goworkapplier"
	"github.com/thumbrise/resilience/cmd/multimod/model"
)

// Applier receives desired State and makes the filesystem match it.
// Idempotent: if files already match desired state, no writes occur.
// Orchestrates go.work sync and go.mod sync (read → map desired → flush).
type Applier struct {
	goWork *goworkapplier.GoWorkApplier
}

// NewApplier creates an Applier.
func NewApplier() *Applier {
	return &Applier{
		goWork: goworkapplier.NewGoWorkApplier(),
	}
}

// Apply makes the filesystem match the desired State.
// Syncs go.work, then go.mod for each sub-module.
func (a *Applier) Apply(state model.State) error {
	if err := a.goWork.Apply(state); err != nil {
		return fmt.Errorf("sync go.work: %w", err)
	}

	for _, sub := range state.Subs {
		if err := a.syncGoMod(state, sub); err != nil {
			return fmt.Errorf("sync go.mod for %s: %w", sub.Path, err)
		}
	}

	return nil
}

// syncGoMod reads a sub-module's go.mod, maps desired State into it, writes if changed.
// Unit of Work: parse (open) → mutate → format+write (flush).
func (a *Applier) syncGoMod(state model.State, sub model.Module) error {
	gomodPath := filepath.Join(sub.Dir.String(), model.ModuleMarker)

	data, err := os.ReadFile(gomodPath) //nolint:gosec // path from model
	if err != nil {
		return fmt.Errorf("reading %s: %w", gomodPath, err)
	}

	file, err := modfile.Parse(gomodPath, data, nil)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", gomodPath, err)
	}

	// Map desired State → modfile. All mutations are unconditional — desired wins.
	changed := false

	changed = a.mapGoVersion(file, sub) || changed
	changed = a.mapReplaces(file, state, sub) || changed
	changed = a.mapRequireVersions(file, sub) || changed

	if !changed {
		return nil
	}

	return a.flushGoMod(gomodPath, file)
}

// mapGoVersion sets the go directive to the desired version.
func (a *Applier) mapGoVersion(file *modfile.File, sub model.Module) bool {
	if sub.GoVersion == "" || (file.Go != nil && file.Go.Version == sub.GoVersion) {
		return false
	}

	_ = file.AddGoStmt(sub.GoVersion)

	return true
}

// mapReplaces ensures go.mod has exactly the desired internal replace directives.
// Preserves external replaces. Drops unwanted/stale internal, adds missing.
func (a *Applier) mapReplaces(file *modfile.File, state model.State, sub model.Module) bool {
	desired := a.buildDesiredReplaces(state, sub)

	changed := a.dropUnwantedReplaces(file, desired, state.InternalPaths())
	changed = a.addMissingReplaces(file, desired) || changed

	return changed
}

// mapRequireVersions pins require versions for internal modules.
func (a *Applier) mapRequireVersions(file *modfile.File, sub model.Module) bool {
	if len(sub.RequireVersions) == 0 {
		return false
	}

	changed := false

	for _, req := range file.Require {
		desired, ok := sub.RequireVersions[req.Mod.Path]
		if !ok || req.Mod.Version == desired {
			continue
		}

		_ = file.AddRequire(req.Mod.Path, desired)

		changed = true
	}

	return changed
}

// buildDesiredReplaces computes the desired replace map: module path → relative dir.
func (a *Applier) buildDesiredReplaces(state model.State, sub model.Module) map[string]string {
	desired := make(map[string]string, len(sub.Replaces))

	for _, depPath := range sub.Replaces {
		depDir := state.ModuleDir(depPath)
		if depDir == "" {
			continue
		}

		rel, err := sub.Dir.Rel(depDir)
		if err != nil {
			continue
		}

		desired[depPath] = rel
	}

	return desired
}

// dropUnwantedReplaces removes internal replaces not in desired or with stale paths.
func (a *Applier) dropUnwantedReplaces(file *modfile.File, desired map[string]string, internalPaths map[string]bool) bool {
	type entry struct {
		path    string
		version string
	}

	var toDrop []entry

	for _, rep := range file.Replace {
		if !internalPaths[rep.Old.Path] {
			continue
		}

		desiredRel, ok := desired[rep.Old.Path]
		if !ok || filepath.Clean(rep.New.Path) != desiredRel {
			toDrop = append(toDrop, entry{rep.Old.Path, rep.Old.Version})
		}
	}

	for _, e := range toDrop {
		_ = file.DropReplace(e.path, e.version)
	}

	return len(toDrop) > 0
}

// addMissingReplaces adds desired replaces not yet present in the file.
func (a *Applier) addMissingReplaces(file *modfile.File, desired map[string]string) bool {
	existing := make(map[string]bool, len(file.Replace))
	for _, rep := range file.Replace {
		existing[rep.Old.Path] = true
	}

	changed := false

	for depPath, rel := range desired {
		if !existing[depPath] {
			_ = file.AddReplace(depPath, "", rel, "")

			changed = true
		}
	}

	return changed
}

// flushGoMod formats and writes a go.mod file.
func (a *Applier) flushGoMod(path string, file *modfile.File) error {
	file.Cleanup()

	data, err := file.Format()
	if err != nil {
		return fmt.Errorf("formatting %s: %w", path, err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil { //nolint:gosec // go.mod must be world-readable
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}
