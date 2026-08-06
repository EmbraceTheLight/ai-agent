.PHONY: run
run:
	go run ./cmd/assistant-cli

.PHONY: stream
stream:
	go run ./cmd/assistant-cli --outputType stream

.PHONY: json
json:
	go run ./cmd/assistant-cli --outputType json

.PHONY: function_call
function_call:
	go run ./cmd/assistant-cli --outputType function_call

.PHONY: test
test:
	go test ./...
