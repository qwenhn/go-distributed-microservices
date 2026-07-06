.PHONY: up down logs build restart tidy

up:
	docker compose up --build
down:
	docker compose down
logs:
	docker compose logs -f
build:
	docker compose build
restart:
	docker compose down && docker compose up --build
tidy:
	find . -name go.mod -execdir go mod tidy \;