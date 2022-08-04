build:
	docker build --tag=my_proxy:latest .
	docker image prune -f

run:
	docker run --interactive --tty --name=my_proxy_container --rm my_proxy:latest

restart: stop run

stop:
	@docker rm -f my_proxy_container &>/dev/null && echo "Stopped any existing container"
