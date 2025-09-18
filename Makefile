test:
	@./go.test.sh
.PHONY: test

coverage:
	@./go.coverage.sh
.PHONY: coverage

test_fast:
	go test ./...

tidy:
	go mod tidy
up:
	go run ./cmd/vega-install
helm:
	bash _hack/helm-install.sh

