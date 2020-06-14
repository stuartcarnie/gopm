package rpc

import (
	"fmt"
	"sort"
)

// GetFullName get the full name of program includes group and name
func (x *ProcessInfo) GetFullName() string {
	if len(x.Group) > 0 {
		return fmt.Sprintf("%s:%s", x.Group, x.Name)
	}
	return x.Name
}

type ProcessInfos []*ProcessInfo

func (pi ProcessInfos) SortByName() {
	sort.Slice(pi, func(i, j int) bool {
		return pi[i].Name < pi[j].Name
	})
}
