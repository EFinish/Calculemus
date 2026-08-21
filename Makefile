# Calculemus build entry points. The wasm target is the seam between the Go
# engine and the browser app: it drops the compiled engine plus Go's JS
# shim into app/public/, where Vite serves them as static assets.

GOROOT := $(shell go env GOROOT)

.PHONY: test wasm smoke dev build e2e e2e-boolean serve release

test:
	go test ./...

wasm:
	GOOS=js GOARCH=wasm go build -o app/public/calculemus.wasm ./wasm
	cp app/public/calculemus.wasm app-boolean/public/calculemus.wasm
	cp "$(GOROOT)/lib/wasm/wasm_exec.js" app/public/wasm_exec.js
	cp app/public/wasm_exec.js app-boolean/public/wasm_exec.js

# Node-based smoke test of the bridge — proves evaluate() works through
# WASM without needing a browser.
smoke: wasm
	node scripts/smoke-wasm.mjs

dev: wasm
	cd app && npm run dev

build: test wasm
	cd app && npm run build
	cd app-boolean && npm run build

# The M1 dogfood test in headless Firefox: compose a universe in the real
# app, verify live verdicts, reload, verify persistence.
e2e: wasm
	cd app && npx playwright test

# The frozen Boolean edition's suite — run when touching app-boolean/ or core.
e2e-boolean: wasm
	cd app-boolean && npx playwright test

# Production: engine tests, WASM, app build, then one binary serving
# everything — the app, and the /api document store for shared universes.
serve: build
	go run ./server -addr :8737 -data data -dist app/dist -boolean-dist app-boolean/dist

# Self-contained binary: both built apps embedded via the "dist" build tag
# (webdist.go). CI cross-compiles this same thing for tagged releases.
release: build
	go build -tags dist -trimpath -ldflags "-s -w" -o calculemus ./server
