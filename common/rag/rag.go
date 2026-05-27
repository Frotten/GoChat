package rag

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
)

var (
	globalService *Service
	once          sync.Once
)

type Service struct {
	cfg      Config
	embedder *Embedder
	qdrant   *QdrantClient
}

func GetService() *Service {
	once.Do(func() {
		cfg := LoadConfig()
		globalService = &Service{
			cfg:      cfg,
			embedder: NewEmbedder(cfg),
			qdrant:   NewQdrantClient(cfg),
		}
	})
	return globalService
}

func (s *Service) Enabled() bool {
	return s.cfg.Enabled()
}

// IndexFromInfo 扫描 PROJECT_ROOT 下的 .txt 文件并写入 Qdrant
func (s *Service) IndexFromInfo(ctx context.Context) error {
	if !s.Enabled() {
		return fmt.Errorf("RAG is not configured: set OLLAMA_EMBEDDING_MODEL, QDRANT_HTTP_URL and QDRANT_COLLECTION in Env.env")
	}
	if err := os.MkdirAll(s.cfg.ProjectRoot, 0755); err != nil {
		return fmt.Errorf("create project root: %w", err)
	}
	if err := s.qdrant.EnsureCollection(ctx); err != nil {
		return err
	}

	var files []string
	err := filepath.Walk(s.cfg.ProjectRoot, func(path string, info os.FileInfo, err error) error {
		//匿名回调函数，filepath.Walk会遍历指定目录下的所有文件和子目录，并对每个文件或目录调用这个函数
		//用来获取所有的.txt文件路径
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".txt") ||
			strings.EqualFold(filepath.Ext(path), ".md") ||
			strings.EqualFold(filepath.Ext(path), ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(files) == 0 {
		log.Printf("[RAG] no .txt files found in %s", s.cfg.ProjectRoot)
		return nil
	}

	total := 0
	for _, filePath := range files {
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("[RAG] skip %s: %v", filePath, err)
			continue
		}
		//固定大小滑动窗口分割，通过字符数硬性切割，但通过重叠来弥补语义断裂的问题
		chunks := splitText(string(data), s.cfg.ChunkSize, s.cfg.ChunkOverlap)
		rel, _ := filepath.Rel(s.cfg.ProjectRoot, filePath)
		for i, chunk := range chunks {
			vec, err := s.embedder.Embed(ctx, chunk)
			if err != nil {
				return fmt.Errorf("embed %s chunk %d: %w", rel, i, err)
			}
			pointID := chunkPointID(rel, i)
			point := qdrantPoint{
				ID:     pointID,
				Vector: vec,
				Payload: map[string]interface{}{
					"text":     chunk,
					"source":   rel,
					"chunk_id": i,
				},
			}
			if err := s.qdrant.Upsert(ctx, []qdrantPoint{point}); err != nil {
				return err
			}
			total++
		}
		log.Printf("[RAG] indexed %s (%d chunks)", rel, len(chunks))
	}
	log.Printf("[RAG] index complete: %d points from %d files", total, len(files))
	return nil
}

// Retrieve 根据用户问题检索相关文档片段
func (s *Service) Retrieve(ctx context.Context, query string) string {
	if !s.Enabled() || strings.TrimSpace(query) == "" {
		return ""
	}
	vec, err := s.embedder.Embed(ctx, query)
	if err != nil {
		log.Printf("[RAG] retrieve embed error: %v", err)
		return ""
	}
	chunks, err := s.qdrant.Search(ctx, vec, s.cfg.TopK)
	if err != nil {
		log.Printf("[RAG] retrieve search error: %v", err)
		return ""
	}
	if len(chunks) == 0 {
		return ""
	}
	var b strings.Builder
	for i, c := range chunks {
		b.WriteString(fmt.Sprintf("[%d] %s\n\n", i+1, c))
	}
	return strings.TrimSpace(b.String())
}

// chunkPointID 生成 Qdrant 要求的 UUID（同一文件+分块索引可稳定复现，便于重复索引覆盖）
func chunkPointID(source string, chunkIndex int) string {
	name := fmt.Sprintf("%s:%d", source, chunkIndex)
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(name)).String()
}
