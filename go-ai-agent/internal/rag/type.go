package rag

type EmbedReq struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type EmbedResp struct {
	EmbeddingsData [][]float64 `json:"embeddings"`
}
