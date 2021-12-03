package gopmctlcmd

import (
	"sync"
)

func merge(cs ...<-chan []byte) <-chan []byte {
	if len(cs) == 0 {
		return nil
	} else if len(cs) == 1 {
		return cs[0]
	}

	var wg sync.WaitGroup
	out := make(chan []byte)

	// Start an output goroutine for each input channel in cs. output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan []byte) {
		defer wg.Done()
		for n := range c {
			out <- n
		}
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
