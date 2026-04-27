package handler

import (
	"net/http"

	"github.com/yourname/ai-learning-platform/task-service/internal/service"
	"github.com/yourname/ai-learning-platform/task-service/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TaskHandler struct {
	svc service.TaskService
	log *zap.Logger
}

func NewTaskHandler(svc service.TaskService, log *zap.Logger) *TaskHandler {
	return &TaskHandler{svc: svc, log: log}
}

// CreateTaskRequest 创建任务的请求体
type CreateTaskRequest struct {
	Title       string   `json:"title"        binding:"required,max=256"`
	Goal        string   `json:"goal"         binding:"required"`
	Backgrounds []string `json:"backgrounds"`                        // 已有技术背景
	DeadlineDays int     `json:"deadline_days" binding:"min=1,max=365"`
}

// CreateTask POST /api/v1/tasks
func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	// 从 JWT 中间件拿 userID
	userID := c.GetUint64("user_id")

	task, err := h.svc.CreateTask(c.Request.Context(), service.CreateTaskParams{
		UserID:       userID,
		Title:        req.Title,
		Goal:         req.Goal,
		Backgrounds:  req.Backgrounds,
		DeadlineDays: req.DeadlineDays,
	})
	if err != nil {
		h.log.Error("create task failed", zap.Error(err), zap.Uint64("user_id", userID))
		response.Fail(c, http.StatusInternalServerError, "create task failed")
		return
	}

	response.OK(c, task)
}

// GetTask GET /api/v1/tasks/:task_id
func (h *TaskHandler) GetTask(c *gin.Context) {
	taskID := c.Param("task_id")
	userID := c.GetUint64("user_id")

	task, err := h.svc.GetTask(c.Request.Context(), taskID, userID)
	if err != nil {
		response.Fail(c, http.StatusNotFound, "task not found")
		return
	}

	response.OK(c, task)
}

// ListTasks GET /api/v1/tasks
func (h *TaskHandler) ListTasks(c *gin.Context) {
	userID := c.GetUint64("user_id")

	tasks, err := h.svc.ListTasks(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "list tasks failed")
		return
	}

	response.OK(c, tasks)
}

// CancelTask DELETE /api/v1/tasks/:task_id
func (h *TaskHandler) CancelTask(c *gin.Context) {
	taskID := c.Param("task_id")
	userID := c.GetUint64("user_id")

	if err := h.svc.CancelTask(c.Request.Context(), taskID, userID); err != nil {
		response.Fail(c, http.StatusBadRequest, err.Error())
		return
	}

	response.OK(c, nil)
}

// Register POST /api/v1/auth/register
func (h *TaskHandler) Register(c *gin.Context) {
	// TODO: 注册逻辑
	response.OK(c, gin.H{"message": "register endpoint"})
}

// Login POST /api/v1/auth/login
func (h *TaskHandler) Login(c *gin.Context) {
	// TODO: 登录逻辑，返回 JWT
	response.OK(c, gin.H{"message": "login endpoint"})
}

// TaskProgress GET /ws/tasks/:task_id
// WebSocket 推送任务进度
func (h *TaskHandler) TaskProgress(c *gin.Context) {
	// TODO: WebSocket upgrade
	// 1. 升级 HTTP → WebSocket
	// 2. 订阅 Redis pub/sub 对应任务的进度频道
	// 3. 实时推送给客户端
	taskID := c.Param("task_id")
	h.log.Info("websocket connect", zap.String("task_id", taskID))
}
