MODULE := github.com/gemrest/capybara
DEFAULT_ROOT := gemini://fuwn.me

.PHONY: fmt run build docker

fmt:
	go fmt $(MODULE)...

run: fmt
	go run $(MODULE) $(DEFAULT_ROOT)

build: fmt
	go build $(MODULE)

docker: fmt
	docker build -t fuwn/capybara:latest .

