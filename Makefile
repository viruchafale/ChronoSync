.PHONY: engine-build engine-test engine-lint engine-format-check check
engine-build:
	cd engine && go build -o ../bin/chronosync-engine ./cmd/chronosync-engine
engine-test:
	cd engine && go test -race ./...
engine-lint:
	cd engine && go vet ./...
engine-format-check:
	test -z "$$(gofmt -l engine)"
check: engine-format-check engine-lint engine-test engine-build
	npm run format:check
	npm run lint
	npm run typecheck
	npm test
	npm run build
	npm run check:docs
	npm run test:integration
	npm run test:recorder

.PHONY: native-check
native-check:
	npm run engine:sidecar
	cargo fmt --manifest-path apps/desktop/src-tauri/Cargo.toml --check
	cargo clippy --locked --manifest-path apps/desktop/src-tauri/Cargo.toml -- -D warnings
	cargo test --locked --manifest-path apps/desktop/src-tauri/Cargo.toml
	npm run desktop:build
