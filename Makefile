# Kanjō — Makefile
# Utilisation : make <cible>  (make help pour la liste)

.PHONY: help build build-all test lint tidy desktop

help: ## Afficher l'aide
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | sort | awk 'BEGIN{FS=":.*##"}{printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

build: ## Compiler le binaire kanjo (CGO_ENABLED=0)
	CGO_ENABLED=0 go build -o kanjo ./cmd/kanjo

build-all: ## Cross-compiler les 6 cibles (CGO_ENABLED=0)
	@scripts/build-all.sh 2>/dev/null || \
	  for GOOS in linux darwin windows; do \
	    for GOARCH in amd64 arm64; do \
	      echo "➜ $$GOOS/$$GOARCH"; \
	      CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH go build ./... || exit 1; \
	    done; \
	  done

test: ## Exécuter tous les tests
	CGO_ENABLED=0 go test ./...

lint: ## Exécuter golangci-lint
	golangci-lint run ./...

tidy: ## go mod tidy
	go mod tidy

.PHONY: desktop
desktop: ## Construire le client lourd (nécessite Wails + SDK natif de l'OS courant)
	cd gui/wails && go mod tidy && wails build
