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

package cmds

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"slices"

	"github.com/spf13/cobra"

	"github.com/thumbrise/resilience/pkg/multimod"
)

// ErrModuleFailed is returned when one or more modules fail during multi-module execution.
var ErrModuleFailed = errors.New("module(s) failed")

// multiModuleCommands is the registry of go subcommands that require
// per-module iteration when used with ./... .
var multiModuleCommands = map[string]bool{
	"test":  true,
	"vet":   true,
	"build": true,
}

// multiModuleCompoundCommands is the registry of compound go subcommands
// that are always multi-module regardless of ./... .
var multiModuleCompoundCommands = map[string]bool{
	"mod tidy": true,
}

// Go is the transparent proxy to go with multi-module awareness.
// FS is already synced by bootstrap — Go just iterates modules.
type Go struct {
	*cobra.Command
	state  multimod.State
	logger *slog.Logger
}

// NewGo creates the go proxy command.
func NewGo(state multimod.State, logger *slog.Logger) *Go {
	g := &Go{state: state, logger: logger}

	g.Command = &cobra.Command{
		Use:                "go",
		Short:              "Transparent proxy to go with multi-module iteration",
		DisableFlagParsing: true,
		RunE:               g.run,
	}

	return g
}

func (g *Go) run(cmd *cobra.Command, args []string) error {
	if g.isMultiModule(args) {
		return g.runMultiModule(cmd, args)
	}

	return g.runPassthrough(args)
}

// isMultiModule checks if the go command should iterate all modules.
func (g *Go) isMultiModule(args []string) bool {
	if len(args) == 0 {
		return false
	}

	if len(args) >= 2 && multiModuleCompoundCommands[args[0]+" "+args[1]] {
		return true
	}

	return multiModuleCommands[args[0]] && slices.Contains(args, "./...")
}

// runMultiModule executes go <args> in root + every sub-module directory.
func (g *Go) runMultiModule(cmd *cobra.Command, args []string) error {
	dirs := make([]string, 0, 1+len(g.state.Subs))
	dirs = append(dirs, g.state.Root.Dir.String())

	for _, sub := range g.state.Subs {
		dirs = append(dirs, sub.Dir.String())
	}

	var failed int

	for _, dir := range dirs {
		g.logger.InfoContext(cmd.Context(), "executing",
			slog.String("command", "go "+args[0]),
			slog.String("dir", dir),
		)

		c := exec.CommandContext(cmd.Context(), "go", args...) //nolint:gosec // CLI proxy
		c.Dir = dir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		if err := c.Run(); err != nil {
			g.logger.ErrorContext(cmd.Context(), "module failed",
				slog.String("dir", dir),
				slog.Any("error", err),
			)

			failed++
		}
	}

	if failed > 0 {
		return fmt.Errorf("%w: %d of %d", ErrModuleFailed, failed, len(dirs))
	}

	return nil
}

// runPassthrough replaces the current process with go <args>.
func (g *Go) runPassthrough(args []string) error {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("finding go binary: %w", err)
	}

	argv := append([]string{"go"}, args...)

	return execSyscall(goBin, argv, os.Environ())
}
