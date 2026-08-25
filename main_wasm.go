//go:build js && wasm

package main

import (
	"net/http"
	"syscall/js"
	"time"
)

// createGPX is exposed to JS as window.createGPX(text, onLog) and returns a
// Promise resolving to the GPX document as a string. onLog is called with one
// progress line per address. The work runs in a goroutine: blocking HTTP calls
// are forbidden on the JS event loop thread.
func createGPX(this js.Value, args []js.Value) any {
	input := args[0].String()
	onLog := args[1]

	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) any {
		resolve, reject := promiseArgs[0], promiseArgs[1]
		go func() {
			c := &http.Client{Timeout: 10 * time.Second}
			out, err := generate(c, input, func(msg string) {
				onLog.Invoke(msg)
			})
			if err != nil {
				reject.Invoke(js.Global().Get("Error").New(err.Error()))
				return
			}
			resolve.Invoke(out)
		}()
		return nil
	})
	return js.Global().Get("Promise").New(handler)
}

func main() {
	js.Global().Set("createGPX", js.FuncOf(createGPX))
	// Keep the Go runtime alive so the exported function stays callable.
	select {}
}
