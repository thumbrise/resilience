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

// Package testutil provides shared test helpers for multimod packages.
// Not for production use.
package testutil

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/thumbrise/resilience/cmd/multimod/model"
)

// Scaffold copies testdata/<name>/ into a fresh temp directory and returns its AbsDir.
// Testdata is resolved relative to the multimod package root, not the caller.
func Scaffold(t *testing.T, name string) model.AbsDir {
	t.Helper()

	src := filepath.Join(testdataDir(), name)
	dst := t.TempDir()

	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o750)
		}

		data, err := os.ReadFile(path) //nolint:gosec // test helper, path from WalkDir
		if err != nil {
			return err
		}

		return os.WriteFile(target, data, 0o600) //nolint:gosec // test helper, paths from our own testdata
	})
	if err != nil {
		t.Fatalf("scaffold %q: %v", name, err)
	}

	abs, err := model.NewAbsDir(dst)
	if err != nil {
		t.Fatalf("scaffold abs: %v", err)
	}

	return abs
}

// testdataDir returns the absolute path to pkg/multimod/testdata/.
// Uses runtime.Caller to resolve relative to this file's location.
func testdataDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("testutil: cannot determine file path")
	}

	return filepath.Join(filepath.Dir(file), "..", "testdata")
}
