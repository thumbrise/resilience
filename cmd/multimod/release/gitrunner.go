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

package release

import "context"

// GitRunner executes scoped git operations needed by the release flow.
// Interface for testability — tests inject a fake, production uses internal/gitcli.
type GitRunner interface {
	// CurrentBranch returns the name of the current git branch.
	CurrentBranch(ctx context.Context) (string, error)

	// Tag creates a lightweight tag at HEAD.
	Tag(ctx context.Context, name string) error

	// CheckoutDetach detaches HEAD from the current branch.
	CheckoutDetach(ctx context.Context) error

	// Checkout switches to the given branch.
	Checkout(ctx context.Context, branch string) error

	// StageAll stages all changes (git add -A).
	StageAll(ctx context.Context) error

	// Commit creates a commit with the given message.
	Commit(ctx context.Context, message string) error

	// PushTags pushes all tags to origin.
	PushTags(ctx context.Context) error
}
