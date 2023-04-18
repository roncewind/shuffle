package shuffle

import (
	"context"
	"testing"
)

// ----------------------------------------------------------------------------

// test Bridge
func TestUtil_Shuffle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// generate values from 0 to n
	n := 2000
	generateValues := func() chan int {
		stream := make(chan int)
		go func() {
			defer close(stream)
			for i := 1; i <= n; i++ {
				stream <- i
			}
		}()
		return stream
	}

	accumulator := 0
	for item := range Shuffle(ctx, generateValues()) {
		accumulator += item
	}

	if accumulator != (n*(n+1))/2 {
		t.Fatal("error in Shuffle")
	}
}
