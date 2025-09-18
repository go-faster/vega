test:
	@./go.test.sh
.PHONY: test

coverage:
	@./go.coverage.sh
.PHONY: coverage

test_fast:
	go test ./...
.PHONY: test_fast

tidy:
	go mod tidy
.PHONY: tidy

up:
	go run ./cmd/vega-install
.PHONY: up

down:
	kind delete cluster -n vega
.PHONY: down

helm:
	bash _hack/helm-install.sh
.PHONY: helm
