# Calculemus build entry points. The wasm target is the seam between the Go
# engine and the browser app: it drops the compiled engine plus Go's JS
# shim into app/public/, where Vite serves them as static assets.

GOROOT := $(shell go env GOROOT)

.PHONY: test wasm smoke dev build e2e

test:
	go test ./...

wasm:
	GOOS=js GOARCH=wasm go build -o app/public/calculemus.wasm ./wasm
	cp "$(GOROOT)/lib/wasm/wasm_exec.js" app/public/wasm_exec.js

# Node-based smoke test of the bridge — proves evaluate() works through
# WASM without needing a browser.
smoke: wasm
	node scripts/smoke-wasm.mjs

dev: wasm
	cd app && npm run dev

build: test wasm
	cd app && npm run build

# The M1 dogfood test in headless Firefox: compose a universe in the real
# app, verify live verdicts, reload, verify persistence.
e2e: wasm
	cd app && npx playwright test
