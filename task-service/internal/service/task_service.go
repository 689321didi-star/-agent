package service

import (
	"context"
	"fmt"
	"time"

	"github.com/yourname/ai-learning-platform/task-service/internal/model"
	"github.com/yourname/ai-learning-platform/task-service/internal/repository"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type TaskService interface {
	CreateTask(ctx context.Context, params CreateTaskParams) (*model.Task, error)
	GetTask(ctx context.Context, taskID string, userID uint64) (*model.Task, error)
	ListTasks(ctx context.Context, userID uint64) ([]*model.Task, error)
	CancelTask(ctx context.Context, taskID string, userID uint64) error
}

type CreateTaskParams struct {
	UserID       uint64
	Title        string
	Goal         string
	Backgrounds  []string
	DeadlineDays int
}

type taskService struct {
	repo repository.TaskRepository
	rdb  *redis.Client
	log  *zap.Logger
}

func NewTaskService(repo repository.TaskRepository, rdb *redis.Client, log *zap.Logger) TaskService {
	return &taskService{repo: repo, rdb: rdb, log: log}
}

func (s *taskService) CreateTask(ctx context.Context, params CreateTaskParams) (*model.Task, error) {
	// 1. 构建任务
	deadline := time.Now().AddDate(0, 0, params.DeadlineDays)
	task := &model.Task{
		TaskID:     uuid.New().String(),
		UserID:     params.UserID,
		Title:      params.Title,
		Goal:       params.Goal,
		Status:     model.TaskStatusPending,
		Priority:   5,
		MaxRetries: 3,
		Deadline:   &deadline,
	}

	// 2. 写入 MySQL（持久化）
	if err := s.repo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create task in db: %w", err)
	}

	// 3. 推送到 Redis 任务队列（供 Scheduler 消费）
	if err := s.pushToQueue(ctx, task); err != nil {
		// 队列失败不影响主流程，记录日志等待补偿
		s.log.Warn("push task to queue failed",
			zap.String("task_id", task.TaskID),
			zap.Error(err),
		)
	}

	s.log.Info("task created",
		zap.String("task_id", task.TaskID),
		zap.Uint64("user_id", params.UserID),
	)

	return task, nil
}

func (s *taskService) GetTask(ctx context.Context, taskID string, userID uint64) (*model.Task, error) {
	// 先查 Redis 缓存
	// TODO: cache-aside pattern
	return s.repo.FindByTaskID(ctx, taskID, userID)
}

func (s *taskService) ListTasks(ctx context.Context, userID uint64) ([]*model.Task, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *taskService) CancelTask(ctx context.Context, taskID string, userID uint64) error {
	task, err := s.repo.FindByTaskID(ctx, taskID, userID)
	if err != nil {
		return fmt.Errorf("task not found")
	}

	// 只有 Pending 状态可以取消
	if task.Status != model.TaskStatusPending {
		return fmt.Errorf("only pending task can be cancelled, current: %s", task.Status)
	}

	return s.repo.UpdateStatus(ctx, taskID, model.TaskStatusFailed)
}

// pushToQueue 将任务推入 Redis 队列
func (s *taskService) pushToQueue(ctx context.Context, task *model.Task) error {
	// 用 Redis List 做任务队列，LPUSH + BRPOP
	// key 按优先级分层，Scheduler 优先消费高优先级队列
	queueKey := fmt.Sprintf("task_queue:priority:%d", task.Priority)
	return s.rdb.LPush(ctx, queueKey, task.TaskID).Err()
}
