-include gomk/main.mk

build: reportcard dir
	@go build --ldflags "$(LDFLAGS)" -o "$(OUT)" --trimpath ./cmd/*

strip: build
	@find build -type f -exec ./tools/stripper {} +
