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
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/thumbrise/resilience/cmd/multimod/cmds"
	"github.com/thumbrise/resilience/pkg/multimod"
)

// NewCommands assembles the CLI command tree from individual commands.
// FS is already synced by bootstrap — commands just use State.
// Returns root-level commands to be registered on Root.
func NewCommands(state multimod.State, logger *slog.Logger) []*cobra.Command {
	goCmd := cmds.NewGo(state, logger)

	return []*cobra.Command{
		goCmd.Command,
	}
}
