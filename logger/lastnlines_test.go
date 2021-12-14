package logger

import (
	"testing"
)

var lastNLinesTests = []struct {
	testName string
	input    string
	n        int
	want     string
}{{
	testName: "Empty",
	input:    "",
	n:        0,
	want:     "",
}, {
	testName: "SingleLine",
	input:    "foo\n",
	n:        1,
	want:     "foo\n",
}, {
	testName: "TwoLinesNotLast",
	input:    "hello\nworld\nand\ngopm\n",
	n:        2,
	want:     "and\ngopm\n",
}, {
	testName: "ZeroLinesNotLast",
	input:    "hello\nworld\nand\ngopm\n",
	n:        0,
	want:     "",
}, {
	testName: "MoreLinesThanThereAre",
	input:    "hello\nbar\n",
	n:        20,
	want:     "hello\nbar\n",
}, {
	testName: "NotNewlineTerminated",
	input:    "hello\nbar",
	n:        1,
	want:     "bar",
}, {
	testName: "NotNewlineTerminated",
	input:    "x\na\nb",
	n:        2,
	want:     "a\nb",
}, {
	testName: "NewlineAtStart",
	input:    "\nxx",
	n:        2,
	want:     "\nxx",
}}

func TestLastNLines(t *testing.T) {
	for _, test := range lastNLinesTests {
		t.Run(test.testName, func(t *testing.T) {
			got := string(lastNLines([]byte(test.input), test.n))
			if got != test.want {
				t.Fatalf("unexpected output;\ngot %q\nwant %q", got, test.want)
			}
		})
	}
}
