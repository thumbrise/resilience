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

// multirelease creates a detached publish-state commit with tags for a Go multi-module project.
//
// Reads module map from stdin (JSON from `multimod modules`), transforms go.mod files,
// creates detached commit + tags. Zero knowledge of multimod internals.
//
// Usage:
//
//	multimod modules | multirelease v1.2.3                   — dry-run
//	multimod modules | multirelease v1.2.3 --write           — local commit + tags
//	multimod modules | multirelease v1.2.3 --write --push    — commit + tags + push
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil)).With("component", "multirelease")

	root := newRootCmd(logger)

	if err := root.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

// newRootCmd creates the cobra root command.
func newRootCmd(logger *slog.Logger) *cobra.Command {
	var write, push bool

	cmd := &cobra.Command{
		Use:   "multirelease <version>",
		Short: "Create a detached publish-state commit with tags for a Go multi-module project",
		Long: `Reads module map from stdin (JSON from multimod modules), transforms go.mod files,
creates detached commit + tags. Zero knowledge of multimod internals.

  multimod modules | multirelease v1.2.3                   — dry-run
  multimod modules | multirelease v1.2.3 --write           — local commit + tags
  multimod modules | multirelease v1.2.3 --write --push    — commit + tags + push`,
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			if push && !write {
				return errors.New("--push requires --write")
			}

			return requireStdin()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), args[0], write, push, os.Stdin, cmd.OutOrStdout(), logger)
		},
	}

	cmd.Flags().BoolVar(&write, "write", false, "Create detached commit and tags (default: dry-run)")
	cmd.Flags().BoolVar(&push, "push", false, "Push tags to origin (requires --write)")

	return cmd
}

// requireStdin checks that stdin is a pipe, not a terminal.
// Fails fast with a clear message instead of blocking on io.ReadAll.
func requireStdin() error {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return fmt.Errorf("checking stdin: %w", err)
	}

	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return errors.New("no input on stdin — pipe module map from `multimod modules`")
	}

	return nil
}
