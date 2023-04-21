package shuffle

import (
	"context"
	"fmt"

	// "fmt"
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

var bufferSize int = 1000
var delayInMicros time.Duration = 1

// ----------------------------------------------------------------------------

// input a channel of records to be shuffled
// output a channel of shuffled records
func Shuffle[T any](ctx context.Context, in chan T, targetDistance int) <-chan T {

	bufferSize = int(math.Round(float64(targetDistance) * 1.7))
	if bufferSize <= 0 {
		bufferSize = 1
	}
	// fmt.Println("bufferSize:", bufferSize)
	count := 0
	recordChan := make(chan *record, 5)
	outChan := make(chan T)
	go func() {
		for item := range util.OrDone(ctx, in) {
			count++
			r := record{
				item:         item,
				count:        0,
				initPosition: count,
			}
			recordChan <- &r
		}
		close(recordChan)
		// fmt.Println("Total records received:", count)
	}()
	go doShuffle(ctx, recordChan, outChan)
	return outChan
}

// ----------------------------------------------------------------------------

// shuffle the records
func doShuffle[T any](ctx context.Context, in chan *record, out chan T) {

	var wg sync.WaitGroup
	readCount := 0
	writeCount := 0
	distance := 0
	var sliceMutex sync.RWMutex
	recordBuffer := make([]*record, bufferSize)
	getRecordPtr := func(slot int) *record {
		sliceMutex.RLock()
		rptr := recordBuffer[slot]
		sliceMutex.RUnlock()
		return rptr
	}
	doneReading := make(chan struct{})
	wg.Add(2)
	// read from the in channel randomly putting records in slice slots
	go func() {
		r := rand.New(rand.NewSource(time.Now().Unix()))
		for item := range util.OrDone(ctx, in) {
			readCount++
			slot := r.Intn(bufferSize)
			for getRecordPtr(slot) != nil {
				slot = r.Intn(bufferSize)
			}
			sliceMutex.Lock()
			recordBuffer[slot] = item
			sliceMutex.Unlock()
		}
		doneReading <- struct{}{}
		wg.Done()
	}()

	// randomly read records from slice slots and put them in the out channel
	go func() {
		r := rand.New(rand.NewSource(time.Now().Unix()))
		var item *record
	stillReading:
		for {
			select {
			case <-doneReading:
				break stillReading
			default:
				slot := r.Intn(bufferSize)
				for getRecordPtr(slot) == nil {
					slot = r.Intn(bufferSize)
				}

				sliceMutex.Lock()
				item, recordBuffer[slot] = recordBuffer[slot], nil
				sliceMutex.Unlock()
				writeCount++
				distance += int(math.Abs(float64(writeCount - item.initPosition)))
				// pause before each round to allow the writer to fill in the slice
				time.Sleep(delayInMicros * time.Microsecond)
				// fmt.Println(item.initPosition, "-->", writeCount, "d=", writeCount-item.initPosition)
				out <- item.item.(T)

			}
		}
		// flush the rest from the buffer
		sliceMutex.Lock()
		for i, item := range recordBuffer {
			if item != nil {
				writeCount++
				distance += int(math.Abs(float64(writeCount - item.initPosition)))
				// fmt.Println(item.initPosition, "-->", writeCount, "d=", writeCount-item.initPosition)
				out <- item.item.(T)
				recordBuffer[i] = nil
			}
		}
		sliceMutex.Unlock()
		close(out)
		wg.Done()
	}()
	wg.Wait()
	fmt.Println("Total records written:", writeCount)
	fmt.Println("Total distance:", distance)
	fmt.Println("Average distance:", distance/writeCount)
}
