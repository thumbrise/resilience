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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thumbrise/resilience/pkg/multimod"
)

// ErrNoGoMod is returned when no go.mod exists in the current directory.
var ErrNoGoMod = errors.New("no go.mod in current directory — run multimod from the project root")

// Boot holds the result of project root discovery.
type Boot struct {
	// RootDir is the absolute path to the project root (cwd).
	RootDir multimod.AbsDir

	// MultiModule is true if sub-directories contain their own go.mod files.
	MultiModule bool

	// NoGit is true when no .git directory exists in the project root.
	// Not an error — but worth a warning (CI misconfiguration, shallow clone, etc.).
	NoGit bool
}

// Bootloader determines if cwd is a multi-module project root.
// No directory traversal — cwd is the root. Like goreleaser, terraform.
type Bootloader struct{}

// NewBootloader creates a Bootloader.
func NewBootloader() Bootloader {
	return Bootloader{}
}

// Boot checks that cwd has a go.mod and determines whether it's multi-module.
// cwd = project root. No upward search. Run multimod from the root directory.
func (b Bootloader) Boot() (Boot, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Boot{}, fmt.Errorf("getting working directory: %w", err)
	}

	rootDir, err := multimod.NewAbsDir(cwd)
	if err != nil {
		return Boot{}, fmt.Errorf("resolving absolute path: %w", err)
	}

	if _, err := os.Stat(filepath.Join(rootDir.String(), multimod.ModuleMarker)); err != nil {
		return Boot{}, ErrNoGoMod
	}

	multi, err := b.hasSubModules(rootDir)
	if err != nil {
		return Boot{}, fmt.Errorf("scanning for sub-modules: %w", err)
	}

	noGit := !b.hasGitDir(rootDir)

	return Boot{RootDir: rootDir, MultiModule: multi, NoGit: noGit}, nil
}

// hasGitDir checks whether a .git directory exists in rootDir.
func (b Bootloader) hasGitDir(rootDir multimod.AbsDir) bool {
	info, err := os.Stat(filepath.Join(rootDir.String(), ".git"))

	return err == nil && info.IsDir()
}

// hasSubModules checks whether any subdirectory of rootDir contains a go.mod file.
func (b Bootloader) hasSubModules(rootDir multimod.AbsDir) (bool, error) {
	found := false

	err := filepath.WalkDir(rootDir.String(), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// micro-optimization: hoist filter before WalkDir for large monorepos
			if multimod.NewDefaultDirFilter().ShouldSkip(d.Name()) {
				return filepath.SkipDir
			}

			return nil
		}

		if d.Name() == multimod.ModuleMarker && filepath.Dir(path) != rootDir.String() {
			found = true

			return filepath.SkipAll
		}

		return nil
	})
	if err != nil {
		return false, err
	}

	return found, nil
}
