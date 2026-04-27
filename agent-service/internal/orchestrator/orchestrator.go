package orchestrator

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
	"go.uber.org/zap"
)

// AgentResult 单个 Agent 的执行结果
type AgentResult struct {
	AgentType string
	Data      interface{}
	Duration  time.Duration
}

// TaskInput 编排器接收的任务输入
type TaskInput struct {
	TaskID      string
	Goal        string
	UserProfile UserProfile
	Deadline    time.Time
}

type UserProfile struct {
	Backgrounds []string
	WeakPoints  []string
	PreferFmt   []string
}

// Agent 接口，所有 Agent 实现此接口
type Agent interface {
	Name() string
	Run(ctx context.Context, input TaskInput) (interface{}, error)
}

// Orchestrator 任务编排器
// ─────────────────────────────────────────────────────────────
// 面试考点：fan-out / fan-in 并发模式
//   fan-out：把一个任务拆分给多个 Agent 并行执行
//   fan-in：收集所有 Agent 结果，聚合返回
//
//   关键：errgroup + context
//   → 任意一个 Agent 失败，cancel 其他所有 Agent
//   → context 超时自动级联取消

type Orchestrator struct {
	searchAgent Agent
	genAgent    Agent
	evalAgent   Agent
	log         *zap.Logger
}

func New(search, gen, eval Agent, log *zap.Logger) *Orchestrator {
	return &Orchestrator{
		searchAgent: search,
		genAgent:    gen,
		evalAgent:   eval,
		log:         log,
	}
}

// Run 执行完整的多 Agent 流程
func (o *Orchestrator) Run(ctx context.Context, input TaskInput) (map[string]interface{}, error) {
	o.log.Info("orchestrator started", zap.String("task_id", input.TaskID))

	// ── Phase 1: Search + Generate 并发执行 ──────────────────
	// 这两个 Agent 相互独立，可以并发
	results := make(map[string]interface{})
	var mu = &syncMap{m: results}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return o.runAgent(gCtx, o.searchAgent, input, mu)
	})

	g.Go(func() error {
		return o.runAgent(gCtx, o.genAgent, input, mu)
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("phase1 failed: %w", err)
	}

	// ── Phase 2: Eval Agent（依赖 Phase1 结果） ───────────────
	// Eval 需要 Search 和 Generate 的结果，所以串行在后面
	if err := o.runAgent(ctx, o.evalAgent, input, mu); err != nil {
		return nil, fmt.Errorf("eval failed: %w", err)
	}

	o.log.Info("orchestrator done", zap.String("task_id", input.TaskID))
	return results, nil
}

// runAgent 运行单个 Agent，带耗时记录
func (o *Orchestrator) runAgent(ctx context.Context, agent Agent, input TaskInput, result *syncMap) error {
	start := time.Now()

	data, err := agent.Run(ctx, input)
	if err != nil {
		o.log.Error("agent failed",
			zap.String("agent", agent.Name()),
			zap.String("task_id", input.TaskID),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)
		return fmt.Errorf("agent %s: %w", agent.Name(), err)
	}

	result.set(agent.Name(), data)
	o.log.Info("agent done",
		zap.String("agent", agent.Name()),
		zap.String("task_id", input.TaskID),
		zap.Duration("duration", time.Since(start)),
	)
	return nil
}

// syncMap 并发安全的结果收集
import "sync"

type syncMap struct {
	mu sync.Mutex
	m  map[string]interface{}
}

func (s *syncMap) set(key string, val interface{}) {
	s.mu.Lock()
	s.m[key] = val
	s.mu.Unlock()
}
