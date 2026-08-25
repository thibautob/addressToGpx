//go:build !js

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	c := &http.Client{Timeout: 10 * time.Second}
	out, err := generate(c, string(input), func(msg string) {
		fmt.Fprintln(os.Stderr, msg)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	os.Stdout.WriteString(out)
}
