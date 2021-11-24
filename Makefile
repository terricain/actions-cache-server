create-sqlite-migration:
	migrate create -ext sql -dir pkg/database/sqlite/migrations -seq $(NAME)
create-postgres-migration:
	migrate create -ext sql -dir pkg/database/postgres/migrations -seq $(NAME)


build:
	go build -o actions-cache-server ./cli/actions-cache-server/main.go

lint:
	gofumpt -l -w .
	golangci-lint run

generate_mocks:
	mockgen -package=mock_backend -mock_names='Backend=MockStorageBackend' -destination tests/mock_backend/storage.go github.com/terrycain/actions-cache-server/pkg/storage Backend
	mockgen -package=mock_backend -mock_names='Backend=MockDatabaseBackend' -destination tests/mock_backend/database.go github.com/terrycain/actions-cache-server/pkg/database Backend
