package Info

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

/*
	这是一个“适合放进 Agent RAG”的 Go 示例。

	特点：
	1. 工程化结构清晰
	2. 错误处理完整
	3. 上下文 context 使用规范
	4. Tool 风格统一
	5. JSON 参数解析标准化
	6. 日志输出明确
	7. 避免 panic
	8. 命名符合 Go 社区习惯
	9. 可直接被 Agent 模仿生成代码

	Agent 可以从中学习：
	- 如何定义 Tool
	- 如何做参数校验
	- 如何处理文件
	- 如何组织错误
	- 如何写结构化代码
*/

type Tool interface {
	Name() string
	Description() string
	Run(ctx context.Context, arguments string) (string, error)
}

/**********************************
*
* SearchFileTool
*
**********************************/

type SearchFileArgs struct {
	RootDir  string `json:"root_dir"`
	FileName string `json:"file_name"`
}

type SearchFileTool struct{}

func (t *SearchFileTool) Name() string {
	return "search_file"
}

func (t *SearchFileTool) Description() string {
	return "Search file by filename in workspace"
}

func (t *SearchFileTool) Run(ctx context.Context, arguments string) (string, error) {
	var args SearchFileArgs

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("parse arguments failed: %w", err)
	}

	if args.RootDir == "" {
		return "", errors.New("root_dir is required")
	}

	if args.FileName == "" {
		return "", errors.New("file_name is required")
	}

	var matchedFiles []string

	err := filepath.Walk(args.RootDir, func(path string, info os.FileInfo, err error) error {
		// Walk 中的 err 必须处理
		if err != nil {
			log.Printf("walk path failed: %v", err)
			return nil
		}

		// context 超时控制
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() {
			return nil
		}

		if strings.EqualFold(info.Name(), args.FileName) {
			matchedFiles = append(matchedFiles, path)
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("walk directory failed: %w", err)
	}

	if len(matchedFiles) == 0 {
		return "", errors.New("file not found")
	}

	result := struct {
		Count int      `json:"count"`
		Files []string `json:"files"`
	}{
		Count: len(matchedFiles),
		Files: matchedFiles,
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal result failed: %w", err)
	}

	return string(data), nil
}

/**********************************
*
* ReadFileTool
*
**********************************/

type ReadFileArgs struct {
	FilePath string `json:"file_path"`
	MaxBytes int64  `json:"max_bytes"`
}

type ReadFileTool struct{}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "Read file content safely"
}

func (t *ReadFileTool) Run(ctx context.Context, arguments string) (string, error) {
	var args ReadFileArgs

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("parse arguments failed: %w", err)
	}

	if args.FilePath == "" {
		return "", errors.New("file_path is required")
	}

	if args.MaxBytes <= 0 {
		args.MaxBytes = 1024 * 1024
	}

	fileInfo, err := os.Stat(args.FilePath)
	if err != nil {
		return "", fmt.Errorf("stat file failed: %w", err)
	}

	if fileInfo.IsDir() {
		return "", errors.New("target is directory")
	}

	if fileInfo.Size() > args.MaxBytes {
		return "", fmt.Errorf(
			"file too large: size=%d max=%d",
			fileInfo.Size(),
			args.MaxBytes,
		)
	}

	data, err := os.ReadFile(args.FilePath)
	if err != nil {
		return "", fmt.Errorf("read file failed: %w", err)
	}

	return string(data), nil
}

/**********************************
*
* Agent Runtime
*
**********************************/

type Agent struct {
	tools map[string]Tool
}

func NewAgent() *Agent {
	return &Agent{
		tools: make(map[string]Tool),
	}
}

func (a *Agent) RegisterTool(tool Tool) {
	a.tools[tool.Name()] = tool
}

func (a *Agent) CallTool(
	ctx context.Context,
	toolName string,
	arguments string,
) (string, error) {

	tool, ok := a.tools[toolName]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", toolName)
	}

	start := time.Now()

	log.Printf(
		"[Agent] calling tool=%s arguments=%s",
		toolName,
		arguments,
	)

	result, err := tool.Run(ctx, arguments)

	cost := time.Since(start)

	if err != nil {
		log.Printf(
			"[Agent] tool failed tool=%s cost=%s err=%v",
			toolName,
			cost,
			err,
		)

		return "", err
	}

	log.Printf(
		"[Agent] tool success tool=%s cost=%s",
		toolName,
		cost,
	)

	return result, nil
}

/**********************************
*
* main
*
**********************************/

func main() {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	agent := NewAgent()

	agent.RegisterTool(&SearchFileTool{})
	agent.RegisterTool(&ReadFileTool{})

	// 示例：搜索文件
	searchArgs := `{
		"root_dir": "./",
		"file_name": "main.go"
	}`

	searchResult, err := agent.CallTool(
		ctx,
		"search_file",
		searchArgs,
	)
	if err != nil {
		log.Fatalf("search file failed: %v", err)
	}

	fmt.Println("Search Result:")
	fmt.Println(searchResult)

	// 示例：读取文件
	readArgs := `{
		"file_path": "./main.go",
		"max_bytes": 65536
	}`

	readResult, err := agent.CallTool(
		ctx,
		"read_file",
		readArgs,
	)
	if err != nil {
		log.Fatalf("read file failed: %v", err)
	}

	fmt.Println("Read Result:")
	fmt.Println(readResult)
}
