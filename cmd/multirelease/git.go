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
	"context"
	"os"
	"os/exec"
	"strings"
)

// gitRun executes a git command in the given directory.
// Output goes to stderr — stdout is reserved for pipe.
func gitRun(ctx context.Context, dir string, args ...string) error {
	c := exec.CommandContext(ctx, "git", args...) //nolint:gosec // release tool
	c.Dir = dir
	c.Stdout = os.Stderr
	c.Stderr = os.Stderr

	return c.Run()
}

// gitOutput executes a git command and returns captured stdout (trimmed).
func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	c := exec.CommandContext(ctx, "git", args...) //nolint:gosec // release tool
	c.Dir = dir

	out, err := c.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}
