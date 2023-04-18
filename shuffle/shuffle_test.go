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

	generateValues := func() chan int {
		stream := make(chan int)
		go func() {
			defer close(stream)
			for i := 0; i < 20; i++ {
				stream <- i
			}
		}()
		return stream
	}

	accumulator := 0
	for item := range Shuffle(ctx, generateValues()) {
		accumulator += item
	}

	if accumulator != 190 {
		t.Fatal("error in Shuffle")
	}
}
