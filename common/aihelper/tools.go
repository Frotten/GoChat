package aihelper

import (
	"GopherAI/service/tools"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

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
	return tools.SearchFile(args.Keyword)
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
	return tools.ReadFile(args.FilePath)
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
	return tools.CreateFile(args.Title)
}

type EditFileTool struct{}

type EditFileParams struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

func (t *EditFileTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "edit_file",
		Desc: "编辑work目录下的文件内容，将文件内容替换为Content。file_path 使用 search_files 返回的 path（如 work/TempT），或 create_file 返回的 path（如 TempT）。禁止写入 Info 目录。",
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
	return tools.EditFile(args.FilePath, args.Content)
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
	return tools.GoFmtCode(args.Code)
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
	return tools.RenameFile(args.OldPath, args.NewTitle)
}

type WebSearchTool struct{}

type WebSearchParams struct {
	Query string `json:"query"`
}

func (t *WebSearchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "web_search",
		Desc: "在网络上搜索信息，返回搜索结果列表。query 是搜索关键词,代指你要搜索的内容",
		ParamsOneOf: schema.NewParamsOneOfByParams(
			map[string]*schema.ParameterInfo{
				"query": {
					Type:     schema.String,
					Desc:     "搜索关键词",
					Required: true,
				},
			},
		),
	}, nil
}

func (t *WebSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args WebSearchParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return toolNoRetry("参数解析失败，请检查 query 字段"), nil
	}
	Ans, err := tools.TavilySearch(ctx, args.Query)
	if err != nil {
		return toolNoRetry(fmt.Sprintf("网络搜索失败: %v", err)), nil
	}
	return Ans, nil
}
