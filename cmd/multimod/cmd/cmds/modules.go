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
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thumbrise/resilience/cmd/multimod/model"
)

// ModulesCommand outputs the project module map as JSON to stdout.
// Designed for piping into external tools (multirelease, jq, scripts).
// Thin command layer — marshals State into a stable JSON contract.
type ModulesCommand struct {
	*cobra.Command
	state model.State
}

// NewModulesCommand creates the modules command.
func NewModulesCommand(state model.State) *ModulesCommand {
	mc := &ModulesCommand{state: state}

	mc.Command = &cobra.Command{
		Use:   "modules",
		Short: "Output project module map as JSON",
		Long: `Outputs discovered module structure as JSON to stdout.
Designed for piping into external tools:

  multimod modules | multirelease v1.2.3 --write
  multimod modules | jq '.subs[].dir'`,
		Args: cobra.NoArgs,
		RunE: mc.run,
	}

	return mc
}

func (mc *ModulesCommand) run(cmd *cobra.Command, _ []string) error {
	output := mc.buildOutput()

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling modules: %w", err)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))

	return nil
}

// modulesOutput is the stable JSON contract for external tools.
// Fields use json tags — this is a public API, not an internal type.
type modulesOutput struct {
	Root moduleEntry   `json:"root"`
	Subs []moduleEntry `json:"subs"`
}

// moduleEntry is a single module in the JSON output.
type moduleEntry struct {
	Path      string   `json:"path"`
	Dir       string   `json:"dir"`
	GoVersion string   `json:"go_version,omitempty"`
	Requires  []string `json:"requires,omitempty"`
}

// buildOutput converts internal State into the stable JSON contract.
// Dirs are absolute — pipe consumers don't know the caller's cwd.
func (mc *ModulesCommand) buildOutput() modulesOutput {
	root := moduleEntry{
		Path:      mc.state.Root.Path,
		Dir:       mc.state.Root.Dir.String(),
		GoVersion: mc.state.Root.GoVersion,
	}

	subs := make([]moduleEntry, 0, len(mc.state.Subs))

	for _, sub := range mc.state.Subs {
		entry := moduleEntry{
			Path: sub.Path,
			Dir:  sub.Dir.String(),
		}

		// Only include internal requires — external deps are not our concern.
		internalPaths := mc.state.InternalPaths()

		for _, req := range sub.Requires {
			if internalPaths[req] {
				entry.Requires = append(entry.Requires, req)
			}
		}

		subs = append(subs, entry)
	}

	return modulesOutput{
		Root: root,
		Subs: subs,
	}
}
