package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourname/ai-learning-platform/task-service/internal/handler"
	"github.com/yourname/ai-learning-platform/task-service/internal/middleware"
	"github.com/yourname/ai-learning-platform/task-service/internal/repository"
	"github.com/yourname/ai-learning-platform/task-service/internal/service"
	"github.com/yourname/ai-learning-platform/task-service/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// ── 初始化日志 ──────────────────────────────────────────
	log := logger.New()
	defer log.Sync()

	// ── 初始化依赖（DB / Redis / Kafka） ────────────────────
	db := repository.NewMySQL()
	rdb := repository.NewRedis()
	// kafkaProducer := repository.NewKafkaProducer()

	// ── 注入依赖 ─────────────────────────────────────────────
	taskRepo := repository.NewTaskRepo(db)
	taskSvc := service.NewTaskService(taskRepo, rdb, log)
	taskHandler := handler.NewTaskHandler(taskSvc, log)

	// ── 路由 ─────────────────────────────────────────────────
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger(log))
	r.Use(middleware.Tracing())
	r.Use(middleware.RateLimit(rdb)) // 令牌桶限流

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Prometheus 指标
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API 路由组
	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("")
		auth.Use(middleware.JWT())
		{
			auth.POST("/tasks", taskHandler.CreateTask)
			auth.GET("/tasks/:task_id", taskHandler.GetTask)
			auth.GET("/tasks", taskHandler.ListTasks)
			auth.DELETE("/tasks/:task_id", taskHandler.CancelTask)
		}

		// 不需要鉴权
		v1.POST("/auth/register", taskHandler.Register)
		v1.POST("/auth/login", taskHandler.Login)
	}

	// WebSocket（任务进度推送）
	r.GET("/ws/tasks/:task_id", taskHandler.TaskProgress)

	// ── 优雅关闭 ─────────────────────────────────────────────
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		log.Info("task-service started on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
