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

// Package gitcli provides the real git command execution via os/exec.
// Internal to release — not visible outside.
package gitcli

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/thumbrise/resilience/cmd/multimod/model"
)

// Runner executes real git commands via os/exec.
// Implements release.GitRunner with scoped operations.
type Runner struct {
	dir model.AbsDir
}

// NewRunner creates a Runner that runs git in the given directory.
func NewRunner(dir model.AbsDir) *Runner {
	return &Runner{dir: dir}
}

// CurrentBranch returns the name of the current git branch.
func (r *Runner) CurrentBranch(ctx context.Context) (string, error) {
	return r.output(ctx, "rev-parse", "--abbrev-ref", "HEAD")
}

// Tag creates a lightweight tag at HEAD.
func (r *Runner) Tag(ctx context.Context, name string) error {
	return r.run(ctx, "tag", name)
}

// CheckoutDetach detaches HEAD from the current branch.
func (r *Runner) CheckoutDetach(ctx context.Context) error {
	return r.run(ctx, "checkout", "--detach")
}

// Checkout switches to the given branch.
func (r *Runner) Checkout(ctx context.Context, branch string) error {
	return r.run(ctx, "checkout", branch)
}

// StageAll stages all changes (git add -A).
func (r *Runner) StageAll(ctx context.Context) error {
	return r.run(ctx, "add", "-A")
}

// Commit creates a commit with the given message.
func (r *Runner) Commit(ctx context.Context, message string) error {
	return r.run(ctx, "commit", "-m", message)
}

// PushTags pushes all tags to origin.
func (r *Runner) PushTags(ctx context.Context) error {
	return r.run(ctx, "push", "origin", "--tags")
}

// run executes a git command and inherits stdout/stderr.
func (r *Runner) run(ctx context.Context, args ...string) error {
	c := exec.CommandContext(ctx, "git", args...) //nolint:gosec // release tool
	c.Dir = r.dir.String()
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return c.Run()
}

// output executes a git command and returns captured stdout.
func (r *Runner) output(ctx context.Context, args ...string) (string, error) {
	c := exec.CommandContext(ctx, "git", args...) //nolint:gosec // release tool
	c.Dir = r.dir.String()

	out, err := c.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}
