create-sqlite-migration:
	migrate create -ext sql -dir pkg/database/sqlite/migrations -seq $(NAME)
