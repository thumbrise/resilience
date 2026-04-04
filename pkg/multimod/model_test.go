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

package multimod_test

import (
	"testing"

	"github.com/thumbrise/resilience/pkg/multimod"
)

func TestAbsDir_Join(t *testing.T) {
	dir := multimod.AbsDir("/root/project")

	got := dir.Join("sub", "go.mod")
	want := "/root/project/sub/go.mod"

	if got != want {
		t.Errorf("Join = %q, want %q", got, want)
	}
}

func TestAbsDir_Rel(t *testing.T) {
	from := multimod.AbsDir("/root/project/sub")
	to := multimod.AbsDir("/root/project")

	got, err := from.Rel(to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != ".." {
		t.Errorf("Rel = %q, want %q", got, "..")
	}
}

func TestDirFilter_ShouldSkip(t *testing.T) {
	f := multimod.NewDefaultDirFilter()

	tests := []struct {
		name string
		want bool
	}{
		{"vendor", true},
		{"testdata", true},
		{"_tools", true},
		{".git", true},
		{"src", false},
		{"pkg", false},
	}

	for _, tt := range tests {
		if got := f.ShouldSkip(tt.name); got != tt.want {
			t.Errorf("ShouldSkip(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
