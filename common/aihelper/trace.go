package aihelper

import (
	"context"
	"sync"
)

// ToolInvocation records one tool execution requested by the agent.
type ToolInvocation struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type toolTraceContextKey struct{}

// ToolTrace is a concurrency-safe per-request tool execution trace.
type ToolTrace struct {
	mu    sync.Mutex
	calls []ToolInvocation
}

// WithToolTrace enables tool execution tracing for model calls made with ctx.
func WithToolTrace(ctx context.Context) (context.Context, *ToolTrace) {
	trace := &ToolTrace{}
	return context.WithValue(ctx, toolTraceContextKey{}, trace), trace
}

func toolTraceFromContext(ctx context.Context) *ToolTrace {
	trace, _ := ctx.Value(toolTraceContextKey{}).(*ToolTrace)
	return trace
}

func (t *ToolTrace) record(name, arguments string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.calls = append(t.calls, ToolInvocation{Name: name, Arguments: arguments})
	t.mu.Unlock()
}

// Calls returns a stable snapshot of all observed tool executions.
func (t *ToolTrace) Calls() []ToolInvocation {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]ToolInvocation, len(t.calls))
	copy(out, t.calls)
	return out
}

// RecordToolInvocation lets custom AIModel implementations participate in evaluation tracing.
func RecordToolInvocation(ctx context.Context, name, arguments string) {
	toolTraceFromContext(ctx).record(name, arguments)
}
