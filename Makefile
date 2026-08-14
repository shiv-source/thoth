.PHONY: web build run test race lint clean

web:
	cd web && pnpm install --frozen-lockfile && pnpm run build
	rm -rf internal/webui/dist
	cp -r web/dist internal/webui/dist

build: web
	go build -o bin/thoth ./cmd/thoth

run: build
	./bin/thoth serve

test:
	go test ./...

race:
	go test -race ./...

lint:
	golangci-lint run
	cd web && pnpm run lint && pnpm exec tsc --noEmit

clean:
	rm -rf bin
