fmt:
	go fmt github.com/gemrest/capybara...

run: fmt
	go run github.com/gemrest/capybara

build: fmt
	go build

docker: fmt
	docker build -t fuwn/capybara:latest .

