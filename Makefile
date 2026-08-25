# addr2gpx build targets

GOROOT := $(shell go env GOROOT)

.PHONY: wasm cli serve clean

# Build the WASM module and refresh the runtime shim into docs/ (GitHub Pages root)
wasm:
	cp "$(GOROOT)/lib/wasm/wasm_exec.js" docs/wasm_exec.js
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o docs/main.wasm .

cli:
	go build -o addr2gpx .

# Local preview of the web app on http://localhost:8000
serve: wasm
	python3 -m http.server 8000 -d docs

clean:
	rm -f addr2gpx docs/main.wasm
