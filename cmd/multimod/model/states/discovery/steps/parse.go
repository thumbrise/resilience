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

// Package steps contains individual discovery pipeline steps.
// Each step is a pure function: State → (State, error).
package steps

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/mod/modfile"

	"github.com/thumbrise/resilience/cmd/multimod/model"
)

// ErrNoRootGoMod is returned when no go.mod is found in the root directory.
var ErrNoRootGoMod = errors.New("no root go.mod")

// ErrRootDirNotSet is returned when Parse is called with an empty Root.Dir.
var ErrRootDirNotSet = errors.New("parse: root dir not set")

// ErrNoModuleDirective is returned when a go.mod file has no module directive.
var ErrNoModuleDirective = errors.New("no module directive")

// Parse scans the filesystem from State.Root.Dir, finds all go.mod files,
// parses each via modfile.Parse, and populates State.Root and State.Subs.
// modfile types do not escape this function — only domain Module is returned.
func Parse(state model.State) (model.State, error) {
	rootDir := state.Root.Dir
	if rootDir == "" {
		return state, ErrRootDirNotSet
	}

	modules, err := findAndParseModules(rootDir)
	if err != nil {
		return state, err
	}

	var root model.Module

	var subs []model.Module

	for _, mod := range modules {
		if mod.Dir == rootDir {
			root = mod
		} else {
			subs = append(subs, mod)
		}
	}

	if root.Path == "" {
		return state, fmt.Errorf("%w in %s", ErrNoRootGoMod, rootDir)
	}

	sort.Slice(subs, func(i, j int) bool {
		return subs[i].Path < subs[j].Path
	})

	state.Root = root
	state.Subs = subs

	return state, nil
}

// findAndParseModules walks rootDir, finds all go.mod files, parses each into Module.
func findAndParseModules(rootDir model.AbsDir) ([]model.Module, error) {
	var modules []model.Module

	err := filepath.WalkDir(rootDir.String(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if model.NewDefaultDirFilter().ShouldSkip(d.Name()) {
				return filepath.SkipDir
			}

			return nil
		}

		if d.Name() != model.ModuleMarker {
			return nil
		}

		mod, parseErr := parseGoMod(path)
		if parseErr != nil {
			return parseErr
		}

		modules = append(modules, mod)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning modules: %w", err)
	}

	return modules, nil
}

// parseGoMod reads and parses a single go.mod file into a domain Module.
// modfile.File does not escape — only extracted fields are returned.
func parseGoMod(gomodPath string) (model.Module, error) {
	data, err := os.ReadFile(gomodPath) //nolint:gosec // path from filepath.WalkDir
	if err != nil {
		return model.Module{}, fmt.Errorf("reading %s: %w", gomodPath, err)
	}

	file, err := modfile.Parse(gomodPath, data, nil)
	if err != nil {
		return model.Module{}, fmt.Errorf("parsing %s: %w", gomodPath, err)
	}

	if file.Module == nil {
		return model.Module{}, fmt.Errorf("parsing %s: %w", gomodPath, ErrNoModuleDirective)
	}

	var goVersion string
	if file.Go != nil {
		goVersion = file.Go.Version
	}

	requires := make([]string, 0, len(file.Require))
	for _, req := range file.Require {
		requires = append(requires, req.Mod.Path)
	}

	replaces := make([]string, 0, len(file.Replace))
	for _, rep := range file.Replace {
		replaces = append(replaces, rep.Old.Path)
	}

	return model.Module{
		Path:      file.Module.Mod.Path,
		Dir:       model.AbsDir(filepath.Dir(gomodPath)), // safe: gomodPath is absolute from WalkDir
		GoVersion: goVersion,
		Requires:  requires,
		Replaces:  replaces,
	}, nil
}
