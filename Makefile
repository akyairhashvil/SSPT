deps-tidy:
	go mod tidy
	go mod verify

deps-check:
	go list -m -u all

mocks:
	go generate ./internal/database
