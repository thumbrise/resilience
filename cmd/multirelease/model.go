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
	"path/filepath"
	"strings"
)

// --- Input model (JSON contract from `multimod modules`) ---

// moduleMap is the input contract. Matches `multimod modules` output exactly.
type moduleMap struct {
	Root moduleEntry   `json:"root"`
	Subs []moduleEntry `json:"subs"`
}

// moduleEntry is a single module from the module map.
type moduleEntry struct {
	Path     string   `json:"path"`
	Dir      string   `json:"dir"`
	Requires []string `json:"requires,omitempty"`
}

// internalPaths returns the set of all module paths in this project.
func (m moduleMap) internalPaths() map[string]bool {
	paths := make(map[string]bool, 1+len(m.Subs))
	paths[m.Root.Path] = true

	for _, sub := range m.Subs {
		paths[sub.Path] = true
	}

	return paths
}

// --- Plan ---

// plan describes what a release will do. Built from moduleMap + version.
type plan struct {
	Version       string
	DevTag        string
	ReleaseTag    string
	SubTags       []string
	CommitMessage string
	ModifiedFiles []string
}

// buildPlan computes the release plan from module map.
// Workspace-only modules (dir starts with "_") are transformed but not tagged.
func buildPlan(modules moduleMap, version string) plan {
	subTags := make([]string, 0, len(modules.Subs))
	modifiedFiles := make([]string, 0, len(modules.Subs))

	for _, sub := range modules.Subs {
		modifiedFiles = append(modifiedFiles, filepath.Join(sub.Dir, "go.mod"))

		rel, err := filepath.Rel(modules.Root.Dir, sub.Dir)
		if err != nil {
			rel = filepath.Base(sub.Dir)
		}

		// Workspace-only modules (e.g. _tools/) are not tagged.
		if strings.HasPrefix(filepath.Base(rel), "_") {
			continue
		}

		subTags = append(subTags, rel+"/"+version)
	}

	return plan{
		Version:       version,
		DevTag:        version + "-dev",
		ReleaseTag:    version,
		SubTags:       subTags,
		CommitMessage: fmt.Sprintf("chore(release): %s [multirelease]", version),
		ModifiedFiles: modifiedFiles,
	}
}
