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

package backoff_test

import (
	"testing"
	"time"

	"github.com/thumbrise/resilience/ecosystem/algos/backoff"
)

//nolint:funlen
func TestBackoff(t *testing.T) {
	type args struct {
		attempt    int
		base       time.Duration
		multiplier float64
	}

	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		// --- basic exponential: each next attempt doubles the wait ---
		{
			name: "attempt 0 with multiplier 2",
			args: args{attempt: 0, base: 1 * time.Second, multiplier: 2},
			want: 1 * time.Second, // 2^0 = 1
		},
		{
			name: "attempt 1 with multiplier 2",
			args: args{attempt: 1, base: 1 * time.Second, multiplier: 2},
			want: 2 * time.Second, // 2^1 = 2
		},
		{
			name: "attempt 3 with multiplier 2",
			args: args{attempt: 3, base: 1 * time.Second, multiplier: 2},
			want: 8 * time.Second, // 2^3 = 8
		},

		// --- constant backoff: multiplier = 1 ---
		{
			name: "constant backoff ignores attempt (attempt=0, multiplier=1)",
			args: args{attempt: 0, base: 5 * time.Second, multiplier: 1},
			want: 5 * time.Second,
		},
		{
			name: "constant backoff ignores attempt (attempt=10, multiplier=1)",
			args: args{attempt: 10, base: 5 * time.Second, multiplier: 1},
			want: 5 * time.Second, // 1^10 = 1
		},

		// --- fractional multiplier (less than 1, greater than 0) ---
		{
			name: "multiplier 0.5 (decay)",
			args: args{attempt: 2, base: 16 * time.Second, multiplier: 0.5},
			want: 4 * time.Second, // 16 * (0.5^2) = 16 * 0.25 = 4
		},

		// --- negative attempt clamped to 0 ---
		{
			name: "negative attempt clamped to 0",
			args: args{attempt: -5, base: 2 * time.Second, multiplier: 3},
			want: 2 * time.Second, // equivalent to attempt=0 → 3^0=1
		},

		// --- non-positive multiplier clamped to 1 ---
		{
			name: "multiplier 0 clamped to 1 (constant)",
			args: args{attempt: 3, base: 7 * time.Second, multiplier: 0},
			want: 7 * time.Second, // multiplier becomes 1
		},
		{
			name: "negative multiplier clamped to 1",
			args: args{attempt: 2, base: 3 * time.Second, multiplier: -1.5},
			want: 3 * time.Second,
		},

		// --- large values, ensure no panic ---
		{
			name: "large attempt and multiplier",
			args: args{attempt: 10, base: 1 * time.Millisecond, multiplier: 10},
			want: 10_000_000_000 * time.Millisecond, // 10^10 ms
		},

		// --- zero base ---
		{
			name: "zero base returns zero",
			args: args{attempt: 5, base: 0, multiplier: 100},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := backoff.Backoff(tt.args.attempt, tt.args.base, tt.args.multiplier)
			if got != tt.want {
				t.Errorf("Backoff() = %v, want %v", got, tt.want)
			}
		})
	}
}
