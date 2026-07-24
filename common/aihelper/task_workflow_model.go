package aihelper

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	defaultTaskMaxRound = 5
	maxStepAttempts     = 2
)

type TaskLayerModel struct {
	executor     AIModel
	planRunner   compose.Runnable[*TaskState, *TaskState]
	stepRunner   compose.Runnable[*TaskState, *TaskState]
	finalRunner  compose.Runnable[*TaskState, *TaskState]
	modelType    string
	maxTaskRound int
}

type TaskState struct {
	Goal           string
	Messages       []*schema.Message
	PlanRaw        string
	Plan           []TaskStep
	CurrentStep    int
	Observations   []Observation
	Round          int
	MaxRound       int
	Done           bool
	NeedUserInput  bool
	FinalAnswer    string
	LastError      string
	LastResult     string
	LastSuccessful bool
}

type TaskStep struct {
	ID          int
	Title       string
	Description string
	Status      string
	Attempts    int
}

type Observation struct {
	StepID  int
	Round   int
	Result  string
	Success bool
	Error   string
}

func NewTaskLayerModel(ctx context.Context, executor AIModel) (*TaskLayerModel, error) {
	m := &TaskLayerModel{
		executor:     executor,
		modelType:    executor.GetModelType() + "+task",
		maxTaskRound: taskMaxRoundFromEnv(),
	}
	planRunner, err := m.buildPlanWorkflow(ctx)
	if err != nil {
		return nil, err
	}
	stepRunner, err := m.buildStepWorkflow(ctx)
	if err != nil {
		return nil, err
	}
	finalRunner, err := m.buildFinalWorkflow(ctx)
	if err != nil {
		return nil, err
	}
	m.planRunner = planRunner
	m.stepRunner = stepRunner
	m.finalRunner = finalRunner
	return m, nil
}

func (m *TaskLayerModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	messages = normalizeMessages(messages)
	if !shouldUseTaskLayer(messages) {
		return m.executor.GenerateResponse(ctx, messages)
	}
	state := newTaskState(messages, m.maxTaskRound)
	var err error
	state, err = m.planRunner.Invoke(ctx, state)
	if err != nil {
		return nil, err
	}

	for !state.Done && !state.NeedUserInput && state.Round < state.MaxRound {
		state, err = m.stepRunner.Invoke(ctx, state)
		if err != nil {
			state.LastError = err.Error()
			break
		}
		state.Round++
	}
	if !state.Done && !state.NeedUserInput && state.LastError == "" && state.Round >= state.MaxRound {
		state.LastError = "任务达到最大执行轮次，已停止继续执行"
	}

	if state.NeedUserInput {
		if strings.TrimSpace(state.FinalAnswer) == "" {
			state.FinalAnswer = strings.TrimSpace(state.LastResult)
		}
		if state.FinalAnswer == "" {
			state.FinalAnswer = "需要你补充更多信息后才能继续执行任务。"
		}
		return &schema.Message{
			Role:    schema.Assistant,
			Content: state.FinalAnswer,
		}, nil
	}

	state, err = m.finalRunner.Invoke(ctx, state)
	if err != nil {
		return nil, err
	}
	return &schema.Message{
		Role:    schema.Assistant,
		Content: state.FinalAnswer,
	}, nil
}

func (m *TaskLayerModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	resp, err := m.GenerateResponse(ctx, messages)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", nil
	}
	if resp.Content != "" {
		cb(resp.Content)
	}
	return resp.Content, nil
}

func (m *TaskLayerModel) GetModelType() string {
	return m.modelType
}

func (m *TaskLayerModel) buildPlanWorkflow(ctx context.Context) (compose.Runnable[*TaskState, *TaskState], error) {
	wf := compose.NewWorkflow[*TaskState, *TaskState]()
	wf.AddLambdaNode("plan", compose.InvokableLambda(func(ctx context.Context, state *TaskState) (*TaskState, error) {
		resp, err := m.executor.GenerateResponse(ctx, buildPlanMessages(state))
		if err != nil {
			state.LastError = err.Error()
			state.Plan = fallbackPlan()
			return state, nil
		}
		if resp != nil {
			state.PlanRaw = strings.TrimSpace(resp.Content)
		}
		state.Plan = parsePlanSteps(state.PlanRaw)
		if len(state.Plan) == 0 {
			state.Plan = fallbackPlan()
		}
		return state, nil
	})).AddInput(compose.START)
	wf.End().AddInput("plan")
	return wf.Compile(ctx, compose.WithGraphName("AIHelperTaskPlanWorkflow"))
}

func (m *TaskLayerModel) buildStepWorkflow(ctx context.Context) (compose.Runnable[*TaskState, *TaskState], error) {
	wf := compose.NewWorkflow[*TaskState, *TaskState]()
	wf.AddLambdaNode("execute", compose.InvokableLambda(func(ctx context.Context, state *TaskState) (*TaskState, error) {
		if state.CurrentStep >= len(state.Plan) {
			state.Done = true
			return state, nil
		}
		state.Plan[state.CurrentStep].Status = "running"
		state.Plan[state.CurrentStep].Attempts++

		resp, err := m.executor.GenerateResponse(ctx, buildExecuteMessages(state))
		state.LastSuccessful = err == nil
		state.LastResult = ""
		if err != nil {
			state.LastError = err.Error()
			return state, nil
		}
		state.LastError = ""
		if resp != nil {
			state.LastResult = strings.TrimSpace(resp.Content)
		}
		return state, nil
	})).AddInput(compose.START)

	wf.AddLambdaNode("observe", compose.InvokableLambda(func(ctx context.Context, state *TaskState) (*TaskState, error) {
		if state.CurrentStep >= len(state.Plan) {
			state.Done = true
			return state, nil
		}
		stepID := state.Plan[state.CurrentStep].ID
		needUserInput := detectNeedUserInput(state.LastResult)
		success := !needUserInput && state.LastSuccessful && state.LastError == "" && state.LastResult != ""
		state.Observations = append(state.Observations, Observation{
			StepID:  stepID,
			Round:   state.Round + 1,
			Result:  state.LastResult,
			Success: success,
			Error:   state.LastError,
		})
		if needUserInput {
			state.NeedUserInput = true
			state.FinalAnswer = strings.TrimSpace(state.LastResult)
		}
		return state, nil
	})).AddInput("execute")

	wf.AddLambdaNode("revise", compose.InvokableLambda(func(ctx context.Context, state *TaskState) (*TaskState, error) {
		if state.CurrentStep >= len(state.Plan) {
			state.Done = true
			return state, nil
		}
		if state.NeedUserInput {
			state.Done = true
			return state, nil
		}
		current := &state.Plan[state.CurrentStep]
		if state.LastSuccessful && state.LastError == "" && state.LastResult != "" {
			current.Status = "done"
			state.CurrentStep++
			if state.CurrentStep >= len(state.Plan) {
				state.Done = true
			}
			return state, nil
		}
		decision, revised := m.reviseAfterFailure(ctx, state)
		switch decision {
		case reviseDecisionNeedUserInput:
			state.NeedUserInput = true
			state.Done = true
			if strings.TrimSpace(state.FinalAnswer) == "" {
				state.FinalAnswer = strings.TrimSpace(revised)
			}
			if state.FinalAnswer == "" {
				state.FinalAnswer = strings.TrimSpace(state.LastResult)
			}
		case reviseDecisionReplan:
			if steps := parsePlanSteps(revised); len(steps) > 0 {
				for i := range steps {
					steps[i].ID = i + 1
				}
				state.Plan = steps
				state.PlanRaw = revised
				state.CurrentStep = 0
			} else {
				current.Status = "failed"
				state.Done = true
				if state.LastError == "" {
					state.LastError = strings.TrimSpace(revised)
				}
			}
		case reviseDecisionRetry:
			if revised != "" {
				current.Description = revised
			}
			current.Status = "pending"
		default:
			current.Status = "failed"
			state.Done = true
			if state.LastError == "" && revised != "" {
				state.LastError = revised
			}
		}
		return state, nil
	})).AddInput("observe")

	wf.End().AddInput("revise")
	return wf.Compile(ctx, compose.WithGraphName("AIHelperTaskStepWorkflow"))
}

func (m *TaskLayerModel) buildFinalWorkflow(ctx context.Context) (compose.Runnable[*TaskState, *TaskState], error) {
	wf := compose.NewWorkflow[*TaskState, *TaskState]()
	wf.AddLambdaNode("final", compose.InvokableLambda(func(ctx context.Context, state *TaskState) (*TaskState, error) {
		resp, err := m.executor.GenerateResponse(ctx, buildFinalMessages(state))
		if err != nil {
			state.FinalAnswer = fallbackFinalAnswer(state)
			return state, nil
		}
		if resp == nil || strings.TrimSpace(resp.Content) == "" {
			state.FinalAnswer = fallbackFinalAnswer(state)
			return state, nil
		}
		state.FinalAnswer = strings.TrimSpace(resp.Content)
		return state, nil
	})).AddInput(compose.START)
	wf.End().AddInput("final")
	return wf.Compile(ctx, compose.WithGraphName("AIHelperTaskFinalWorkflow"))
}

func newTaskState(messages []*schema.Message, maxRound int) *TaskState {
	if maxRound <= 0 {
		maxRound = defaultTaskMaxRound
	}
	return &TaskState{
		Goal:     latestUserContent(messages),
		Messages: messages,
		MaxRound: maxRound,
	}
}

func buildPlanMessages(state *TaskState) []*schema.Message {
	msgs := append([]*schema.Message{}, state.Messages...)
	msgs = append(msgs, &schema.Message{
		Role: schema.User,
		Content: "你现在处于 Plan 阶段。请为用户目标制定一个简洁、可执行的步骤列表。\n" +
			"要求：只输出步骤，每行一个步骤；不要调用工具；不要直接完成任务。\n\n用户目标：\n" + state.Goal,
	})
	return msgs
}

func buildExecuteMessages(state *TaskState) []*schema.Message {
	step := state.Plan[state.CurrentStep]
	msgs := append([]*schema.Message{}, state.Messages...)
	msgs = append(msgs, &schema.Message{
		Role: schema.User,
		Content: fmt.Sprintf(
			"你现在处于 Execute 阶段。只执行当前步骤，不要额外扩展其它步骤。\n\n当前步骤：%s\n\n已观察到的结果：\n%s",
			step.Description,
			formatObservations(state.Observations),
		),
	})
	return msgs
}

func buildFinalMessages(state *TaskState) []*schema.Message {
	msgs := append([]*schema.Message{}, state.Messages...)
	msgs = append(msgs, &schema.Message{
		Role: schema.User,
		Content: fmt.Sprintf(
			"你现在处于 Final 阶段。请基于执行结果给用户一个自然、简洁、真实的最终回复。\n"+
				"不要编造未完成的内容；如果有失败或限制，请明确说明。\n\n任务目标：\n%s\n\n计划：\n%s\n\n执行观察：\n%s\n\n最后错误：%s",
			state.Goal,
			formatPlan(state.Plan),
			formatObservations(state.Observations),
			state.LastError,
		),
	})
	return msgs
}

type reviseDecision string

const (
	reviseDecisionRetry         reviseDecision = "retry"
	reviseDecisionReplan        reviseDecision = "replan"
	reviseDecisionNeedUserInput reviseDecision = "need_user_input"
	reviseDecisionAbort         reviseDecision = "abort"
)

func (m *TaskLayerModel) reviseAfterFailure(ctx context.Context, state *TaskState) (reviseDecision, string) {
	current := state.Plan[state.CurrentStep]
	if current.Attempts >= maxStepAttempts {
		resp, err := m.executor.GenerateResponse(ctx, buildReviseMessages(state))
		if err != nil {
			return reviseDecisionAbort, err.Error()
		}
		raw := ""
		if resp != nil {
			raw = strings.TrimSpace(resp.Content)
		}
		decision, payload := parseReviseDecision(raw)
		if decision != reviseDecisionRetry {
			return decision, payload
		}
	}
	if current.Attempts < maxStepAttempts {
		return reviseDecisionRetry, current.Description
	}
	return reviseDecisionAbort, "步骤多次执行失败，已停止继续执行"
}

func buildReviseMessages(state *TaskState) []*schema.Message {
	step := state.Plan[state.CurrentStep]
	msgs := append([]*schema.Message{}, state.Messages...)
	msgs = append(msgs, &schema.Message{
		Role: schema.User,
		Content: fmt.Sprintf(
			"你现在处于 Revise 阶段。当前步骤执行失败，请决定下一步如何处理。\n"+
				"只输出以下格式之一：\n"+
				"RETRY: <修订后的当前步骤描述>\n"+
				"REPLAN:\n<新的完整步骤列表，每行一个步骤>\n"+
				"NEED_USER_INPUT: <需要向用户确认的问题>\n"+
				"ABORT: <无法继续的原因>\n\n"+
				"任务目标：\n%s\n\n当前计划：\n%s\n\n当前步骤：%s\n\n已尝试次数：%d\n\n最近错误：%s\n\n最近结果：%s\n\n执行观察：\n%s",
			state.Goal,
			formatPlan(state.Plan),
			step.Description,
			step.Attempts,
			state.LastError,
			state.LastResult,
			formatObservations(state.Observations),
		),
	})
	return msgs
}

func parseReviseDecision(raw string) (reviseDecision, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return reviseDecisionAbort, "修订阶段没有返回有效决策"
	}
	upper := strings.ToUpper(raw)
	switch {
	case strings.HasPrefix(upper, "RETRY:"):
		return reviseDecisionRetry, strings.TrimSpace(raw[len("RETRY:"):])
	case strings.HasPrefix(upper, "REPLAN:"):
		payload := strings.TrimSpace(raw[len("REPLAN:"):])
		if payload == "" {
			lines := strings.Split(raw, "\n")
			if len(lines) > 1 {
				payload = strings.TrimSpace(strings.Join(lines[1:], "\n"))
			}
		}
		return reviseDecisionReplan, payload
	case strings.HasPrefix(upper, "NEED_USER_INPUT:"):
		return reviseDecisionNeedUserInput, strings.TrimSpace(raw[len("NEED_USER_INPUT:"):])
	case strings.HasPrefix(upper, "ABORT:"):
		return reviseDecisionAbort, strings.TrimSpace(raw[len("ABORT:"):])
	default:
		if steps := parsePlanSteps(raw); len(steps) > 0 {
			return reviseDecisionReplan, raw
		}
		return reviseDecisionAbort, raw
	}
}

func detectNeedUserInput(result string) bool {
	result = strings.TrimSpace(result)
	if result == "" {
		return false
	}
	upper := strings.ToUpper(result)
	if strings.Contains(upper, "NEED_USER_INPUT") {
		return true
	}
	lower := strings.ToLower(result)
	hints := []string{
		"需要你提供", "需要你补充", "需要你确认", "需要你输入",
		"请提供", "请补充", "请确认", "请告诉我", "请说明",
		"能否提供", "是否可以提供", "请用户", "等待用户",
		"need your", "need user input", "please provide", "please confirm",
	}
	for _, hint := range hints {
		if strings.Contains(lower, strings.ToLower(hint)) {
			return true
		}
	}
	return false
}

func parsePlanSteps(raw string) []TaskStep {
	lines := strings.Split(raw, "\n")
	steps := make([]TaskStep, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "-*• \t")
		line = strings.TrimSpace(strings.TrimLeft(line, "0123456789.、)） "))
		if line == "" {
			continue
		}
		steps = append(steps, TaskStep{
			ID:          len(steps) + 1,
			Title:       line,
			Description: line,
			Status:      "pending",
		})
	}
	return steps
}

func fallbackPlan() []TaskStep {
	return []TaskStep{{
		ID:          1,
		Title:       "完成用户请求",
		Description: "完成用户请求",
		Status:      "pending",
	}}
}

func fallbackFinalAnswer(state *TaskState) string {
	if len(state.Observations) == 0 {
		if state.LastError != "" {
			return "任务执行失败：" + state.LastError
		}
		return "任务没有产生可用结果。"
	}
	last := state.Observations[len(state.Observations)-1]
	if last.Success {
		return last.Result
	}
	if last.Error != "" {
		return "任务执行失败：" + last.Error
	}
	return "任务没有成功完成。"
}

func formatPlan(plan []TaskStep) string {
	if len(plan) == 0 {
		return "无"
	}
	var b strings.Builder
	for _, step := range plan {
		fmt.Fprintf(&b, "%d. [%s] %s\n", step.ID, step.Status, step.Description)
	}
	return strings.TrimSpace(b.String())
}

func formatObservations(observations []Observation) string {
	if len(observations) == 0 {
		return "暂无"
	}
	var b strings.Builder
	for _, obs := range observations {
		if obs.Success {
			fmt.Fprintf(&b, "- step=%d round=%d success=true result=%s\n", obs.StepID, obs.Round, obs.Result)
		} else {
			fmt.Fprintf(&b, "- step=%d round=%d success=false error=%s result=%s\n", obs.StepID, obs.Round, obs.Error, obs.Result)
		}
	}
	return strings.TrimSpace(b.String())
}

func shouldUseTaskLayer(messages []*schema.Message) bool {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("GOAI_TASK_LAYER")))
	switch mode {
	case "off", "false", "0", "disable", "disabled":
		return false
	case "always", "true", "1", "enable", "enabled":
		return true
	}

	goal := latestUserContent(messages)
	if goal == "" {
		return false
	}
	lower := strings.ToLower(goal)
	taskHints := []string{
		"请帮", "帮我", "实现", "修改", "修复", "创建", "生成", "写入", "读取", "搜索", "查找",
		"分析", "总结", "重构", "优化", "设计", "添加", "删除", "更新", "测试", "运行",
		"implement", "modify", "fix", "create", "generate", "write", "read", "search",
		"analyze", "summarize", "refactor", "optimize", "design", "add", "delete", "update", "test", "run",
	}
	for _, hint := range taskHints {
		if strings.Contains(lower, strings.ToLower(hint)) {
			return true
		}
	}
	return false
}

func latestUserContent(messages []*schema.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i] != nil && messages[i].Role == schema.User {
			return strings.TrimSpace(messages[i].Content)
		}
	}
	return ""
}

func taskMaxRoundFromEnv() int {
	raw := strings.TrimSpace(os.Getenv("GOAI_TASK_MAX_ROUND"))
	if raw == "" {
		return defaultTaskMaxRound
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultTaskMaxRound
	}
	return n
}
