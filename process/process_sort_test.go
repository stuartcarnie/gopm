package process

import (
	"testing"

	"github.com/stuartcarnie/gopm/config"
)

func TestProcessSorter_Sort(t *testing.T) {
	processes := []*config.Program{{
		Name:      "prog-1",
		DependsOn: []string{"prog-3"},
	}, {
		Name:      "prog-2",
		DependsOn: []string{"prog-1"},
	}, {
		Name:      "prog-3",
		DependsOn: []string{"prog-4", "prog-5"},
	}, {
		Name: "prog-5",
	}, {
		Name: "prog-4",
	}, {
		Name:     "prog-6",
		Priority: 100,
	}, {
		Name:     "prog-7",
		Priority: 99,
	}}
	result := newProcessSorter().sort(processes)
	order := make(map[string]int)
	for i, p := range result {
		order[p.Name] = i
	}
	// Check that processes with dependencies are ordered after those dependencies.
	for i, p := range result {
		for _, d := range p.DependsOn {
			if order[d] >= i {
				t.Errorf("process %q is out of order (%d) with respect to %q (%d)", p.Name, i, d, order[d])
			}
		}
	}
	pri := 0
	prevIndex := 0
	for i, p := range result {
		if p.Priority == 0 {
			continue
		}
		if p.Priority < pri {
			t.Errorf("non-dependency process %q is out of order (%d) with respect to %q (%d)", p.Name, i, result[prevIndex].Name, prevIndex)
		}
		pri = p.Priority
		prevIndex = i
	}
}
