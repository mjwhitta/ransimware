-include gomk/main.mk
-include docker/Makefile
-include local/Makefile

ifneq ($(unameS),Windows)
spellcheck:
	@codespell -f -S ".git,*.pem,png.go"
endif
