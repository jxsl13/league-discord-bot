
.PHONY: generate build

build:
	go build .

generate:
	rm -rf sqlc
	sqlc generate

