package logger

import "bytes"

// lastNLines walks backwards through a buffer to identify the location of the
// newline preceeding the Nth line of the content.
func lastNLines(b []byte, n int) []byte {
	if len(b) == 0 || n <= 0 {
		return nil
	}
	// We'll count the last line as a full line even if it doesn't
	// end with a newline.
	i := len(b)
	if b[len(b)-1] == '\n' {
		i--
	}
	for nfound := 0; nfound < n; {
		nl := bytes.LastIndexByte(b[:i], '\n')
		if nl == -1 {
			return b
		}
		nfound++
		i = nl
	}
	return b[i+1:]
}
