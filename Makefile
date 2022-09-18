app_name := my_proxy
cache_dir_name := cache

build:
	docker build --tag=$(app_name):latest --build-arg APP_NAME=$(app_name) \
		--build-arg CACHE_DIR_NAME=$(cache_dir_name) .
	docker image prune -f

run:
	docker run --interactive --tty --name=$(app_name)_container \
		--publish 8080:8080 --rm \
		--env CACHE_DIR_NAME=$(cache_dir_name) $(app_name):latest

restart: stop run

connect:
	docker exec -it $(app_name)_container sh

clear-cache:
	docker exec -it $(app_name)_container sh -c "rm -f $(cache_dir_name)/*"

stop:
	@docker rm -f $(app_name)_container &>/dev/null && echo "Stopped any existing container"
