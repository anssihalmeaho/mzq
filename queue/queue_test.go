package queue

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestALotOfStuff(t *testing.T) {
	assert := assert.New(t)

	vq := NewQueue(10)
	var wg sync.WaitGroup
	wg.Add(2)

	count := 1000 * 100
	recCount := 0
	putCount := 0

	go func() {
		defer wg.Done()
		for {
			vq.Get()
			recCount++
			if count == recCount {
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < count; i++ {
			vq.Put(i)
			putCount++
		}
	}()

	wg.Wait()
	t.Logf("Rec.count = %d, Put count = %d", recCount, putCount)
	assert.Equal(recCount, putCount)
}
