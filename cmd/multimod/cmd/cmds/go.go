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
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/thumbrise/resilience/cmd/multimod/goinvoker"
	"github.com/thumbrise/resilience/cmd/multimod/model"
)

// GoCommand is the transparent proxy to go with multi-module awareness.
// Thin command layer — all logic lives in GoInvoker.
type GoCommand struct {
	*cobra.Command
	invoker *goinvoker.GoInvoker
}

// NewGoCommand creates the go proxy command.
func NewGoCommand(state model.State, logger *slog.Logger) *GoCommand {
	inv := goinvoker.NewGoInvoker(state, logger)

	g := &GoCommand{invoker: inv}

	g.Command = &cobra.Command{
		Use:                "go",
		Short:              "Transparent proxy to go with multi-module iteration",
		DisableFlagParsing: true,
		RunE:               g.run,
	}

	return g
}

func (g *GoCommand) run(cmd *cobra.Command, args []string) error {
	return g.invoker.Run(cmd.Context(), args)
}
