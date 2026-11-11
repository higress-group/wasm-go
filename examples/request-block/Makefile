.DEFAULT_GOAL := build

WASM_BUILD ?= ../../scripts/build-wasm.sh

.PHONY: build build-lite build-debug clean

build:
	$(WASM_BUILD) -o main.wasm .

build-lite:
	GO_BUILD_TAGS=wasm_lite_http $(WASM_BUILD) -o main.wasm .

build-debug:
	GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm .

clean:
	rm -f main.wasm wasm-unit-test.wasm wasm-unit-test.wasm.unoptimized
