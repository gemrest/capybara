FROM golang:1.16.6-alpine3.14 AS build_base

RUN apk add --no-cache git

WORKDIR /tmp/capybara

COPY go.mod .

COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o ./out/capybara .

FROM alpine:3.14

RUN apk add ca-certificates

COPY --from=build_base /tmp/capybara/out/capybara /app/capybara

WORKDIR /app

EXPOSE 8080

CMD ["./capybara"]
