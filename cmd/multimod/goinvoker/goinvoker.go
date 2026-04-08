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

// Package goinvoker provides the infrastructure for invoking go commands
// with multi-module awareness. It handles the decision of whether to iterate
// modules or passthrough, and executes accordingly.
package goinvoker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"

	"github.com/thumbrise/resilience/cmd/multimod/goinvoker/internal/classifier"
	"github.com/thumbrise/resilience/cmd/multimod/model"
)

// ErrModuleFailed is returned when one or more modules fail during multi-module execution.
var ErrModuleFailed = errors.New("module(s) failed")

// GoInvoker executes go commands with multi-module awareness.
// Decides whether to iterate all modules or passthrough to the system go binary.
type GoInvoker struct {
	state      model.State
	logger     *slog.Logger
	classifier *classifier.Classifier
}

// NewGoInvoker creates a GoInvoker with the default classifier.
func NewGoInvoker(state model.State, logger *slog.Logger) *GoInvoker {
	return &GoInvoker{state: state, logger: logger, classifier: classifier.NewDefault()}
}

// Run executes go with the given args. If the classifier says multi-module,
// it iterates all modules. Otherwise, it replaces the current process with go.
func (g *GoInvoker) Run(ctx context.Context, args []string) error {
	if g.classifier.IsMultiModule(args) {
		return g.iterateModules(ctx, args)
	}

	return g.passthrough(args)
}

// iterateModules executes go <args> in root + every sub-module directory.
func (g *GoInvoker) iterateModules(ctx context.Context, args []string) error {
	dirs := make([]string, 0, 1+len(g.state.Subs))
	dirs = append(dirs, g.state.Root.Dir.String())

	for _, sub := range g.state.Subs {
		dirs = append(dirs, sub.Dir.String())
	}

	var failed int

	for _, dir := range dirs {
		err := g.runInDir(ctx, args, dir)
		if err != nil {
			failed++
		}
	}

	if failed > 0 {
		return fmt.Errorf("%w: %d of %d", ErrModuleFailed, failed, len(dirs))
	}

	return nil
}

// runInDir executes go <args> in the given directory.
func (g *GoInvoker) runInDir(ctx context.Context, args []string, dir string) error {
	g.logger.InfoContext(ctx, "executing",
		slog.String("command", "go "+args[0]),
		slog.String("dir", dir),
	)

	c := exec.CommandContext(ctx, "go", args...) //nolint:gosec // CLI proxy
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	err := c.Run()
	if err != nil {
		g.logger.ErrorContext(ctx, "module failed",
			slog.String("dir", dir),
			slog.Any("error", err),
		)

		return err
	}

	return nil
}

// passthrough replaces the current process with go <args>.
func (g *GoInvoker) passthrough(args []string) error {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("finding go binary: %w", err)
	}

	argv := append([]string{"go"}, args...)

	return syscall.Exec(goBin, argv, os.Environ()) //nolint:gosec // transparent proxy
}
