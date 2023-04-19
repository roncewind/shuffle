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
	recordChan := make(chan *record, 5)
	outChan := make(chan T)
	// recordWG.Add(1)
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
		// recordWG.Done()
		close(recordChan)
		fmt.Println("Total records received:", count)
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
	bufferSize := 50000

	recordBuffer := make([]*record, bufferSize)
	distance := 0
	doneReading := false
	wg.Add(2)
	go func() {
		r := rand.New(rand.NewSource(time.Now().Unix()))
		for item := range util.OrDone(ctx, in) {
			readCount++
			slot := r.Intn(bufferSize)
			for recordBuffer[slot] != nil {
				slot = r.Intn(bufferSize)
			}
			recordBuffer[slot] = item
		}
		fmt.Println("done reading")
		doneReading = true
		wg.Done()
	}()

	go func() {
		r := rand.New(rand.NewSource(time.Now().Unix()))
		var item *record
		for !doneReading {
			slot := r.Intn(bufferSize)
			for recordBuffer[slot] == nil && !doneReading {
				slot = r.Intn(bufferSize)
			}
			if !doneReading {
				item, recordBuffer[slot] = recordBuffer[slot], nil
				writeCount++
				distance += int(math.Abs(float64(writeCount - item.initPosition)))
				fmt.Println(item.initPosition, "-->", writeCount, "d=", writeCount-item.initPosition)
				out <- item.item.(T)
			}
		}
		fmt.Println("time to flush")
		// flush the rest from the buffer
		for i, item := range recordBuffer {
			if item != nil {
				writeCount++
				distance += int(math.Abs(float64(writeCount - item.initPosition)))
				fmt.Println(item.initPosition, "-->", writeCount, "d=", writeCount-item.initPosition)
				out <- item.item.(T)
				recordBuffer[i] = nil
			}
		}
		fmt.Println("flushed, goodbye")
		close(out)
		wg.Done()
	}()
	wg.Wait()
	fmt.Println("Total records written:", writeCount)
	fmt.Println("Total distance:", distance)
	fmt.Println("Average distance:", distance/writeCount)
}

// ----------------------------------------------------------------------------

// shuffle the records
func doShuffleOLD[T any](ctx context.Context, in chan record, out chan T) {
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
