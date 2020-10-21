-include gomk/main.mk

build: reportcard dir
	@go build --ldflags "$(LDFLAGS)" -o "$(OUT)" --trimpath ./cmd/*
	@find build -type f -exec ./tools/stripper {} +
