# Upstream Patches Register

This document tracks all proposed or applied modifications to the core `ffuf` engine (`pkg/`), their justifications, and reproduction instructions.

---

## Patch 001: Signal Listener Goroutine Leak in `interruptMonitor()`

### Status
- **Status:** Proposed Draft Patch (Minimal 5-line diff)
- **Target File:** `pkg/engine/job.go`
- **Component:** `Job.interruptMonitor()`

### Problem Description
In `pkg/engine/job.go`:
```go
func (j *Job) interruptMonitor() {
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		for range sigChan {
			j.setError("Caught keyboard interrupt (Ctrl-C)\n")
			j.Stop()
		}
	}()
}
```
When `Job.Start()` is invoked, `interruptMonitor()` registers `sigChan` with `signal.Notify` and launches a goroutine that loops over `sigChan`. 
However, when the job terminates (either through completion, context cancellation, or `j.Stop()`), `signal.Stop(sigChan)` is never called, and `sigChan` is never closed. Consequently, the background goroutine remains blocked on `for range sigChan` indefinitely. In a long-running desktop application running consecutive scans, this leaks one goroutine per scan, triggering failures under `go.uber.org/goleak`.

### Proposed Minimal Patch
```diff
--- a/pkg/engine/job.go
+++ b/pkg/engine/job.go
@@ -395,11 +395,19 @@ func (j *Job) interruptMonitor() {
 	sigChan := make(chan os.Signal, 2)
 	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
 	go func() {
-		for range sigChan {
-			j.setError("Caught keyboard interrupt (Ctrl-C)\n")
-			// Stop reopens the pause gate itself, so a worker blocked at a
-			// checkpoint wakes and observes the stop.
-			j.Stop()
+		defer signal.Stop(sigChan)
+		for {
+			select {
+			case <-j.Config.Context.Done():
+				return
+			case _, ok := <-sigChan:
+				if !ok {
+					return
+				}
+				j.setError("Caught keyboard interrupt (Ctrl-C)\n")
+				j.Stop()
+				return
+			}
 		}
 	}()
 }
```

### Verification & Justification
- `j.Config.Context` is defined in `pkg/ffuf/config.go:57` as `Context context.Context`.
- When `j.Config.Cancel()` is called by `Job.Stop()`, `j.Config.Context.Done()` fires.
- The `defer signal.Stop(sigChan)` ensures that the OS signal table unregisters the channel, and the goroutine exits immediately.
- Tested and verified with `go test -race` and `goleak.VerifyTestMain`.

---

## Upstream Thread Safety Audit Confirmation

1. **`MatcherManager` (`pkg/filter/filter.go:21`):**
   - Verified that all map operations on `Matchers`, `Filters`, and `PerDomainFilters` are guarded by `mu sync.RWMutex`.
   - Adding or removing filters concurrently via `AddFilter` or `RemoveFilter` while workers call `Matches()` is thread-safe and requires no additional upstream patching.

2. **`RateThrottle` (`pkg/engine/rate.go:14`):**
   - Verified that `RateMutex sync.Mutex` guards rate updates during runtime `ChangeRate()` calls.
