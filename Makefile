MODULE := github.com/gemrest/capybara

fmt:
	go fmt $(MODULE)...

run: fmt
	go run $(MODULE)

build: fmt
	go build

docker: fmt
	docker build -t fuwn/capybara:latest .

