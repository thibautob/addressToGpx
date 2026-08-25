//go:build !js

package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	optimize := flag.Bool("optimize", true, "réordonner les waypoints (trajet ouvert, départ = première adresse) et émettre une <rte>")
	flag.Parse()

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	c := &http.Client{Timeout: 10 * time.Second}
	out, err := generate(c, string(input), *optimize, func(msg string) {
		fmt.Fprintln(os.Stderr, msg)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	os.Stdout.WriteString(out)
}
