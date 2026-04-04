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
// modfile is used internally to parse and write go.mod/go.work files.
// modfile types do not escape this package.
package applier

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"

	"github.com/thumbrise/resilience/pkg/multimod"
)

// Applier receives desired State and makes the filesystem match it.
// Idempotent: if files already match desired state, no writes occur
// (both go.work and go.mod are compared before writing).
// modfile is an internal implementation detail — it does not escape.
type Applier struct{}

// NewApplier creates an Applier.
func NewApplier() *Applier {
	return &Applier{}
}

// Apply makes the filesystem match the desired State.
// Syncs go.work, then go.mod replaces and go versions for each sub-module.
func (a *Applier) Apply(state multimod.State) error {
	if err := a.syncGoWork(state); err != nil {
		return fmt.Errorf("sync go.work: %w", err)
	}

	for _, sub := range state.Subs {
		if err := a.syncGoMod(state, sub); err != nil {
			return fmt.Errorf("sync go.mod for %s: %w", sub.Path, err)
		}
	}

	return nil
}

// syncGoWork generates go.work from desired state and writes it only if changed.
// Compares desired content with existing file to avoid unnecessary writes.
func (a *Applier) syncGoWork(state multimod.State) error {
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

	goworkPath := filepath.Join(state.Root.Dir.String(), multimod.WorkspaceMarker)

	existingData, err := os.ReadFile(goworkPath) //nolint:gosec // path from model
	if err == nil && string(existingData) == string(desiredData) {
		return nil
	}

	if err := os.WriteFile(goworkPath, desiredData, 0o644); err != nil { //nolint:gosec // go.work must be world-readable
		return fmt.Errorf("writing %s: %w", goworkPath, err)
	}

	return nil
}

// syncGoMod reads the sub-module's go.mod, syncs go version and replaces,
// and writes back only if changes were made.
func (a *Applier) syncGoMod(state multimod.State, sub multimod.Module) error {
	gomodPath := filepath.Join(sub.Dir.String(), multimod.ModuleMarker)

	data, err := os.ReadFile(gomodPath) //nolint:gosec // path from model, not user input
	if err != nil {
		return fmt.Errorf("reading %s: %w", gomodPath, err)
	}

	file, err := modfile.Parse(gomodPath, data, nil)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", gomodPath, err)
	}

	changed := false

	// Sync go version.
	if sub.GoVersion != "" && (file.Go == nil || file.Go.Version != sub.GoVersion) {
		if err := file.AddGoStmt(sub.GoVersion); err != nil {
			return fmt.Errorf("setting go directive: %w", err)
		}

		changed = true
	}

	// Sync replaces: desired replaces = sub.Replaces.
	changed = a.syncReplaces(file, state, sub) || changed

	if !changed {
		return nil
	}

	return a.writeGoMod(gomodPath, file)
}

// syncReplaces ensures the go.mod has exactly the desired replace directives
// for internal modules. Adds missing, removes extra, fixes stale paths.
// Returns true if changed.
func (a *Applier) syncReplaces(file *modfile.File, state multimod.State, sub multimod.Module) bool {
	desired := a.buildDesiredReplaces(state, sub)

	changed := a.dropUnwantedReplaces(file, state, desired)
	changed = a.addMissingReplaces(file, desired) || changed

	return changed
}

// buildDesiredReplaces computes the desired replace map: module path → relative dir.
func (a *Applier) buildDesiredReplaces(state multimod.State, sub multimod.Module) map[string]string {
	desired := make(map[string]string, len(sub.Replaces))

	for _, depPath := range sub.Replaces {
		depDir := a.findModuleDir(state, depPath)
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

// dropUnwantedReplaces removes internal replaces that are extra or have stale paths.
// Stale replaces are dropped here and re-added by addMissingReplaces.
func (a *Applier) dropUnwantedReplaces(file *modfile.File, state multimod.State, desired map[string]string) bool {
	internalPaths := a.internalPaths(state)
	changed := false

	for _, rep := range file.Replace {
		if !internalPaths[rep.Old.Path] {
			continue
		}

		desiredRel, ok := desired[rep.Old.Path]

		// filepath.Clean normalizes trailing slashes: modfile preserves "../" from go.mod,
		// but AbsDir.Rel returns ".." — both mean the same directory.
		if !ok || filepath.Clean(rep.New.Path) != desiredRel {
			_ = file.DropReplace(rep.Old.Path, rep.Old.Version)

			changed = true
		}
	}

	return changed
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

// findModuleDir returns the AbsDir for a module path in the state.
func (a *Applier) findModuleDir(state multimod.State, modPath string) multimod.AbsDir {
	if modPath == state.Root.Path {
		return state.Root.Dir
	}

	for _, sub := range state.Subs {
		if sub.Path == modPath {
			return sub.Dir
		}
	}

	return ""
}

// internalPaths returns a set of all module paths in the project.
func (a *Applier) internalPaths(state multimod.State) map[string]bool {
	paths := make(map[string]bool, 1+len(state.Subs))
	paths[state.Root.Path] = true

	for _, sub := range state.Subs {
		paths[sub.Path] = true
	}

	return paths
}

// writeGoMod formats and writes a go.mod file.
func (a *Applier) writeGoMod(path string, file *modfile.File) error {
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
