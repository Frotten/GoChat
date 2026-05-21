package aihelper

import (
	"context"
	"encoding/json"
	"io"
	"os"
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
	file, err := os.Open(args.FileName)
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
