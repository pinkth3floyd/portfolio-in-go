.PHONY: css run build docker seed-clean

css:
	cd tailwind && npm install && npm run build

run: css
	go run ./cmd/server

build: css
	go build -o bin/server ./cmd/server

docker:
	docker compose up --build

seed-clean:
	rm -f data/app.db data/app.db-*
