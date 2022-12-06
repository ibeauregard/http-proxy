app_name := http-proxy
container_name := $(app_name)_container
volume_name := $(app_name)_cache
cache_dir_name := cache
entry_script_name := entry.sh
test_coverage_filename := coverage.out
unit_tests_dir_path := tests/unit

build:
	docker build --tag=$(app_name):latest --build-arg APP_NAME=$(app_name) \
		--build-arg CACHE_DIR_NAME=$(cache_dir_name) \
		--build-arg ENTRY_SCRIPT_NAME=$(entry_script_name) \
		--build-arg TEST_COVERAGE_FILENAME=$(test_coverage_filename) \
		--build-arg UNIT_TESTS_DIR_PATH=$(unit_tests_dir_path) .
	docker image prune -f

run:
	docker run --interactive --tty --name=$(container_name) \
		--volume $(volume_name):/home/$(app_name)/$(cache_dir_name) \
		--volume=$(shell pwd)/$(unit_tests_dir_path):/home/$(app_name)/$(unit_tests_dir_path) \
		--publish 8080:8080 --rm \
		--env APP_NAME=$(app_name) \
		--env CACHE_DIR_NAME=$(cache_dir_name) \
		--env ENTRY_SCRIPT_NAME=$(entry_script_name) \
		--env TEST_COVERAGE_FILENAME=$(test_coverage_filename) \
		--env UNIT_TESTS_DIR_PATH=$(unit_tests_dir_path) $(app_name):latest

restart: stop run

connect:
	docker exec -it $(container_name) sh

clear-cache:
	@sh clear_cache.sh $(container_name) $(volume_name)

stop:
	@docker rm -f $(container_name) &>/dev/null && echo "Stopped any existing container"
