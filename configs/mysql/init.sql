-- ============================================================
-- AI Learning Platform - Database Schema
-- ============================================================

CREATE DATABASE IF NOT EXISTS ai_learning DEFAULT CHARSET utf8mb4;
USE ai_learning;

-- ── 用户表 ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    username    VARCHAR(64)  NOT NULL UNIQUE,
    email       VARCHAR(128) NOT NULL UNIQUE,
    password    VARCHAR(256) NOT NULL,
    level       TINYINT      NOT NULL DEFAULT 1 COMMENT '1=beginner 2=intermediate 3=advanced',
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at  DATETIME     DEFAULT NULL,
    INDEX idx_email (email),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── 用户画像表 ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_profiles (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id         BIGINT UNSIGNED NOT NULL UNIQUE,
    backgrounds     JSON    COMMENT '已有技术背景 ["Python","Java"]',
    strengths       JSON    COMMENT '擅长领域',
    weak_points     JSON    COMMENT '薄弱知识点',
    prefer_formats  JSON    COMMENT '偏好资源类型 ["video","docs","code"]',
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── 任务主表（按 user_id 分表预留） ──────────────────────────
CREATE TABLE IF NOT EXISTS tasks (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    task_id     VARCHAR(64)  NOT NULL UNIQUE COMMENT 'UUID',
    user_id     BIGINT UNSIGNED NOT NULL,
    title       VARCHAR(256) NOT NULL,
    goal        TEXT         NOT NULL COMMENT '用户的学习目标描述',
    status      TINYINT      NOT NULL DEFAULT 0
                COMMENT '0=Pending 1=Running 2=Done 3=Failed 4=Retry',
    priority    TINYINT      NOT NULL DEFAULT 5 COMMENT '1最高 10最低',
    retry_count TINYINT      NOT NULL DEFAULT 0,
    max_retries TINYINT      NOT NULL DEFAULT 3,
    deadline    DATETIME     DEFAULT NULL,
    result_url  VARCHAR(512) DEFAULT NULL COMMENT 'MinIO presigned URL',
    error_msg   TEXT         DEFAULT NULL,
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id),
    INDEX idx_status (status),
    INDEX idx_priority_created (priority, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── 子任务表（Agent 执行单元） ──────────────────────────────
CREATE TABLE IF NOT EXISTS sub_tasks (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    task_id     VARCHAR(64)  NOT NULL,
    agent_type  VARCHAR(32)  NOT NULL COMMENT 'search/generate/eval',
    status      TINYINT      NOT NULL DEFAULT 0,
    input       JSON         COMMENT 'Agent 输入参数',
    output      JSON         COMMENT 'Agent 输出结果',
    duration_ms INT          DEFAULT NULL COMMENT '执行耗时',
    created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_task_id (task_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── 生成资源表 ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS resources (
    id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    task_id      VARCHAR(64)  NOT NULL,
    user_id      BIGINT UNSIGNED NOT NULL,
    type         VARCHAR(32)  NOT NULL COMMENT 'roadmap/exercise/summary/reference',
    title        VARCHAR(256) NOT NULL,
    content      LONGTEXT     COMMENT '结构化内容',
    file_key     VARCHAR(512) DEFAULT NULL COMMENT 'MinIO object key',
    quality_score FLOAT       DEFAULT NULL COMMENT 'Eval Agent 打分',
    created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_task_id (task_id),
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
