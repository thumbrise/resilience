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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
)

// run is the core entry point. Reads module map, builds plan, executes or prints.
func run(ctx context.Context, version string, write, push bool, stdin io.Reader, stdout io.Writer, logger *slog.Logger) error {
	modules, err := readModuleMap(stdin)
	if err != nil {
		return err
	}

	p := buildPlan(modules, version)

	if !write {
		return printPlan(stdout, p)
	}

	return execute(ctx, modules, p, push, logger)
}

// readModuleMap reads and parses the module map from stdin via streaming decoder.
func readModuleMap(r io.Reader) (moduleMap, error) {
	var modules moduleMap
	if err := json.NewDecoder(r).Decode(&modules); err != nil {
		return moduleMap{}, fmt.Errorf("parsing module map from stdin: %w", err)
	}

	return modules, nil
}

// printPlan outputs the dry-run plan to stdout.
func printPlan(w io.Writer, p plan) error {
	_, _ = fmt.Fprintf(w, "Release plan for %s\n\n", p.Version)
	_, _ = fmt.Fprintf(w, "  Dev tag:     %s\n", p.DevTag)
	_, _ = fmt.Fprintf(w, "  Release tag: %s\n", p.ReleaseTag)

	for _, subTag := range p.SubTags {
		_, _ = fmt.Fprintf(w, "  Sub tag:     %s\n", subTag)
	}

	_, _ = fmt.Fprintf(w, "  Commit:      %s\n", p.CommitMessage)
	_, _ = fmt.Fprintf(w, "\nModified files:\n")

	for _, f := range p.ModifiedFiles {
		_, _ = fmt.Fprintf(w, "  %s\n", f)
	}

	_, _ = fmt.Fprintf(w, "\nDry-run complete. Use --write to execute.\n")

	return nil
}

// execute performs the full release flow: tag dev → detach → transform → commit → tag → return.
func execute(ctx context.Context, modules moduleMap, p plan, push bool, logger *slog.Logger) error {
	rootDir := modules.Root.Dir

	branch, err := gitOutput(ctx, rootDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("getting current branch: %w", err)
	}

	// 1. Tag current HEAD for traceability.
	if err := gitRun(ctx, rootDir, "tag", p.DevTag); err != nil {
		return fmt.Errorf("tagging dev: %w", err)
	}

	logger.InfoContext(ctx, "tagged dev", slog.String("tag", p.DevTag))

	// 2. Detach HEAD.
	if err := gitRun(ctx, rootDir, "checkout", "--detach"); err != nil {
		return fmt.Errorf("detaching HEAD: %w", err)
	}

	// Ensure we return to the original branch even on error.
	defer func() {
		if checkoutErr := gitRun(ctx, rootDir, "checkout", branch); checkoutErr != nil {
			logger.ErrorContext(ctx, "failed to return to branch",
				slog.String("branch", branch),
				slog.Any("error", checkoutErr),
			)
		}
	}()

	// 3. Transform go.mod files.
	internalPaths := modules.internalPaths()

	for _, f := range p.ModifiedFiles {
		if err := transformGoMod(f, internalPaths, p.Version); err != nil {
			return fmt.Errorf("transforming %s: %w", f, err)
		}
	}

	logger.InfoContext(ctx, "transformed go.mod files", slog.Int("count", len(p.ModifiedFiles)))

	// 4. Commit.
	if err := gitRun(ctx, rootDir, "add", "-A"); err != nil {
		return fmt.Errorf("staging: %w", err)
	}

	if err := gitRun(ctx, rootDir, "commit", "-m", p.CommitMessage); err != nil {
		return fmt.Errorf("committing: %w", err)
	}

	logger.InfoContext(ctx, "committed release", slog.String("message", p.CommitMessage))

	// 5. Tag detached commit.
	if err := gitRun(ctx, rootDir, "tag", p.ReleaseTag); err != nil {
		return fmt.Errorf("tagging release: %w", err)
	}

	for _, subTag := range p.SubTags {
		if err := gitRun(ctx, rootDir, "tag", subTag); err != nil {
			return fmt.Errorf("tagging %s: %w", subTag, err)
		}
	}

	logger.InfoContext(ctx, "tagged release",
		slog.String("root", p.ReleaseTag),
		slog.Int("subs", len(p.SubTags)),
	)

	// 6. Return to branch (handled by defer).

	// 7. Push if requested.
	if push {
		if err := gitRun(ctx, rootDir, "push", "origin", "--tags"); err != nil {
			return fmt.Errorf("pushing tags: %w", err)
		}

		logger.InfoContext(ctx, "pushed tags")
	}

	return nil
}
