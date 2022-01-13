package config

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

var verifyDependenciesTests = []struct {
	testName string
	deps     map[string][]string
	want     [][]string
}{{
	testName: "Initial",
	deps: map[string][]string{
		"A": {"B", "C", "F"},
		"C": {"D", "E"},
		"E": {"F"},
	},
	want: [][]string{
		{"B", "D", "F"},
		{"E"},
		{"C"},
		{"A"},
	},
}, {
	testName: "Independent",
	deps: map[string][]string{
		"A": {},
		"B": {},
		"C": {},
		"D": {},
	},
	want: [][]string{{"A", "B", "C", "D"}},
}, {
	testName: "WithJoin",
	deps: map[string][]string{
		"A": {"B", "C", "G"},
		"B": {"D"},
		"C": {"D", "F"},
		"D": {"E", "F"},
	},
	want: [][]string{
		{"E", "F", "G"},
		{"D"},
		{"B", "C"},
		{"A"},
	},
}}

func TestVerifyDependencies(t *testing.T) {
	for _, test := range verifyDependenciesTests {
		t.Run(test.testName, func(t *testing.T) {
			cfg := &Config{
				Programs: make(map[string]*Program),
			}
			for name, deps := range test.deps {
				depsMap := make(map[string]bool)
				for _, dep := range deps {
					depsMap[dep] = true
				}
				p := cfg.Programs[name]
				if p == nil {
					cfg.Programs[name] = &Program{
						Name:      name,
						DependsOn: depsMap,
					}
				} else {
					p.DependsOn = depsMap
				}
				// Make sure all the dependencies have nodes too,
				// so we don't have to specify terminal nodes in each test.
				for _, dep := range deps {
					if cfg.Programs[dep] == nil {
						cfg.Programs[dep] = &Program{
							Name: dep,
						}
					}
				}
			}
			err := cfg.verifyDependencies()
			qt.Assert(t, err, qt.IsNil)
			result := make([][]string, len(cfg.TopoSortedPrograms))
			for i, progs := range cfg.TopoSortedPrograms {
				result[i] = make([]string, len(progs))
				for j, p := range progs {
					result[i][j] = p.Name
				}
			}
			qt.Assert(t, result, qt.DeepEquals, test.want)
		})
	}
}
