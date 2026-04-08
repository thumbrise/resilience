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

// Package release orchestrates the release flow: git detach → apply publish-state → commit → tag → return.
// go.mod transformation is delegated to Applier. Git operations are delegated to GitRunner.
package release

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/thumbrise/resilience/cmd/multimod/applier"
	"github.com/thumbrise/resilience/cmd/multimod/model"
	"github.com/thumbrise/resilience/cmd/multimod/model/states/publish"
	"github.com/thumbrise/resilience/cmd/multimod/release/internal/gitcli"
)

// Plan describes what a release will do. Returned by dry-run, executed by write.
type Plan struct {
	// Version is the release version (e.g. "v1.2.3").
	Version string

	// DevTag is the traceability tag on current HEAD (e.g. "v1.2.3-dev").
	DevTag string

	// ReleaseTag is the root module tag (e.g. "v1.2.3").
	ReleaseTag string

	// SubTags are per-sub-module tags (e.g. "otel/v1.2.3").
	SubTags []string

	// CommitMessage is the release commit message.
	CommitMessage string

	// ModifiedFiles lists go.mod files that will be transformed.
	ModifiedFiles []string
}

// Releaser orchestrates the release flow: git detach → apply publish-state → commit → tag → return.
// Does not touch go.mod directly — delegates to Applier with publish-state.
// Does not execute git directly — delegates to GitRunner.
type Releaser struct {
	state   model.State
	logger  *slog.Logger
	git     GitRunner
	applier *applier.Applier
}

// NewReleaser creates a Releaser with default dependencies.
func NewReleaser(state model.State, logger *slog.Logger) *Releaser {
	return NewReleaserWith(state, logger, gitcli.NewRunner(state.Root.Dir), applier.NewApplier())
}

// NewReleaserWith creates a Releaser with explicit dependencies.
// Use in tests to inject a fake GitRunner.
func NewReleaserWith(state model.State, logger *slog.Logger, git GitRunner, a *applier.Applier) *Releaser {
	return &Releaser{state: state, logger: logger, git: git, applier: a}
}

// DryRun computes the release plan without touching the filesystem or git.
// Sub-modules whose relative directory starts with "_" are workspace-only — not tagged.
func (r *Releaser) DryRun(version string) Plan {
	subTags := make([]string, 0, len(r.state.Subs))
	modifiedFiles := make([]string, 0, len(r.state.Subs))

	for _, sub := range r.state.Subs {
		rel, err := r.state.Root.Dir.Rel(sub.Dir)
		if err != nil {
			rel = sub.Dir.String()
		}

		modifiedFiles = append(modifiedFiles, filepath.Join(sub.Dir.String(), model.ModuleMarker))

		// Workspace-only modules (e.g. _tools/) are not tagged for release.
		if strings.HasPrefix(filepath.Base(rel), "_") {
			continue
		}

		subTags = append(subTags, rel+"/"+version)
	}

	return Plan{
		Version:       version,
		DevTag:        version + "-dev",
		ReleaseTag:    version,
		SubTags:       subTags,
		CommitMessage: fmt.Sprintf("chore(release): %s [multimod]", version),
		ModifiedFiles: modifiedFiles,
	}
}

// Execute performs the full release: detached commit + tags.
// Returns to the original branch after completion.
// If push is true, pushes tags to origin.
func (r *Releaser) Execute(ctx context.Context, version string, push bool) error {
	plan := r.DryRun(version)

	branch, err := r.git.CurrentBranch(ctx)
	if err != nil {
		return fmt.Errorf("getting current branch: %w", err)
	}

	// 1. Tag current HEAD for traceability.
	if err := r.git.Tag(ctx, plan.DevTag); err != nil {
		return fmt.Errorf("tagging dev: %w", err)
	}

	r.logger.InfoContext(ctx, "tagged dev", slog.String("tag", plan.DevTag))

	// 2. Detach HEAD — main is never touched.
	if err := r.git.CheckoutDetach(ctx); err != nil {
		return fmt.Errorf("detaching HEAD: %w", err)
	}

	// Ensure we return to the original branch even on error.
	defer func() {
		if checkoutErr := r.git.Checkout(ctx, branch); checkoutErr != nil {
			r.logger.ErrorContext(ctx, "failed to return to branch",
				slog.String("branch", branch),
				slog.Any("error", checkoutErr),
			)
		}
	}()

	// 3-4. Build publish-state via pipeline, then apply to FS.
	if err := r.applyPublishState(version); err != nil {
		return err
	}

	// 5. Commit.
	if err := r.commitRelease(ctx, plan.CommitMessage); err != nil {
		return err
	}

	// 6. Tag detached commit: root + each sub.
	if err := r.tagRelease(ctx, plan); err != nil {
		return err
	}

	// 7. Return to original branch (handled by defer).

	// 8. Push if requested.
	if push {
		if err := r.git.PushTags(ctx); err != nil {
			return fmt.Errorf("pushing tags: %w", err)
		}

		r.logger.InfoContext(ctx, "pushed tags")
	}

	return nil
}

// commitRelease stages all changes and commits with the release message.
func (r *Releaser) commitRelease(ctx context.Context, message string) error {
	if err := r.git.StageAll(ctx); err != nil {
		return fmt.Errorf("staging changes: %w", err)
	}

	if err := r.git.Commit(ctx, message); err != nil {
		return fmt.Errorf("committing release: %w", err)
	}

	r.logger.InfoContext(ctx, "committed release", slog.String("message", message))

	return nil
}

// applyPublishState builds publish-state via pipeline and applies to FS.
func (r *Releaser) applyPublishState(version string) error {
	publishPipeline := model.NewPipeline(publish.NewDefaultPipeline(version))

	publishState, err := publishPipeline.Run(r.state)
	if err != nil {
		return fmt.Errorf("building publish-state: %w", err)
	}

	if err := r.applier.Apply(publishState); err != nil {
		return fmt.Errorf("applying publish-state: %w", err)
	}

	return nil
}

// tagRelease tags the detached commit: root tag + per-sub-module tags.
func (r *Releaser) tagRelease(ctx context.Context, plan Plan) error {
	if err := r.git.Tag(ctx, plan.ReleaseTag); err != nil {
		return fmt.Errorf("tagging release: %w", err)
	}

	for _, subTag := range plan.SubTags {
		if err := r.git.Tag(ctx, subTag); err != nil {
			return fmt.Errorf("tagging sub-module %s: %w", subTag, err)
		}
	}

	r.logger.InfoContext(ctx, "tagged release",
		slog.String("root", plan.ReleaseTag),
		slog.Int("subs", len(plan.SubTags)),
	)

	return nil
}
