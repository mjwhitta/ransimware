ifneq ($(unameS),windows)
docker: phony_dummy
	@docker tag ransimware:exfil ransimware:delete &>/dev/null || true
	@docker build -f docker/exfil --pull -t ransimware:exfil . || \
	    docker tag ransimware:delete ransimware:exfil &>/dev/null || \
	    true
	@docker buildx prune -f &>/dev/null || true
	@docker rmi alpine:exfil &>/dev/null || true
	@docker rmi ransimware:delete &>/dev/null || true
endif
