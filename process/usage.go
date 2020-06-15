package process

import (
	"fmt"
)

// ResourceUsage describes the system resource usage of a process.
type ResourceUsage struct {
	CPU      float64
	Memory   float64
	Resident int
}

// HumanResident returns a human readable version of the memory usage.
func (r ResourceUsage) HumanResident() string {
	return humanBytes(r.Resident)
}

// StatError is returned when Stat-ing a PID.
type StatError struct {
	PID int
	Err error
}

// Error implements error
func (e StatError) Error() string {
	return fmt.Sprintf("failed to stat pid %d: %s", e.PID, e.Err)
}

// Unwrap allows StatError to be used with errors.Is/As.
func (e StatError) Unwrap() error {
	return e.Err
}

func humanBytes(b int) string {
	const magnitudes = "kMGTPE"
	const unit = 1000

	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(b)/float64(div), magnitudes[exp])
}
