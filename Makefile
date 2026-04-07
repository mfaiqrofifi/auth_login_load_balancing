.PHONY: run docker-local docker-prod logs stop test

run:
	go run ./cmd/server

docker-local:
	docker compose up --build

docker-prod:
	docker compose -f docker-compose.prod.yml up --build -d

logs:
	docker compose logs -f

stop:
	docker compose down
	docker compose -f docker-compose.prod.yml down

test:
	go test ./...
