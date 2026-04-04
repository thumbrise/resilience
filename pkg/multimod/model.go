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

// Package multimod is a zero-config multi-module management tool for Go monorepos.
//
// Architecture: Discovery → desired State → Applier.Apply(state).
// Discovery reads FS and builds the desired State (validated, enriched).
// Applier receives desired State and makes the filesystem match it.
// State is the boundary — pure domain model, no infrastructure types.
package multimod

import "path/filepath"

// AbsDir is an absolute directory path. Guaranteed absolute at construction time.
// Eliminates repeated filepath.Abs calls throughout the codebase.
type AbsDir string

// NewAbsDir resolves path to absolute form. Returns error if resolution fails.
func NewAbsDir(path string) (AbsDir, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return AbsDir(abs), nil
}

// String returns the absolute path as a string.
func (d AbsDir) String() string { return string(d) }

// Join appends path elements to the directory.
func (d AbsDir) Join(elem ...string) string {
	return filepath.Join(append([]string{string(d)}, elem...)...)
}

// Rel returns the relative path from this directory to target, using forward slashes.
func (d AbsDir) Rel(target AbsDir) (string, error) {
	rel, err := filepath.Rel(string(d), string(target))
	if err != nil {
		return "", err
	}

	return filepath.ToSlash(rel), nil
}

// Module is a Go module within the project.
// Pure domain model — no modfile, no AST, no FS handles.
// Discovery populates it from go.mod. Applier uses it to write files.
type Module struct {
	// Path is the module path (e.g. "github.com/thumbrise/resilience/otel").
	Path string

	// Dir is the absolute path to the directory containing this module's go.mod.
	Dir AbsDir

	// GoVersion is the go directive value (e.g. "1.25.0"). Empty if absent.
	GoVersion string

	// Requires lists module paths this module depends on (from require directives).
	Requires []string

	// Replaces lists module paths that have replace directives in this module's go.mod.
	Replaces []string
}

// State is the desired state of a multi-module Go project.
// Built by Discovery (validated and enriched). Read-only after construction.
// Applier receives State and makes the filesystem match it.
//
// State is the architectural boundary between domain and infrastructure.
// No modfile types, no FS paths beyond AbsDir, no infrastructure concerns.
type State struct {
	// Root is the root module of the project.
	Root Module

	// Subs is all discovered sub-modules, in deterministic order.
	Subs []Module

	// Workspace lists modules that go.work must contain.
	// Always root + all subs in the current design.
	Workspace []Module
}

// ModuleMarker is the filename that identifies a Go module directory.
const ModuleMarker = "go.mod"

// WorkspaceMarker is the filename that identifies a Go workspace root.
const WorkspaceMarker = "go.work"

// defaultExcludedDirs are directory names skipped during module scanning.
var defaultExcludedDirs = []string{"vendor", "testdata"} //nolint:gochecknoglobals // package-level defaults

// defaultExcludedPrefixes are directory name prefixes skipped during scanning.
var defaultExcludedPrefixes = []string{"_", "."} //nolint:gochecknoglobals // package-level defaults

// DirFilter holds exclusion rules for directory scanning.
type DirFilter struct {
	// ExcludedDirs are directory names to skip (exact match).
	ExcludedDirs []string

	// ExcludedPrefixes are directory name prefixes to skip.
	ExcludedPrefixes []string
}

// NewDefaultDirFilter creates a DirFilter with default exclusion rules.
func NewDefaultDirFilter() DirFilter {
	return DirFilter{
		ExcludedDirs:     defaultExcludedDirs,
		ExcludedPrefixes: defaultExcludedPrefixes,
	}
}

// ShouldSkip checks if a directory name matches exclusion rules.
func (f DirFilter) ShouldSkip(name string) bool {
	for _, d := range f.ExcludedDirs {
		if name == d {
			return true
		}
	}

	for _, p := range f.ExcludedPrefixes {
		if len(name) >= len(p) && name[:len(p)] == p {
			return true
		}
	}

	return false
}
