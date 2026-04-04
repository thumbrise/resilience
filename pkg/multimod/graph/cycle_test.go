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

package graph_test

import (
	"testing"

	"github.com/thumbrise/resilience/pkg/multimod/graph"
)

func TestDetectCycle_Acyclic(t *testing.T) {
	adj := map[string][]string{
		"a": {"b"},
		"b": {"c"},
	}

	if cycle := graph.DetectCycle(adj); cycle != nil {
		t.Errorf("expected nil, got %v", cycle)
	}
}

func TestDetectCycle_DirectCycle(t *testing.T) {
	adj := map[string][]string{
		"a": {"b"},
		"b": {"a"},
	}

	cycle := graph.DetectCycle(adj)
	if cycle == nil {
		t.Fatal("expected cycle, got nil")
	}

	if len(cycle) < 3 {
		t.Errorf("cycle too short: %v", cycle)
	}

	if cycle[0] != cycle[len(cycle)-1] {
		t.Errorf("cycle should start and end with same node: %v", cycle)
	}
}

func TestDetectCycle_TriangleCycle(t *testing.T) {
	adj := map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {"a"},
	}

	cycle := graph.DetectCycle(adj)
	if cycle == nil {
		t.Fatal("expected cycle, got nil")
	}

	if cycle[0] != cycle[len(cycle)-1] {
		t.Errorf("cycle should start and end with same node: %v", cycle)
	}
}

func TestDetectCycle_Empty(t *testing.T) {
	if cycle := graph.DetectCycle(nil); cycle != nil {
		t.Errorf("expected nil for empty graph, got %v", cycle)
	}
}

func TestDetectCycle_SingleNode(t *testing.T) {
	adj := map[string][]string{
		"a": {},
	}

	if cycle := graph.DetectCycle(adj); cycle != nil {
		t.Errorf("expected nil for single node, got %v", cycle)
	}
}

func TestDetectCycle_DisconnectedWithCycle(t *testing.T) {
	adj := map[string][]string{
		"a": {"b"},
		"b": {},
		"x": {"y"},
		"y": {"x"},
	}

	cycle := graph.DetectCycle(adj)
	if cycle == nil {
		t.Fatal("expected cycle in disconnected component, got nil")
	}
}
