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

package cmd

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/thumbrise/resilience/cmd/multimod/model"
)

// Root is the top-level multimod command.
type Root struct {
	cmd *cobra.Command
}

// NewRoot creates the root cobra command.
func NewRoot() *Root {
	return &Root{cmd: &cobra.Command{
		Use:           "multimod",
		Short:         "Zero-config multi-module management tool for Go monorepos",
		SilenceUsage:  true,
		SilenceErrors: true,
	}}
}

func (r *Root) Register(state model.State, logger *slog.Logger) {
	for _, cmd := range NewCommands(state, logger) {
		r.cmd.AddCommand(cmd)
	}
}

func (r *Root) ExecuteContext(ctx context.Context) error {
	return r.cmd.ExecuteContext(ctx)
}

func (r *Root) Warn(banner string) {
	r.cmd.Long = banner
}
