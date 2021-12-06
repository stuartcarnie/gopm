package process

import (
	"sort"

	"github.com/stuartcarnie/gopm/config"
)

type processByPriority []*config.Program

func (p processByPriority) Len() int {
	return len(p)
}

func (p processByPriority) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p processByPriority) Less(i, j int) bool {
	return p[i].Priority < p[j].Priority
}

// ProcessSorter sort the program by its priority
type processSorter struct {
	dependsOnGraph      map[string][]string
	procsWithoutDepends []*config.Program
}

// newProcessSorter returns a new process sorter.
func newProcessSorter() *processSorter {
	return &processSorter{
		dependsOnGraph:      make(map[string][]string),
		procsWithoutDepends: make([]*config.Program, 0),
	}
}

func (p *processSorter) initDepends(processes []*config.Program) {
	// sort by dependsOn
	for _, proc := range processes {
		if len(proc.DependsOn) == 0 {
			continue
		}
		for _, dependsOnProc := range proc.DependsOn {
			p.dependsOnGraph[proc.Name] = append(p.dependsOnGraph[proc.Name], dependsOnProc)
		}
	}
}

func (p *processSorter) initProcessWithoutDepends(processes []*config.Program) {
	dependsOnProcesses := p.getDependsOnInfo()
	for _, config := range processes {
		if _, ok := dependsOnProcesses[config.Name]; !ok {
			p.procsWithoutDepends = append(p.procsWithoutDepends, config)
		}
	}
}

func (p *processSorter) getDependsOnInfo() map[string]string {
	dependsOnProcesses := make(map[string]string)

	for k, v := range p.dependsOnGraph {
		dependsOnProcesses[k] = k
		for _, t := range v {
			dependsOnProcesses[t] = t
		}
	}

	return dependsOnProcesses
}

func (p *processSorter) sortDepends() []string {
	finishedProcesses := make(map[string]string)
	procsWithDependsInfo := p.getDependsOnInfo()
	procsStartOrder := make([]string, 0)

	// Find all processes without depends_on
	for name := range procsWithDependsInfo {
		if _, ok := p.dependsOnGraph[name]; !ok {
			finishedProcesses[name] = name
			procsStartOrder = append(procsStartOrder, name)
		}
	}

	for len(finishedProcesses) < len(procsWithDependsInfo) {
		for progName := range p.dependsOnGraph {
			if _, ok := finishedProcesses[progName]; !ok && p.inFinishedProcess(progName, finishedProcesses) {
				finishedProcesses[progName] = progName
				procsStartOrder = append(procsStartOrder, progName)
			}
		}
	}

	return procsStartOrder
}

func (p *processSorter) inFinishedProcess(processName string, finishedPrograms map[string]string) bool {
	if dependsOn, ok := p.dependsOnGraph[processName]; ok {
		for _, dependProgram := range dependsOn {
			if _, finished := finishedPrograms[dependProgram]; !finished {
				return false
			}
		}
	}
	return true
}

// sort sorts processes by dependency and returns the resulting slice.
func (p *processSorter) sort(processes []*config.Program) []*config.Program {
	p.initDepends(processes)
	p.initProcessWithoutDepends(processes)
	result := make([]*config.Program, 0)

	for _, proc := range p.sortDepends() {
		for _, config := range processes {
			if config.Name == proc {
				result = append(result, config)
			}
		}
	}

	sort.Sort(processByPriority(p.procsWithoutDepends))
	result = append(result, p.procsWithoutDepends...)
	return result
}
