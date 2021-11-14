create-sqlite-migration:
	migrate create -ext sql -dir pkg/database/sqlite/migrations -seq $(NAME)

build:
	go build -o actions-cache-server ./cli/actions-cache-server/main.go
