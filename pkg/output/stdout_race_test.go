package output

import (
	"fmt"
	"sync"
	"testing"

	"github.com/parsaazari/fuzznexus/pkg/fuzznexus"
)

// TestStdoutput_ConcurrentResult locks the fix for an unsynchronized append:
// Result is called from the engine's worker goroutines, and it appended to
// CurrentResults without a lock, so under concurrency results were silently
// lost (and -race flagged the write). Every result must survive.
//
// Run with -race to also catch the data race directly.
func TestStdoutput_ConcurrentResult(t *testing.T) {
	conf := &fuzznexus.Config{}
	s := NewStdoutput(conf)

	const n = 200
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Result(fuzznexus.Response{
				StatusCode: 200,
				Request: &fuzznexus.Request{
					Input: map[string][]byte{"FUZZ": []byte(fmt.Sprintf("w%d", i))},
					Url:   fmt.Sprintf("http://example/%d", i),
				},
			})
		}(i)
	}
	wg.Wait()

	if got := len(s.GetCurrentResults()); got != n {
		t.Errorf("got %d results, want %d (results lost to the unsynchronized append)", got, n)
	}
}
