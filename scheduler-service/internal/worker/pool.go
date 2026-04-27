package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Task 调度任务
type Task struct {
	ID       string
	UserID   uint64
	Priority int8
	Payload  []byte
	// 超时控制
	Timeout  time.Duration
	// 重试信息
	RetryCount int
	MaxRetries int
}

// Worker Pool
// ─────────────────────────────────────────────────────────────
// 面试考点：
//   1. 为什么不直接 go func()？
//      → 无限创建 goroutine 会导致内存爆炸和调度器压力剧增
//   2. buffered channel 的大小怎么定？
//      → 根据压测结果，通常是 maxWorkers 的 2-4 倍
//   3. 如何优雅关闭？
//      → close(taskCh) 触发 worker 退出，WaitGroup 等待全部完成

type Pool struct {
	taskCh     chan Task
	maxWorkers int
	wg         sync.WaitGroup
	quit       chan struct{}
	log        *zap.Logger
	processor  TaskProcessor
}

// TaskProcessor 实际执行任务的接口，方便测试 mock
type TaskProcessor interface {
	Process(ctx context.Context, task Task) error
}

func NewPool(maxWorkers int, bufferSize int, processor TaskProcessor, log *zap.Logger) *Pool {
	return &Pool{
		taskCh:     make(chan Task, bufferSize),
		maxWorkers: maxWorkers,
		quit:       make(chan struct{}),
		log:        log,
		processor:  processor,
	}
}

// Start 启动固定数量的 worker goroutine
func (p *Pool) Start() {
	for i := 0; i < p.maxWorkers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	p.log.Info("worker pool started", zap.Int("workers", p.maxWorkers))
}

// Submit 提交任务到队列（非阻塞）
func (p *Pool) Submit(task Task) error {
	select {
	case p.taskCh <- task:
		return nil
	case <-p.quit:
		return fmt.Errorf("pool is shutting down")
	default:
		// 队列满了，拒绝并返回错误，由调用方决定是否重试
		return fmt.Errorf("task queue is full, task_id=%s", task.ID)
	}
}

// Shutdown 优雅关闭：等待正在执行的任务完成
func (p *Pool) Shutdown(ctx context.Context) error {
	p.log.Info("shutting down worker pool...")
	close(p.quit)
	close(p.taskCh)

	// 等待所有 worker 完成，或超时
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.log.Info("worker pool stopped gracefully")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout")
	}
}

// worker 单个 worker goroutine
func (p *Pool) worker(id int) {
	defer p.wg.Done()
	p.log.Debug("worker started", zap.Int("worker_id", id))

	for task := range p.taskCh {
		p.executeTask(id, task)
	}

	p.log.Debug("worker stopped", zap.Int("worker_id", id))
}

// executeTask 执行单个任务，带超时控制
func (p *Pool) executeTask(workerID int, task Task) {
	timeout := task.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute // 默认超时
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	p.log.Info("task started",
		zap.String("task_id", task.ID),
		zap.Int("worker_id", workerID),
	)

	err := p.processor.Process(ctx, task)

	duration := time.Since(start)
	if err != nil {
		p.log.Error("task failed",
			zap.String("task_id", task.ID),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		// TODO: 更新任务状态为 Failed/Retry，通知 FSM
		return
	}

	p.log.Info("task done",
		zap.String("task_id", task.ID),
		zap.Duration("duration", duration),
	)
}
