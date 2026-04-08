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

package main

import (
	"fmt"
	"os"

	"golang.org/x/mod/modfile"
)

// transformGoMod strips internal replaces and pins internal requires in a single go.mod.
// Writes back only if changes were made.
func transformGoMod(gomodPath string, internalPaths map[string]bool, version string) error {
	data, err := os.ReadFile(gomodPath) //nolint:gosec // path from module map
	if err != nil {
		return fmt.Errorf("reading %s: %w", gomodPath, err)
	}

	file, err := modfile.Parse(gomodPath, data, nil)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", gomodPath, err)
	}

	changed := false

	changed = stripInternalReplaces(file, internalPaths) || changed
	changed = pinInternalRequires(file, internalPaths, version) || changed

	if !changed {
		return nil
	}

	file.Cleanup()

	out, err := file.Format()
	if err != nil {
		return fmt.Errorf("formatting %s: %w", gomodPath, err)
	}

	if err := os.WriteFile(gomodPath, out, 0o644); err != nil { //nolint:gosec // go.mod must be world-readable
		return fmt.Errorf("writing %s: %w", gomodPath, err)
	}

	return nil
}

// stripInternalReplaces removes replace directives for internal modules.
func stripInternalReplaces(file *modfile.File, internalPaths map[string]bool) bool {
	changed := false

	for _, rep := range file.Replace {
		if internalPaths[rep.Old.Path] {
			_ = file.DropReplace(rep.Old.Path, rep.Old.Version)

			changed = true
		}
	}

	return changed
}

// pinInternalRequires sets internal require versions to the release version.
func pinInternalRequires(file *modfile.File, internalPaths map[string]bool, version string) bool {
	changed := false

	for _, req := range file.Require {
		if internalPaths[req.Mod.Path] && req.Mod.Version != version {
			_ = file.AddRequire(req.Mod.Path, version)

			changed = true
		}
	}

	return changed
}
