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
	input = normalizeWorkFilePath(input)
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

// normalizeWorkFilePath 统一 work 目录下文件的路径表示。
// search_files 返回 work/TempT 时，edit_file 根目录已是 work/，需去掉多余前缀。
func normalizeWorkFilePath(input string) string {
	input = strings.TrimSpace(strings.ReplaceAll(input, "\\", "/"))
	for strings.HasPrefix(input, "./") {
		input = strings.TrimPrefix(input, "./")
	}
	if strings.HasPrefix(strings.ToLower(input), "work/") {
		return input[len("work/"):]
	}
	return input
}

func resolveReadableFilePath(input string) (string, error) {
	input = strings.TrimSpace(strings.ReplaceAll(input, "\\", "/"))
	if input == "" {
		return "", fmt.Errorf("file_path 不能为空")
	}
	lower := strings.ToLower(input)
	if strings.HasPrefix(lower, "info/") {
		return resolveSafeFilePath(input, fileBasePath)
	}
	if path, err := resolveSafeFilePath(normalizeWorkFilePath(input), fileWorkPath); err == nil {
		if _, statErr := os.Stat(path); statErr == nil {
			return path, nil
		}
	}
	return resolveSafeFilePath(input, fileBasePath)
}

func walkSearchRoots() ([]struct {
	root    string
	relBase string
}, error) {
	work, err := fileWorkPath()
	if err != nil {
		return nil, err
	}
	roots := []struct {
		root    string
		relBase string
	}{
		{root: work, relBase: "work"},
	}
	base, err := fileBasePath()
	if err != nil {
		return nil, err
	}
	infoDir := filepath.Join(base, "Info")
	if st, statErr := os.Stat(infoDir); statErr == nil && st.IsDir() {
		roots = append(roots, struct {
			root    string
			relBase string
		}{root: infoDir, relBase: "Info"})
	}
	return roots, nil
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
		Desc:        "获取现在的时间",
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
		Desc: "在 work 工作目录与 Info 知识库目录中搜索文件（匹配文件名或文本内容），返回 path 列表。work 下文件 path 形如 work/TempT；Info 下形如 Info/巴别塔.txt。read_file 与 edit_file 必须原样使用该 path。",
		ParamsOneOf: schema.NewParamsOneOfByParams(
			map[string]*schema.ParameterInfo{
				"keyword": {
					Type:     schema.String,
					Desc:     "搜索关键词，例如 TempT、note.txt 或内容关键词",
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

	roots, err := walkSearchRoots()
	if err != nil {
		return toolNoRetry(err.Error()), nil
	}

	lowerKeyword := strings.ToLower(keyword)
	type fileItem struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	matches := make([]fileItem, 0)
	seen := make(map[string]bool)

	for _, root := range roots {
		walkErr := filepath.Walk(root.root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}
			if !isSearchableTextFile(info.Name()) {
				return nil
			}

			rel, relErr := filepath.Rel(root.root, path)
			if relErr != nil {
				return nil
			}
			displayPath := filepath.ToSlash(filepath.Join(root.relBase, rel))

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
			if seen[displayPath] {
				return nil
			}
			seen[displayPath] = true
			matches = append(matches, fileItem{
				Name: info.Name(),
				Path: displayPath,
			})
			return nil
		})
		if walkErr != nil {
			return toolNoRetry(fmt.Sprintf("搜索失败: %v", walkErr)), nil
		}
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
		"message": "请从 files 中选择一项，将其 path 原样传给 read_file 或 edit_file",
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
					Desc:     "search_files 返回的 path 原样传入，例如 work/TempT 或 Info/example.txt",
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

	path, err := resolveReadableFilePath(args.FilePath)
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
	if st, err := os.Stat(newPath); err == nil {
		rel, _ := filepath.Rel(base, newPath)
		if st.Size() == 0 {
			return toolJSONResult(map[string]interface{}{
				"success": true,
				"path":    filepath.ToSlash(rel),
				"message": "文件已存在且为空，请使用 edit_file 写入内容",
			}), nil
		}
		return toolNoRetry("文件已存在且非空，请更换 title 或使用 edit_file 修改"), nil
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
		Desc: "向 work 目录下的文件覆盖写入 content。file_path 使用 search_files 返回的 path（如 work/TempT），或 create_file 返回的 path（如 TempT）。禁止写入 Info 目录。",
		ParamsOneOf: schema.NewParamsOneOfByParams(
			map[string]*schema.ParameterInfo{
				"file_path": {
					Type:     schema.String,
					Desc:     "search_files 的 path（如 work/TempT）或 create_file 的 path（如 note.txt）",
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
	err = os.WriteFile(path, []byte(args.Content), 0644)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("写入文件失败: %v。不要重试 edit_file，请检查路径是否正确", err)), nil
	}
	return toolJSONResult(map[string]interface{}{
		"success": true,
		"path":    filepath.ToSlash(args.FilePath),
	}), nil
}

type FormatGoCodeTool struct{}

type FormatGoCodeParams struct {
	Code string `json:"code"`
}

func (t *FormatGoCodeTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "format_go_code",
		Desc: "使用 gofmt 格式化 Go 代码，参数 code 是待格式化的 Go 代码字符串，返回格式化后的代码字符串，只应该对.go后缀的go文件使用该工具，禁止对非 Go 代码使用",
		ParamsOneOf: schema.NewParamsOneOfByParams(
			map[string]*schema.ParameterInfo{
				"code": {
					Type:     schema.String,
					Desc:     "待格式化的 Go 代码字符串",
					Required: true,
				},
			},
		),
	}, nil
}

func (t *FormatGoCodeTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args FormatGoCodeParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return toolNoRetry("参数解析失败，请检查 code 字段"), nil
	}
	formatted, err := GoFmt(args.Code)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("代码格式化失败: %v。请确保只对 Go 代码使用 format_go_code 工具", err)), nil
	}
	return formatted, nil
}

type RenameFileTool struct{}

type RenameFileParams struct {
	OldPath  string `json:"old_path"`
	NewTitle string `json:"new_title"`
}

func (t *RenameFileTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "rename_file",
		Desc: "重命名 work 目录下的文件。old_path 使用 search_files 返回的 path（如 work/TempT）。new_title 仅作为新的文件名使用，禁止包含路径分隔符，文件会被重命名在原目录下。",
		ParamsOneOf: schema.NewParamsOneOfByParams(
			map[string]*schema.ParameterInfo{
				"old_path": {
					Type:     schema.String,
					Desc:     "search_files 的 path，例如 work/TempT",
					Required: true,
				},
				"new_title": {
					Type:     schema.String,
					Desc:     "新的文件标题，例如 NewName.txt，禁止包含路径分隔符",
					Required: true,
				},
			},
		),
	}, nil
}

func (t *RenameFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args RenameFileParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return toolNoRetry("参数解析失败，请检查 old_path 和 new_title 字段"), nil
	}
	oldPath, err := resolveSafeFilePath(args.OldPath, fileWorkPath)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("旧路径无效: %v。不要重试 rename_file，请重新调用 search_files 获取真实路径", err)), nil
	}
	newTitle := strings.TrimSpace(args.NewTitle)
	if newTitle == "" {
		return toolNoRetry("new_title 不能为空"), nil
	}
	if strings.ContainsAny(newTitle, `\/`) {
		return toolNoRetry("new_title 不能包含路径分隔符"), nil
	}
	base, err := fileWorkPath()
	if err != nil {
		return toolNoRetry(err.Error()), nil
	}
	newPath := filepath.Join(base, newTitle)
	if st, err := os.Stat(newPath); err == nil {
		if st.IsDir() {
			return toolNoRetry("目标文件已存在且是目录，请更换 new_title 或使用 edit_file 修改内容"), nil
		}
		return toolNoRetry("目标文件已存在且非空，请更换 new_title 或使用 edit_file 修改"), nil
	} else if !os.IsNotExist(err) {
		return toolNoRetry(fmt.Sprintf("无法访问文件系统: %v", err)), nil
	}
	err = os.Rename(oldPath, newPath)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("重命名文件失败: %v。不要重试 rename_file，请检查路径是否正确", err)), nil
	}
	rel, _ := filepath.Rel(base, newPath)
	return toolJSONResult(map[string]interface{}{
		"success": true,
		"path":    filepath.ToSlash(rel),
	}), nil
}
