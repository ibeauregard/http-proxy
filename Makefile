app_name := my_proxy
container_name := $(app_name)_container
volume_name := $(app_name)_cache
cache_dir_name := cache

build:
	docker build --tag=$(app_name):latest --build-arg APP_NAME=$(app_name) \
		--build-arg CACHE_DIR_NAME=$(cache_dir_name) .
	docker image prune -f

run:
	docker run --interactive --tty --name=$(container_name) \
		--volume $(volume_name):/home/$(app_name)/$(cache_dir_name) \
		--publish 8080:8080 --rm \
		--env CACHE_DIR_NAME=$(cache_dir_name) $(app_name):latest

restart: stop run

connect:
	docker exec -it $(container_name) sh

clear-cache:
	@sh clear_cache.sh $(container_name) $(volume_name)

stop:
	@docker rm -f $(container_name) &>/dev/null && echo "Stopped any existing container"
