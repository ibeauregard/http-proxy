#!/bin/sh

clear()
{
  docker volume rm -f "$1" >/dev/null
  echo "Cache cleared"
}

if [ "$(docker ps -qf name="$1")" ]; then
  make stop
  clear "$2"
  make run
else
  clear "$2"
fi
