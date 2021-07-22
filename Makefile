fmt:
	go fmt github.com/fuwn/capybara...

run: fmt validate
	go run github.com/fuwn/capybara

build: fmt
	go build

docker: fmt
	docker build -t fuwn/capybara:latest .
