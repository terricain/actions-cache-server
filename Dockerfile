FROM golang:1.17.3-alpine AS base
FROM base AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY cli ./cli/
COPY pkg ./pkg/

RUN apk add --update gcc musl-dev

RUN CGO_ENABLED=1 GOOS=linux go build -o actions-cache-server --tags "linux" ./cli/actions-cache-server/main.go

FROM base AS deploy

COPY --from=build /app/actions-cache-server /actions-cache-server

ENTRYPOINT ["/actions-cache-server"]