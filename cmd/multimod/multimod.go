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
//	multimod go <args>              — transparent proxy with multi-module awareness
//	multimod                        — show project status
package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/thumbrise/resilience/pkg/multimod"
	"github.com/thumbrise/resilience/pkg/multimod/applier"
	"github.com/thumbrise/resilience/pkg/multimod/discovery"
)

func main() {
	logger := newLogger()

	root := NewRoot()

	// Boot + Discovery + Apply: best-effort. Failure is not fatal — just no commands.
	state, banner := bootstrap(logger)
	root.Long = banner

	// Register commands only if State is available.
	if state != nil {
		for _, cmd := range NewCommands(*state, logger) {
			root.AddCommand(cmd)
		}
	}

	if err := root.Execute(); err != nil {
		logger.Error("fatal", slog.Any("error", err))
		os.Exit(1)
	}
}

const toolHeader = "multimod — Zero-config multi-module management tool"

// warnBanner returns nil State and a banner with a warning message.
func warnBanner(msg string) (*multimod.State, string) {
	return nil, fmt.Sprintf("%s\n\n⚠ %s", toolHeader, msg)
}

// bootstrap runs Boot + Discovery + Apply and returns State + banner string.
// Apply is always called — any multimod interaction guarantees synced FS.
// Never fails fatally — returns nil State and a warning banner instead.
func bootstrap(logger *slog.Logger) (*multimod.State, string) {
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
func buildBanner(state multimod.State) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "%s\n\n✓ %s (go %s)\n", toolHeader, state.Root.Path, state.Root.GoVersion)

	for _, sub := range state.Subs {
		rel, err := state.Root.Dir.Rel(sub.Dir)
		if err != nil {
			rel = sub.Dir.String()
		}

		deps := ""
		if len(sub.Replaces) > 0 {
			deps = fmt.Sprintf(" → replaces %d internal", len(sub.Replaces))
		}

		fmt.Fprintf(&sb, "  %s (%s)%s\n", sub.Path, rel, deps)
	}

	fmt.Fprintf(&sb, "  Workspace: %d modules", len(state.Workspace))

	return sb.String()
}

// newLogger creates a slog.Logger writing to stderr with [multimod] component tag.
func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil)).With("component", "multimod")
}
