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
	"io"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/thumbrise/resilience/cmd/multimod/model"
	"github.com/thumbrise/resilience/cmd/multimod/release"
)

// ErrPushRequiresWrite is returned when --push is used without --write.
var ErrPushRequiresWrite = errors.New("--push requires --write")

// ReleaseCommand transforms dev-state → publish-state.
// Thin command layer — all logic lives in Releaser.
type ReleaseCommand struct {
	*cobra.Command
	releaser *release.Releaser
}

// NewReleaseCommand creates the release command.
func NewReleaseCommand(state model.State, logger *slog.Logger) *ReleaseCommand {
	rel := release.NewReleaser(state, logger)
	rc := &ReleaseCommand{releaser: rel}

	rc.Command = &cobra.Command{
		Use:   "release <version>",
		Short: "Prepare a multi-module release (detached commit + tags)",
		Long: `Transforms dev-state go.mod files into publish-state.

Three levels of trust:
  (no flags)     Dry-run: show plan, touch nothing
  --write        Local: detached commit + tags on your machine
  --write --push CI: commit + tags + push to origin`,
		Args: cobra.ExactArgs(1),
		RunE: rc.run,
	}

	rc.Command.Flags().Bool("write", false, "Create detached commit and tags (default: dry-run)")
	rc.Command.Flags().Bool("push", false, "Push tags to origin (requires --write)")

	return rc
}

func (rc *ReleaseCommand) run(cmd *cobra.Command, args []string) error {
	version := args[0]

	write, _ := cmd.Flags().GetBool("write")
	push, _ := cmd.Flags().GetBool("push")

	if push && !write {
		return ErrPushRequiresWrite
	}

	if !write {
		return rc.dryRun(cmd.OutOrStdout(), version)
	}

	return rc.releaser.Execute(cmd.Context(), version, push)
}

func (rc *ReleaseCommand) dryRun(w io.Writer, version string) error {
	plan := rc.releaser.DryRun(version)

	_, _ = fmt.Fprintf(w, "Release plan for %s\n\n", plan.Version)
	_, _ = fmt.Fprintf(w, "  Dev tag:     %s\n", plan.DevTag)
	_, _ = fmt.Fprintf(w, "  Release tag: %s\n", plan.ReleaseTag)

	for _, subTag := range plan.SubTags {
		_, _ = fmt.Fprintf(w, "  Sub tag:     %s\n", subTag)
	}

	_, _ = fmt.Fprintf(w, "  Commit:      %s\n", plan.CommitMessage)
	_, _ = fmt.Fprintf(w, "\nModified files:\n")

	for _, f := range plan.ModifiedFiles {
		_, _ = fmt.Fprintf(w, "  %s\n", f)
	}

	_, _ = fmt.Fprintf(w, "\nDry-run complete. Use --write to execute.\n")

	return nil
}
