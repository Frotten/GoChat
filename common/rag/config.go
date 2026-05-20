package rag

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ProjectRoot          string
	QdrantURL            string
	Collection           string
	EmbeddingModel       string
	EmbeddingDim         int
	EmbeddingType        string // openai | ollama，可与 OPENAI_TYPE（聊天）分离
	EmbeddingBaseURL     string
	OllamaEmbeddingModel string
	OpenAIAPIKey         string
	TopK                 int
	ChunkSize            int
	ChunkOverlap         int
}

func LoadConfig() Config {
	// 向量维度：Ollama / OpenAI 共用，须与所选 embedding 模型输出维度一致
	dim, _ := strconv.Atoi(os.Getenv("EMBEDDING_DIM"))
	if dim <= 0 {
		dim, _ = strconv.Atoi(os.Getenv("OPENAI_EMBEDDING_DIM"))
	}
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

	embeddingType := strings.TrimSpace(os.Getenv("EMBEDDING_TYPE"))
	if embeddingType == "" {
		embeddingType = strings.TrimSpace(os.Getenv("OPENAI_TYPE"))
	}
	if embeddingType == "" {
		embeddingType = "openai"
	}

	embeddingBase := strings.TrimSpace(os.Getenv("EMBEDDING_BASE_URL"))
	if embeddingBase == "" {
		if strings.EqualFold(embeddingType, "ollama") {
			embeddingBase = os.Getenv("OLLAMA_BASE_URL")
			if embeddingBase == "" {
				embeddingBase = "http://localhost:11434"
			}
		} else {
			embeddingBase = os.Getenv("OPENAI_BASE_URL")
			if embeddingBase == "" {
				embeddingBase = "https://api.openai.com/v1"
			}
		}
	}
	embeddingBase = strings.TrimRight(embeddingBase, "/")

	projectRoot := os.Getenv("PROJECT_ROOT")
	if projectRoot == "" {
		projectRoot = "./Info"
	}

	collection := os.Getenv("QDRANT_COLLECTION")
	if collection == "" {
		collection = "knowledge"
	}

	ollamaEmbedModel := strings.TrimSpace(os.Getenv("OLLAMA_EMBEDDING_MODEL"))

	return Config{
		ProjectRoot:          projectRoot,
		QdrantURL:            qdrantURL,
		Collection:           collection,
		EmbeddingModel:       os.Getenv("OPENAI_EMBEDDING_MODEL"),
		EmbeddingDim:         dim,
		EmbeddingType:        embeddingType,
		EmbeddingBaseURL:     embeddingBase,
		OllamaEmbeddingModel: ollamaEmbedModel,
		OpenAIAPIKey:         os.Getenv("OPENAI_API_KEY"),
		TopK:                 topK,
		ChunkSize:            chunkSize,
		ChunkOverlap:         overlap,
	}
}

func (c Config) UseOllamaEmbedding() bool {
	return strings.EqualFold(c.EmbeddingType, "ollama")
}

func (c Config) Enabled() bool {
	if c.EmbeddingModel == "" || c.QdrantURL == "" || c.Collection == "" {
		return false
	}
	if c.UseOllamaEmbedding() {
		return true
	}
	return c.OpenAIAPIKey != ""
}
