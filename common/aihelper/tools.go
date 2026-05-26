package aihelper

import (
	"GopherAI/config"
	"context"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const maxFileReadBytes = 32 * 1024

func GoFmt(code string) (string, error) {
	buf, err := format.Source([]byte(code))
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func fileBasePath() (string, error) {
	base := strings.TrimSpace(config.GetConfig().FileConfig.BasePath)
	if base == "" {
		return "", fmt.Errorf("文件目录未配置")
	}
	return filepath.Abs(base)
}

func fileWorkPath() (string, error) {
	base := strings.TrimSpace(config.GetConfig().FileConfig.WorkPath)
	if base == "" {
		return "", fmt.Errorf("文件工作目录未配置")
	}
	return filepath.Abs(base)
}

func resolveSafeFilePath(input string, PathFunc func() (string, error)) (string, error) {
	base, err := PathFunc()
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(strings.ReplaceAll(input, "\\", "/"))
	if input == "" {
		return "", fmt.Errorf("file_path 不能为空")
	}
	var candidate string
	if filepath.IsAbs(input) {
		candidate = filepath.Clean(input)
	} else {
		candidate = filepath.Clean(filepath.Join(base, filepath.FromSlash(input)))
	}
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(base, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("不允许访问工作目录外的文件")
	}
	return abs, nil
}

func toolJSONResult(payload map[string]interface{}) string {
	data, err := json.Marshal(payload)
	if err != nil {
		return `{"success":false,"retry":false,"message":"结果序列化失败"}`
	}
	return string(data)
}

func toolNoRetry(message string) string {
	return toolJSONResult(map[string]interface{}{
		"success": false,
		"retry":   false,
		"message": message,
	})
}

func isSearchableTextFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".txt", ".md", ".json", ".yaml", ".yml", ".xml", ".csv", ".log", ".go", ".js", ".ts", ".vue", ".html", ".css":
		return true
	default:
		return ext == ""
	}
}

type GetCurrentTimeTool struct{}

func (t *GetCurrentTimeTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "get_current_time",
		Desc:        "获取当前时间",
		ParamsOneOf: schema.NewParamsOneOfByParams(nil),
	}, nil
}

func (t *GetCurrentTimeTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	local, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return "", err
	}
	return time.Now().In(local).Format("2006-01-02 15:04:05"), nil
}

type SearchFilesTool struct{}

type SearchFilesParams struct {
	Keyword string `json:"keyword"`
}

func (t *SearchFilesTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "search_files",
		Desc: "根据关键词在工作目录中搜索文件（匹配文件名或文本内容），返回真实文件列表。读取文件前必须先调用此工具，不要猜测文件名。",
		ParamsOneOf: schema.NewParamsOneOfByParams(
			map[string]*schema.ParameterInfo{
				"keyword": {
					Type:     schema.String,
					Desc:     "搜索关键词，例如文件名片段或内容关键词",
					Required: true,
				},
			},
		),
	}, nil
}

func (t *SearchFilesTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	var args SearchFilesParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return toolNoRetry("参数解析失败，请检查 keyword 字段"), nil
	}

	keyword := strings.TrimSpace(args.Keyword)
	if keyword == "" {
		return toolNoRetry("keyword 不能为空"), nil
	}

	base, err := fileBasePath()
	if err != nil {
		return toolNoRetry(err.Error()), nil
	}

	lowerKeyword := strings.ToLower(keyword)
	type fileItem struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	matches := make([]fileItem, 0)

	err = filepath.Walk(base, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		if !isSearchableTextFile(info.Name()) {
			return nil
		}

		rel, relErr := filepath.Rel(base, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		nameMatched := strings.Contains(strings.ToLower(info.Name()), lowerKeyword)
		contentMatched := false
		if !nameMatched {
			content, readErr := os.ReadFile(path)
			if readErr == nil && strings.Contains(strings.ToLower(string(content)), lowerKeyword) {
				contentMatched = true
			}
		}
		if !nameMatched && !contentMatched {
			return nil
		}

		matches = append(matches, fileItem{
			Name: info.Name(),
			Path: rel,
		})
		return nil
	})
	if err != nil {
		return toolNoRetry(fmt.Sprintf("搜索失败: %v", err)), nil
	}

	if len(matches) == 0 {
		return toolJSONResult(map[string]interface{}{
			"success": true,
			"files":   []fileItem{},
			"message": fmt.Sprintf("未找到包含关键词 %q 的文件，请勿猜测文件名，可更换关键词或告知用户", keyword),
			"retry":   false,
		}), nil
	}

	return toolJSONResult(map[string]interface{}{
		"success": true,
		"files":   matches,
		"message": "请从 files 中选择一项，将其 path 原样传给 read_file 的 file_path 参数",
		"retry":   false,
	}), nil
}

type ReadFileTool struct{}

type ReadFileParams struct {
	FilePath string `json:"file_path"`
}

func (t *ReadFileTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "read_file",
		Desc: "读取 search_files 返回列表中的 file_path，必须原样使用搜索结果中的 path，禁止猜测路径",
		ParamsOneOf: schema.NewParamsOneOfByParams(
			map[string]*schema.ParameterInfo{
				"file_path": {
					Type:     schema.String,
					Desc:     "search_files 返回的 path 字段，例如 Info/example.txt",
					Required: true,
				},
			},
		),
	}, nil
}

func (t *ReadFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	var args ReadFileParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return toolNoRetry("参数解析失败，请使用 search_files 返回的 file_path"), nil
	}

	path, err := resolveSafeFilePath(args.FilePath, fileBasePath)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("路径无效: %v。不要重试 read_file，请重新调用 search_files 获取真实路径", err)), nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return toolNoRetry(fmt.Sprintf("文件 %q 不存在。不要重试 read_file，请重新调用 search_files 或告知用户文件不存在", args.FilePath)), nil
		}
		return toolNoRetry(fmt.Sprintf("无法访问文件: %v。不要重试 read_file", err)), nil
	}
	if info.IsDir() {
		return toolNoRetry("目标是目录而非文件。不要重试 read_file，请从 search_files 结果中选择文件 path"), nil
	}

	file, err := os.Open(path)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("打开文件失败: %v。不要重试 read_file", err)), nil
	}
	defer file.Close()

	limited := io.LimitReader(file, maxFileReadBytes+1)
	contentBytes, err := io.ReadAll(limited)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("读取文件失败: %v。不要重试 read_file", err)), nil
	}

	truncated := len(contentBytes) > maxFileReadBytes
	if truncated {
		contentBytes = contentBytes[:maxFileReadBytes]
	}

	content := string(contentBytes)
	if truncated {
		content += fmt.Sprintf("\n\n[内容已截断，仅返回前 %d 字节]", maxFileReadBytes)
	}

	return toolJSONResult(map[string]interface{}{
		"success": true,
		"path":    filepath.ToSlash(args.FilePath),
		"content": content,
	}), nil
}

type CreateFileTool struct{}

type CreateFileParams struct {
	Title string `json:"title"`
}

func (t *CreateFileTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "create_file",
		Desc: "创建新文件，返回文件路径。title 仅作为文件名使用，禁止包含路径分隔符，文件会被创建在工作目录下",
		ParamsOneOf: schema.NewParamsOneOfByParams(
			map[string]*schema.ParameterInfo{
				"title": {
					Type:     schema.String,
					Desc:     "文件标题，例如 MyNote.txt，禁止包含路径分隔符",
					Required: true,
				},
			},
		),
	}, nil
}

func (t *CreateFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args CreateFileParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return toolNoRetry("参数解析失败，请检查 title 字段"), nil
	}
	title := strings.TrimSpace(args.Title)
	if title == "" {
		return toolNoRetry("title 不能为空"), nil
	}
	if strings.ContainsAny(title, `\/`) {
		return toolNoRetry("title 不能包含路径分隔符"), nil
	}
	base, err := fileWorkPath()
	if err != nil {
		return toolNoRetry(err.Error()), nil
	}

	newPath := filepath.Join(base, title)
	if _, err := os.Stat(newPath); err == nil {
		return toolNoRetry("文件已存在，请更换 title"), nil
	} else if !os.IsNotExist(err) {
		return toolNoRetry(fmt.Sprintf("无法访问文件系统: %v", err)), nil
	}

	file, err := os.Create(newPath)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("创建文件失败: %v", err)), nil
	}
	_ = file.Close()
	rel, _ := filepath.Rel(base, newPath)
	return toolJSONResult(map[string]interface{}{
		"success": true,
		"path":    filepath.ToSlash(rel),
	}), nil
}

type EditFileTool struct{}

type EditFileParams struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

func (t *EditFileTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "edit_file",
		Desc: "编辑文件，覆盖写入 content 到 file_path 指定的文件中。file_path 必须是 search_files 返回的路径或 create_file 创建的路径，禁止猜测路径",
		ParamsOneOf: schema.NewParamsOneOfByParams(
			map[string]*schema.ParameterInfo{
				"file_path": {
					Type:     schema.String,
					Desc:     "目标文件路径，例如 Info/Note.txt，必须是 search_files 返回的路径或 create_file 创建的路径",
					Required: true,
				},
				"content": {
					Type:     schema.String,
					Desc:     "要写入文件的内容",
					Required: true,
				},
			},
		),
	}, nil
}

func (t *EditFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args EditFileParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return toolNoRetry("参数解析失败，请检查 file_path 和 content 字段"), nil
	}
	path, err := resolveSafeFilePath(args.FilePath, fileWorkPath)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("路径无效: %v。不要重试 edit_file，请重新调用 search_files 或 create_file 获取真实路径", err)), nil
	}
	code, err := GoFmt(args.Content)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("代码格式化失败: %v。不要重试 edit_file，请检查 content 是否为有效 Go 代码", err)), nil
	}
	err = os.WriteFile(path, []byte(code), 0644)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("写入文件失败: %v。不要重试 edit_file，请检查路径是否正确", err)), nil
	}
	return toolJSONResult(map[string]interface{}{
		"success": true,
		"path":    filepath.ToSlash(args.FilePath),
	}), nil
}
