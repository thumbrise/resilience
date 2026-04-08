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

// multimod is a zero-config multi-module management tool for Go monorepos.
//
// Usage:
//
//	multimod                        — show project status
//	multimod go <args>              — transparent proxy with multi-module awareness
//	multimod release <version>      — publish preparation (detached commit + tags)
//	multimod modules                — JSON module map for piping into external tools
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	"github.com/thumbrise/resilience/cmd/multimod/applier"
	"github.com/thumbrise/resilience/cmd/multimod/cmd"
	"github.com/thumbrise/resilience/cmd/multimod/model"
	"github.com/thumbrise/resilience/cmd/multimod/model/states/discovery"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)

	logger := newLogger()

	root := cmd.NewRoot()

	// Boot + Discovery + Apply: best-effort. Failure is not fatal — just no commands.
	state, banner := bootstrap(logger)
	root.Warn(banner)

	// Register commands only if State is available.
	if state != nil {
		root.Register(*state, logger)
	}

	exitCode := 0

	if err := root.ExecuteContext(ctx); err != nil {
		logger.Error("fatal", slog.Any("error", err))

		exitCode = 1
	}

	stop()
	os.Exit(exitCode)
}

const toolHeader = "multimod — Zero-config multi-module management tool"

// warnBanner returns nil State and a banner with a warning message.
// Free function: bootstrap DSL helper, not a method — no owning type in main package.
func warnBanner(msg string) (*model.State, string) {
	return nil, fmt.Sprintf("%s\n\n⚠ %s", toolHeader, msg)
}

// bootstrap runs Boot + Discovery + Apply and returns State + banner string.
// Apply is always called — any multimod interaction guarantees synced FS.
// Never fails fatally — returns nil State and a warning banner instead.
// Free function: orchestrates multiple independent components (Boot, Discovery, Applier),
// does not belong to any single type.
func bootstrap(logger *slog.Logger) (*model.State, string) {
	boot, err := NewBootloader().Boot()
	if err != nil {
		logger.Warn("boot failed", slog.Any("error", err))

		return warnBanner(err.Error())
	}

	if boot.NoGit {
		logger.Warn("no .git directory in project root — is this a git repository?")
	}

	if !boot.MultiModule {
		return warnBanner("Not a multi-module project")
	}

	disc := discovery.NewDefaultDiscovery()

	state, err := disc.Discover(boot.RootDir)
	if err != nil {
		logger.Warn("discovery failed", slog.Any("error", err))

		return warnBanner(fmt.Sprintf("Discovery failed: %v", err))
	}

	// Apply: always sync FS to desired state. Any multimod usage guarantees consistency.
	a := applier.NewApplier()
	if err := a.Apply(state); err != nil {
		logger.Warn("apply failed", slog.Any("error", err))

		return warnBanner(fmt.Sprintf("Apply failed: %v", err))
	}

	return &state, buildBanner(state)
}

// buildBanner creates the status banner from discovered State.
// Free function: pure formatting, bootstrap DSL — no owning type in main package.
func buildBanner(state model.State) string {
	var sb strings.Builder

	_, _ = fmt.Fprintf(&sb, "%s\n\n✓ %s (go %s)\n", toolHeader, state.Root.Path, state.Root.GoVersion)

	for _, sub := range state.Subs {
		rel, err := state.Root.Dir.Rel(sub.Dir)
		if err != nil {
			rel = sub.Dir.String()
		}

		deps := ""
		if len(sub.Replaces) > 0 {
			deps = fmt.Sprintf(" → replaces %d internal", len(sub.Replaces))
		}

		_, _ = fmt.Fprintf(&sb, "  %s (%s)%s\n", sub.Path, rel, deps)
	}

	_, _ = fmt.Fprintf(&sb, "  Workspace: %d modules", len(state.Workspace))

	return sb.String()
}

// newLogger creates a slog.Logger writing to stderr with [multimod] component tag.
func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil)).With("component", "multimod")
}
