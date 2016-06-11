package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/fd0/probe"
)

// Alias is used to replace Name with Value.
type Alias struct {
	Name  string
	Value string
}

// NewAlias returns a new alias.
func NewAlias(name, value string) Alias {
	return Alias{Name: name, Value: value}
}

// parseAliases parses all aliases in the map and returns the list.
func parseAliases(data map[string]string) (map[string]Alias, error) {
	m := make(map[string]Alias, len(data))
	for name, value := range data {
		m[name] = NewAlias(name, value)
	}

	err := resolveAliases(m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

var aliasName = regexp.MustCompile("{{([a-zA-Z0-9_-]+)}}")

func (a Alias) deps() map[string]struct{} {
	deps := make(map[string]struct{})
	for _, match := range aliasName.FindAllStringSubmatch(a.Value, -1) {
		deps[match[1]] = struct{}{}
	}

	return deps
}

// topoSort returns a topological sorting of the nodes. See
// https://en.wikipedia.org/wiki/Topological_sorting#Kahn.27s_algorithm
//
// edges is a list of dependencies, edges[i][j] == true iff i depends on j.
func topoSort(edges [][]bool) (sorted []int, err error) {
	if len(edges) == 0 {
		return nil, nil
	}

	var startNodes []int
	var sorting []int

	// find start nodes which do not have any dependencies
nextNode:
	for i, row := range edges {
		for _, v := range row {
			if v {
				continue nextNode
			}
		}

		startNodes = append(startNodes, i)
	}

	if len(startNodes) == 0 {
		return nil, probe.Trace(errors.New("no alilses without dependencies found"))
	}

	for len(startNodes) > 0 {
		node := startNodes[0]
		startNodes = startNodes[1:]
		sorting = append(sorting, node)

		// remove dependencies on node, note candidates which may have no other dependencies
		var candidates []int
		for i := 0; i < len(edges); i++ {
			if edges[i][node] {
				edges[i][node] = false
				candidates = append(candidates, i)
			}
		}

	nextCandidate:
		for _, cand := range candidates {
			// check if there are incoming edges to this nodes
			for i := 0; i < len(edges); i++ {
				if edges[cand][i] {
					continue nextCandidate
				}
			}

			// if this node does not have any dependencies, add it to the list of start nodes
			startNodes = append(startNodes, cand)
		}
	}

	// check for remaining unresolved dependencies, then we have a circle
	for _, row := range edges {
		for _, v := range row {
			if v {
				return nil, probe.Trace(errors.New("aliases have cyclic dependencies"))
			}
		}
	}

	return sorting, nil
}

// resolveAliases replaces {{foo}} in the alias strings with the value of foo.
func resolveAliases(aliases map[string]Alias) error {

	// fix one ordering for the aliases
	list := make([]Alias, 0, len(aliases))
	for _, alias := range aliases {
		list = append(list, alias)
	}

	// index resolves an alias name to an index
	index := make(map[string]int, len(list))
	for i, alias := range list {
		index[alias.Name] = i
	}

	// graph holds all dependencies
	graph := make([][]bool, len(list))
	for i := range list {
		graph[i] = make([]bool, len(list))
	}

	for i, alias := range list {
		for d := range alias.deps() {
			j, ok := index[d]
			if !ok {
				return probe.Trace(fmt.Errorf("alias %v depends on unknown alias %v", alias.Name, d))
			}

			graph[i][j] = true
		}
	}

	sorted, err := topoSort(graph)
	if err != nil {
		return probe.Trace(err)
	}

	for _, i := range sorted {
		alias := list[i]

		value := alias.Value
		for name := range alias.deps() {
			a, ok := aliases[name]
			if !ok {
				return probe.Trace(fmt.Errorf("dependency alias %v not found", name))
			}

			value = strings.Replace(value, "{{"+name+"}}", a.Value, -1)
		}

		alias.Value = value
		aliases[alias.Name] = alias
	}

	return nil
}
