package aihelper

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestTaskLayerModelRunsTaskWorkflow(t *testing.T) {
	t.Setenv("GOAI_TASK_LAYER", "always")
	t.Setenv("GOAI_TASK_MAX_ROUND", "3")

	var executeCount int
	base := &fakeAIModel{
		modelType: "fake",
		generate: func(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
			latest := messages[len(messages)-1].Content
			switch {
			case strings.Contains(latest, "Plan 阶段"):
				return &schema.Message{Content: "1. 检查现状\n2. 给出结论"}, nil
			case strings.Contains(latest, "Execute 阶段"):
				executeCount++
				return &schema.Message{Content: "step done"}, nil
			case strings.Contains(latest, "Final 阶段"):
				return &schema.Message{Content: "final answer"}, nil
			default:
				return &schema.Message{Content: "direct answer"}, nil
			}
		},
		stream: func(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
			return "", nil
		},
	}

	model, err := NewTaskLayerModel(context.Background(), base)
	if err != nil {
		t.Fatalf("NewTaskLayerModel failed: %v", err)
	}

	resp, err := model.GenerateResponse(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "请帮我完成一个任务"},
	})
	if err != nil {
		t.Fatalf("GenerateResponse failed: %v", err)
	}
	if resp.Content != "final answer" {
		t.Fatalf("unexpected response: %s", resp.Content)
	}
	if executeCount != 2 {
		t.Fatalf("expected 2 execute calls, got %d", executeCount)
	}
}

func TestTaskLayerModelBypassesSimpleChatInAutoMode(t *testing.T) {
	t.Setenv("GOAI_TASK_LAYER", "auto")

	base := &fakeAIModel{
		modelType: "fake",
		generate: func(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
			latest := messages[len(messages)-1].Content
			if strings.Contains(latest, "Plan 阶段") {
				t.Fatal("simple chat should not enter task workflow")
			}
			return &schema.Message{Content: "direct answer"}, nil
		},
		stream: func(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
			return "", nil
		},
	}

	model, err := NewTaskLayerModel(context.Background(), base)
	if err != nil {
		t.Fatalf("NewTaskLayerModel failed: %v", err)
	}

	resp, err := model.GenerateResponse(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "你好"},
	})
	if err != nil {
		t.Fatalf("GenerateResponse failed: %v", err)
	}
	if resp.Content != "direct answer" {
		t.Fatalf("unexpected response: %s", resp.Content)
	}
}

func TestDetectNeedUserInput(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
	}{
		{"请提供数据库连接信息", true},
		{"step done", false},
		{"NEED_USER_INPUT: 请确认是否继续", true},
		{"", false},
	}
	for _, tc := range cases {
		if got := detectNeedUserInput(tc.input); got != tc.expected {
			t.Fatalf("detectNeedUserInput(%q) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestParseReviseDecision(t *testing.T) {
	decision, payload := parseReviseDecision("RETRY: 重新检查配置文件")
	if decision != reviseDecisionRetry || payload != "重新检查配置文件" {
		t.Fatalf("unexpected retry decision: %s %s", decision, payload)
	}

	decision, payload = parseReviseDecision("NEED_USER_INPUT: 请提供 API Key")
	if decision != reviseDecisionNeedUserInput || payload != "请提供 API Key" {
		t.Fatalf("unexpected need_user_input decision: %s %s", decision, payload)
	}

	decision, payload = parseReviseDecision("REPLAN:\n1. 读取配置\n2. 输出结果")
	if decision != reviseDecisionReplan || !strings.Contains(payload, "读取配置") {
		t.Fatalf("unexpected replan decision: %s %s", decision, payload)
	}
}

func TestTaskLayerModelStopsForNeedUserInput(t *testing.T) {
	t.Setenv("GOAI_TASK_LAYER", "always")
	t.Setenv("GOAI_TASK_MAX_ROUND", "3")

	base := &fakeAIModel{
		modelType: "fake",
		generate: func(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
			latest := messages[len(messages)-1].Content
			switch {
			case strings.Contains(latest, "Plan 阶段"):
				return &schema.Message{Content: "1. 收集必要信息"}, nil
			case strings.Contains(latest, "Execute 阶段"):
				return &schema.Message{Content: "请提供数据库连接信息"}, nil
			case strings.Contains(latest, "Final 阶段"):
				t.Fatal("need user input should skip final workflow")
				return nil, nil
			default:
				return &schema.Message{Content: "unexpected"}, nil
			}
		},
		stream: func(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
			return "", nil
		},
	}

	model, err := NewTaskLayerModel(context.Background(), base)
	if err != nil {
		t.Fatalf("NewTaskLayerModel failed: %v", err)
	}

	resp, err := model.GenerateResponse(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "请帮我连接数据库"},
	})
	if err != nil {
		t.Fatalf("GenerateResponse failed: %v", err)
	}
	if resp.Content != "请提供数据库连接信息" {
		t.Fatalf("unexpected response: %s", resp.Content)
	}
}

func TestTaskLayerModelReplansAfterRepeatedFailure(t *testing.T) {
	t.Setenv("GOAI_TASK_LAYER", "always")
	t.Setenv("GOAI_TASK_MAX_ROUND", "4")

	var executeCount int
	base := &fakeAIModel{
		modelType: "fake",
		generate: func(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
			latest := messages[len(messages)-1].Content
			switch {
			case strings.Contains(latest, "Plan 阶段"):
				return &schema.Message{Content: "1. 第一次尝试\n2. 输出结果"}, nil
			case strings.Contains(latest, "Revise 阶段"):
				return &schema.Message{Content: "REPLAN:\n1. 改用备用方案\n2. 输出结果"}, nil
			case strings.Contains(latest, "Execute 阶段"):
				executeCount++
				if executeCount == 1 {
					return nil, nil
				}
				return &schema.Message{Content: "replan step done"}, nil
			case strings.Contains(latest, "Final 阶段"):
				return &schema.Message{Content: "replan final"}, nil
			default:
				return &schema.Message{Content: "unexpected"}, nil
			}
		},
		stream: func(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
			return "", nil
		},
	}

	model, err := NewTaskLayerModel(context.Background(), base)
	if err != nil {
		t.Fatalf("NewTaskLayerModel failed: %v", err)
	}

	resp, err := model.GenerateResponse(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "请帮我修复任务流程"},
	})
	if err != nil {
		t.Fatalf("GenerateResponse failed: %v", err)
	}
	if resp.Content != "replan final" {
		t.Fatalf("unexpected response: %s", resp.Content)
	}
	if executeCount < 2 {
		t.Fatalf("expected execute after replan, got %d", executeCount)
	}
}
