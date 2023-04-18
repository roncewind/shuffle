package shuffle

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/roncewind/go-util/util"
)

// ----------------------------------------------------------------------------

// internal type for tracking the record
type record struct {
	item         any
	count        int
	initPosition int
}

var recordWG sync.WaitGroup
var recordMutex sync.Mutex

// ----------------------------------------------------------------------------

// input a channel of records to be shuffled
// output a channel of shuffled records
func Shuffle[T any](ctx context.Context, in chan T) <-chan T {

	count := 0
	recordChan := make(chan record, 5)
	outChan := make(chan T)
	recordWG.Add(1)
	go func() {
		for item := range util.OrDone(ctx, in) {
			count++
			r := record{
				item:         item,
				count:        0,
				initPosition: count,
			}
			recordChan <- r
		}
		recordWG.Done()
		fmt.Println("Total records received:", count)
	}()
	go doShuffle(ctx, recordChan, outChan)
	return outChan
}

// ----------------------------------------------------------------------------

// shuffle the records
func doShuffle[T any](ctx context.Context, in chan record, out chan T) {
	count := 0
	go func() {
		recordWG.Wait()
		recordMutex.Lock() //lock so we don't use the record channel again
		close(in)
	}()
	for record := range util.OrDone(ctx, in) {
		if rand.Intn(10) < 7 && recordMutex.TryLock() {
			record := record
			go func() {
				recordWG.Add(1)
				defer recordWG.Done()
				//shuffle
				record.count++ //future, multiple shuffle
				in <- record
				recordMutex.Unlock()
			}()
		} else {
			count++
			fmt.Println(record.item, record.count)
			out <- record.item.(T)
		}

	}
	fmt.Println("Total records sent:", count)
	close(out)
}
