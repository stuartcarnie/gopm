package procusage

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

// stat gets the resource usage of a pid using `ps`.
func Stat(pid int) (*ResourceUsage, error) {
	cmd := exec.Command("ps", "-o pcpu,pmem,rss -p", strconv.Itoa(pid))
	stdout, _ := cmd.Output()
	split := strings.Split(string(stdout), "\n")
	if len(split) == 0 {
		return nil, &statError{
			pid: pid,
			err: errors.New("no output from ps"),
		}
	}
	fields := strings.Fields(split[1])
	if len(fields) != 3 {
		return nil, &statError{
			pid: pid,
			err: errors.New("wrong number of fields in ps output"),
		}
	}
	cpu, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return nil, &statError{
			pid: pid,
			err: err,
		}
	}
	pmem, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return nil, &statError{
			pid: pid,
			err: err,
		}
	}
	rss, err := strconv.Atoi(fields[2])
	if err != nil {
		return nil, &statError{
			pid: pid,
			err: err,
		}
	}
	return &ResourceUsage{
		CPU:      cpu,
		Memory:   pmem,
		Resident: rss * 1000,
	}, nil
}
