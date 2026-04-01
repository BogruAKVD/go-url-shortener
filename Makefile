test:
	go test ./...

run-memory:
	go run ./cmd/server -storage=memory

run-postgres:
	go run ./cmd/server \
		-storage=postgres \
		-db-host=localhost \
		-db-port=5432 \
		-db-user=postgres \
		-db-password=postgres \
		-db-name=url_shortener

compose-up:
	docker compose up --build
