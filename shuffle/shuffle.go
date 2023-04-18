package shuffle

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

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

// ----------------------------------------------------------------------------

// input a channel of records to be shuffled
// output a channel of shuffled records
func Shuffle[T any](ctx context.Context, in chan T) <-chan T {

	count := 0
	recordChan := make(chan record, 500)
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
	var recordMutex sync.Mutex
	count := 0
	go func() {
		recordWG.Wait()
		recordMutex.Lock() //lock so we don't use the record channel again
		close(in)
	}()
	distance := 0
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for record := range util.OrDone(ctx, in) {
		if r.Intn(100) < 95 && recordMutex.TryLock() {
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
			distance += int(math.Abs(float64(count - record.initPosition)))
			fmt.Println(record.initPosition, record.count, count, count-record.initPosition)
			out <- record.item.(T)
		}

	}
	fmt.Println("Total records sent:", count)
	fmt.Println("Total distance:", distance)
	close(out)
}
