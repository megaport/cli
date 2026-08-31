//go:build !wasm

package output

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSpinnerRunWithPagerRace is a regression test for ESD-1644: a live
// spinner animating in the background must not race with RunWithPager
// concurrently swapping os.Stdout, and its frames (now routed to stderr)
// must never bleed into the pager's captured stdout content. Run with
// -race to exercise the regression this guards against.
func TestSpinnerRunWithPagerRace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing-sensitive race regression test")
	}
	orig := isTerminalCached.Load()
	t.Cleanup(func() { SetIsTerminal(orig) })
	SetIsTerminal(true)

	// Tall terminal so RunWithPager always takes the direct-write path.
	setTerminalHeightForTesting(1000)
	t.Cleanup(func() { setTerminalHeightForTesting(0) })

	const iterations = 30

	stdout := captureStdout(t, func() {
		captureStderr(t, func() {
			spinner := NewSpinner(true)
			spinner.Start("Racing...")

			var wg sync.WaitGroup
			for i := 0; i < iterations; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					_ = RunWithPager(func() error {
						fmt.Printf("row%d\n", i)
						return nil
					})
				}(i)
			}
			wg.Wait()
			spinner.Stop()
		})
	})

	for i := 0; i < iterations; i++ {
		assert.Contains(t, stdout, fmt.Sprintf("row%d", i))
	}
	assert.NotContains(t, stdout, "\r\033[K",
		"spinner frames must never bleed into the pager's stdout content")
}

// TestSpinnerCaptureOutputRace is a regression test for ESD-1644: a live
// spinner writing to os.Stderr must not race with CaptureOutput
// concurrently reassigning both os.Stdout and os.Stderr. Run with -race to
// exercise the regression this guards against.
func TestSpinnerCaptureOutputRace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing-sensitive race regression test")
	}
	orig := isTerminalCached.Load()
	t.Cleanup(func() { SetIsTerminal(orig) })
	SetIsTerminal(true)

	captureStderr(t, func() {
		spinner := NewSpinner(true)
		spinner.Start("Racing...")

		var wg sync.WaitGroup
		const iterations = 30
		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				CaptureOutput(func() {
					fmt.Printf("captured%d\n", i)
				})
			}(i)
		}
		wg.Wait()
		spinner.Stop()
	})
}
