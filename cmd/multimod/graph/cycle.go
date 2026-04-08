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

// Package graph provides pure graph algorithms.
// No domain knowledge, no external dependencies.
package graph

import "sort"

// DetectCycle finds a cycle in a directed graph using DFS with white/gray/black coloring.
// Returns the cycle as a path (e.g. ["a", "b", "c", "a"]) or nil if acyclic.
// Adjacency maps each node to its direct successors.
// Deterministic: nodes are visited in sorted order for reproducible results.
func DetectCycle(adjacency map[string][]string) []string {
	const (
		white = 0
		gray  = 1
		black = 2
	)

	color := make(map[string]int)
	parent := make(map[string]string)

	var dfs func(node string) []string

	dfs = func(node string) []string {
		color[node] = gray

		for _, dep := range adjacency[node] {
			if color[dep] == gray {
				return reconstructCycle(dep, node, parent)
			}

			if color[dep] == white {
				parent[dep] = node

				if cycle := dfs(dep); cycle != nil {
					return cycle
				}
			}
		}

		color[node] = black

		return nil
	}

	nodes := make([]string, 0, len(adjacency))
	for node := range adjacency {
		nodes = append(nodes, node)
	}

	sort.Strings(nodes)

	for _, node := range nodes {
		if color[node] == white {
			if cycle := dfs(node); cycle != nil {
				return cycle
			}
		}
	}

	return nil
}

// reconstructCycle builds the cycle path from DFS parent tracking.
// Returns path in natural order: [start, ..., end, start].
func reconstructCycle(start, end string, parent map[string]string) []string {
	cycle := []string{start, end}

	for cur := end; cur != start; {
		cur = parent[cur]
		cycle = append(cycle, cur)
	}

	for i, j := 0, len(cycle)-1; i < j; i, j = i+1, j-1 {
		cycle[i], cycle[j] = cycle[j], cycle[i]
	}

	return cycle
}
