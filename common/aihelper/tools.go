package aihelper

import (
	"GopherAI/config"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type GetCurrentTimeTool struct{}

func (t *GetCurrentTimeTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "get_current_time",
		Desc:        "获取当前时间",
		ParamsOneOf: schema.NewParamsOneOfByParams(nil),
		//声明参数类型
	}, nil
}

func (t *GetCurrentTimeTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	local, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return "", err
	}
	return time.Now().In(local).Format("2006-01-02 15:04:05"), nil
}

type ReadFileTool struct{}

type ReadFileParams struct {
	FileName string `json:"file_name" desc:"要读取的文件名" required:"true"`
}

func (t *ReadFileTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "read_file",
		Desc: "根据文件名读取文件内容",
		ParamsOneOf: schema.NewParamsOneOfByParams(
			map[string]*schema.ParameterInfo{
				"file_name": {
					Type:     schema.String,
					Desc:     "需要读取的文件名",
					Required: true,
				},
			},
		),
	}, nil
}

func (t *ReadFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args ReadFileParams
	err := json.Unmarshal([]byte(argumentsInJSON), &args)
	if err != nil {
		return "", err
	}
	cfg := config.GetConfig()
	filePath := filepath.Join(cfg.FileConfig.BasePath, args.FileName)
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	contentBytes, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(contentBytes), nil
}

type SearchFileTool struct{}

type SearchFileParams struct {
	FileName string `json:"file_name" desc:"要搜索的文件名" required:"true"`
}

func (t *SearchFileTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "search_file",
		Desc: "根据文件名在工作目录下搜索指定文件，并返回文件路径",
		ParamsOneOf: schema.NewParamsOneOfByParams(
			map[string]*schema.ParameterInfo{
				"file_name": {
					Type:     schema.String,
					Desc:     "需要搜索的文件名",
					Required: true,
				},
			},
		),
	}, nil
}

func (t *SearchFileTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args SearchFileParams
	err := json.Unmarshal([]byte(argumentsInJSON), &args)
	if err != nil {
		return "", err
	}
	cfg := config.GetConfig()
	var foundPath string
	err = filepath.Walk(cfg.FileConfig.BasePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == args.FileName {
			foundPath = path
			return io.EOF
		}
		return nil
	})
	if err != nil && err != io.EOF {
		return "", err
	}
	if foundPath == "" {
		return "未找到文件", nil
	}
	return foundPath, nil
}
