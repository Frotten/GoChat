package rag

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ProjectRoot      string
	QdrantURL        string
	Collection       string
	EmbeddingModel   string
	EmbeddingDim     int
	OpenAIAPIKey     string
	OpenAIBaseURL    string
	TopK             int
	ChunkSize        int
	ChunkOverlap     int
}

func LoadConfig() Config {
	dim, _ := strconv.Atoi(os.Getenv("OPENAI_EMBEDDING_DIM"))
	if dim <= 0 {
		dim = 1024
	}
	topK, _ := strconv.Atoi(os.Getenv("RAG_TOP_K"))
	if topK <= 0 {
		topK = 5
	}
	chunkSize, _ := strconv.Atoi(os.Getenv("RAG_CHUNK_SIZE"))
	if chunkSize <= 0 {
		chunkSize = 500
	}
	overlap, _ := strconv.Atoi(os.Getenv("RAG_CHUNK_OVERLAP"))
	if overlap < 0 {
		overlap = 50
	}

	qdrantURL := strings.TrimSpace(os.Getenv("QDRANT_HTTP_URL"))
	if qdrantURL == "" {
		host := os.Getenv("QDRANT_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("QDRANT_HTTP_PORT")
		if port == "" {
			port = "6333"
		}
		qdrantURL = "http://" + host + ":" + port
	}
	qdrantURL = strings.TrimRight(qdrantURL, "/")

	baseURL := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	projectRoot := os.Getenv("PROJECT_ROOT")
	if projectRoot == "" {
		projectRoot = "./Info"
	}

	collection := os.Getenv("QDRANT_COLLECTION")
	if collection == "" {
		collection = "knowledge"
	}

	return Config{
		ProjectRoot:    projectRoot,
		QdrantURL:      qdrantURL,
		Collection:     collection,
		EmbeddingModel: os.Getenv("OPENAI_EMBEDDING_MODEL"),
		EmbeddingDim:   dim,
		OpenAIAPIKey:   os.Getenv("OPENAI_API_KEY"),
		OpenAIBaseURL:  baseURL,
		TopK:           topK,
		ChunkSize:      chunkSize,
		ChunkOverlap:   overlap,
	}
}

func (c Config) Enabled() bool {
	return c.OpenAIAPIKey != "" &&
		c.EmbeddingModel != "" &&
		c.QdrantURL != "" &&
		c.Collection != ""
}
