default: build
NAME:=elasticsearch-exporter

.PHONY: build
build:
	go build -o ./.bin/$(NAME)
