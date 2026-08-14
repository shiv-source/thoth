.PHONY: web build run test race lint clean

clean:
	rm -rf bin && rm -rf internal/webui/dist

web: clean
	cd web && pnpm install --frozen-lockfile && pnpm run build
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

