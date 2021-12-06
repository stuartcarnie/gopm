package procusage

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		in  int
		out string
	}{
		{in: 1000, out: "1.0kB"},
		{in: 10000, out: "10.0kB"},
		{in: 1000000, out: "1.0MB"},
		{in: 1500000, out: "1.5MB"},
		{in: 2000000000, out: "2.0GB"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.in), func(t *testing.T) {
			assert.Equal(t, tt.out, humanBytes(tt.in))
		})
	}
}
