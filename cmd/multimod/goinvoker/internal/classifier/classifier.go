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

// Package classifier decides whether go args require multi-module iteration.
// Internal to goinvoker — not visible outside.
package classifier

import "slices"

// Matcher decides whether go args require multi-module iteration.
type Matcher func(args []string) bool

// Classifier answers one question: do these args need multi-module iteration?
// Holds a registry of matchers.
type Classifier struct {
	matchers []Matcher
}

// New creates a Classifier with the given matchers.
func New(matchers []Matcher) *Classifier {
	return &Classifier{matchers: matchers}
}

// NewDefault creates a Classifier with the standard matcher set.
// New pattern = new line here.
func NewDefault() *Classifier {
	return New([]Matcher{
		subcmdWithWildcard("test"),
		subcmdWithWildcard("vet"),
		subcmdWithWildcard("build"),
		compoundSubcmd("mod", "tidy"),
		toolWithWildcard(),
	})
}

// IsMultiModule returns true if any matcher considers the args multi-module.
func (c *Classifier) IsMultiModule(args []string) bool {
	for _, match := range c.matchers {
		if match(args) {
			return true
		}
	}

	return false
}

// subcmdWithWildcard matches "go <subcmd> ... ./..." — commands that need
// per-module iteration only when the recursive wildcard is present.
func subcmdWithWildcard(subcmd string) Matcher {
	return func(args []string) bool {
		return len(args) > 0 && args[0] == subcmd && slices.Contains(args, "./...")
	}
}

// compoundSubcmd matches "go <word1> <word2>" — compound commands that are
// always multi-module regardless of other arguments.
func compoundSubcmd(word1, word2 string) Matcher {
	return func(args []string) bool {
		return len(args) >= 2 && args[0] == word1 && args[1] == word2
	}
}

// toolWithWildcard matches "go tool <name> ... ./..." — tool commands that need
// per-module iteration when the recursive wildcard is present.
// Example: go tool govulncheck ./...
func toolWithWildcard() Matcher {
	return func(args []string) bool {
		return len(args) >= 3 && args[0] == "tool" && slices.Contains(args, "./...")
	}
}
