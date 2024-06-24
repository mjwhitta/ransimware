ifneq ($(unameS),windows)
docker: phony_dummy
	@docker pull alpine:latest
	@docker build -f docker/exfil -t ransimware:exfil .
	@docker buildx prune -f
	@docker rmi alpine:latest
endif
