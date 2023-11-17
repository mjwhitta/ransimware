-include gomk/main.mk
-include docker/Makefile
-include local/Makefile

ifneq ($(unameS),windows)
spellcheck:
	@codespell -f -L hilighter -S ".git,*.pem,png.go"
endif
