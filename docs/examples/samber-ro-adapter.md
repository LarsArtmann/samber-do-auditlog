# Bridging OnEvent to samber/ro (Reactive Streams)

This is a **reference example** — `samber-do-auditlog` does **not** depend on
[`samber/ro`](https://github.com/samber/ro). The `Config.OnEvent` callback fires
for every lifecycle event outside the recorder mutex, which makes it a natural
source for a reactive stream.

## Concept

`samber/ro` is a Go implementation of ReactiveX. Its `ro.FromChannel` operator
turns a `<-chan T` into an `Observable[T]` you can transform with `ro.Pipe`,
`ro.Filter`, `ro.Map`, etc., and then subscribe to.

To bridge audit events to a reactive stream:

1. Open a buffered `chan auditlog.Event`.
2. Feed it from `OnEvent` (non-blocking, dropping on a full buffer to protect
   the hot hook path).
3. Wrap the channel with `ro.FromChannel` and compose operators.

This lets you declaratively express queries like "all invocation errors in the
last minute" or "services whose status flipped to error".

## Reference Implementation

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/ro"

	auditlog "github.com/larsartmann/samber-do-auditlog"
)

// eventsChannel is a bounded buffer between the synchronous OnEvent hook and
// the asynchronous observable pipeline. The buffer absorbs bursts; a full
// buffer drops the event rather than blocking the DI hot path.
func eventsChannel(buffer int) (chan auditlog.Event, func(auditlog.Event)) {
	ch := make(chan auditlog.Event, buffer)

	produce := func(evt auditlog.Event) {
		select {
		case ch <- evt:
		default:
			// Buffer full — drop to avoid stalling container hooks.
		}
	}

	return ch, produce
}

func main() {
	ch, produce := eventsChannel(1024)

	plugin, err := auditlog.New(auditlog.Config{
		Enabled: true,
		OnEvent: produce,
	})
	if err != nil {
		panic(err)
	}
	_ = plugin // → do.NewWithOpts(plugin.Opts())

	// Build the observable: keep only invocation errors and project to a
	// readable message. Replace with the operators you need.
	errorStream := ro.Pipe[auditlog.Event, string](
		ro.FromChannel(ch),
		ro.Filter(func(evt auditlog.Event) bool {
			return evt.HasError() && evt.IsInvocation()
		}),
		ro.Map(func(evt auditlog.Event) string {
			return fmt.Sprintf("%s failed: %s", evt.ServiceName, evt.Error)
		}),
	)

	sub := errorStream.Subscribe(ro.Observer[string]{
		Next: func(msg string) { fmt.Println("⚠️ ", msg) },
	})
	defer sub.Unsubscribe()

	// Run your container, then close the channel to signal completion.
	// close(ch)  // do this during shutdown, after the container stops.

	time.Sleep(5 * time.Second)
	_ = context.Background()
}
```

## Notes

- **Backpressure**: the example drops events when the buffer is full. For strict
  delivery, size the buffer for your worst-case burst or use an unbounded queue.
- **Close the channel on shutdown**: `ro.FromChannel` completes the observable
  when the channel is closed, which is the signal to release subscribers. Close
  `ch` after `injector.Shutdown()`.
- **Multiple subscribers**: `ro.FromChannel` fans out to subscribers, but they
  compete for channel values (hot stream). For independent cold streams per
  subscriber, multicast via a separate channel per subscriber from `OnEvent`.
- **No vendor lock-in**: the library only calls your `OnEvent` func — `samber/ro`
  lives entirely in your application.
