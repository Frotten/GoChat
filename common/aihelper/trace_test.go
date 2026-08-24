package aihelper

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func TestToolTraceCallbackPreservesModelCallOrder(t *testing.T) {
	ctx, trace := WithToolTrace(context.Background())
	handler := toolTraceCallback(ctx)
	handler.OnStart(ctx, &callbacks.RunInfo{Component: compose.ComponentOfToolsNode}, &schema.Message{
		ToolCalls: []schema.ToolCall{
			{Function: schema.FunctionCall{Name: "search_files", Arguments: `{"keyword":"x"}`}},
			{Function: schema.FunctionCall{Name: "read_file", Arguments: `{"file_path":"work/x"}`}},
		},
	})
	calls := trace.Calls()
	if len(calls) != 2 || calls[0].Name != "search_files" || calls[1].Name != "read_file" {
		t.Fatalf("unexpected trace: %+v", calls)
	}
}
