# RAG 演示的默认调参项。
# 这里不硬编码 DOCS_PATH，方便调用方自行传入任意本地路径。
# 其余参数保留默认值，同时也允许被覆盖。
CHUNK_SIZE ?= 500
OVERLAP ?= 100
LIMIT_DOCS ?= 2
LIMIT_CHUNKS ?= 3
SHOW_PREVIEW ?= true

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

# 调用方必须手动提供 DOCS_PATH。
# 示例:
#   make embed DOCS_PATH=./testdata/documents/example
# 下面这些变量会原样转发给 rag-cli，实际执行流程是:
#   load -> chunk -> embed
.PHONY: embed
embed:
	$(if $(strip $(DOCS_PATH)),,$(error DOCS_PATH is required. Example: make embed DOCS_PATH=./testdata/documents/example))
	go run ./cmd/rag-cli -docs "$(DOCS_PATH)" -chunkSize $(CHUNK_SIZE) -overlap $(OVERLAP) -limitDocs $(LIMIT_DOCS) -limitChunks $(LIMIT_CHUNKS) -showPreview=$(SHOW_PREVIEW)

.PHONY: test
test:
	go test ./...
