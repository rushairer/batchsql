-- MySQL 初始化脚本
-- 创建测试数据库和用户（如果不存在）

-- 确保数据库存在
CREATE DATABASE IF NOT EXISTS batchsql_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 使用测试数据库
USE batchsql_test;

-- 创建测试表
CREATE TABLE IF NOT EXISTS integration_test (
    id BIGINT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    data TEXT,
    value DECIMAL(10,2),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_name (name),
    INDEX idx_email (email),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 创建性能测试表
CREATE TABLE IF NOT EXISTS performance_test (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    batch_id VARCHAR(50) NOT NULL,
    worker_id INT NOT NULL,
    record_data JSON,
    large_text LONGTEXT,
    numeric_value DECIMAL(15,4),
    timestamp_field TIMESTAMP(6) DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_batch_id (batch_id),
    INDEX idx_worker_id (worker_id),
    INDEX idx_timestamp (timestamp_field)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 创建内存测试表（大量列）
CREATE TABLE IF NOT EXISTS memory_test (
    id BIGINT PRIMARY KEY,
    col_1 VARCHAR(100), col_2 VARCHAR(100), col_3 VARCHAR(100), col_4 VARCHAR(100), col_5 VARCHAR(100),
    col_6 VARCHAR(100), col_7 VARCHAR(100), col_8 VARCHAR(100), col_9 VARCHAR(100), col_10 VARCHAR(100),
    col_11 TEXT, col_12 TEXT, col_13 TEXT, col_14 TEXT, col_15 TEXT,
    num_1 DECIMAL(10,2), num_2 DECIMAL(10,2), num_3 DECIMAL(10,2), num_4 DECIMAL(10,2), num_5 DECIMAL(10,2),
    bool_1 BOOLEAN, bool_2 BOOLEAN, bool_3 BOOLEAN, bool_4 BOOLEAN, bool_5 BOOLEAN,
    date_1 TIMESTAMP, date_2 TIMESTAMP, date_3 TIMESTAMP, date_4 TIMESTAMP, date_5 TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 优化 MySQL 配置（通过变量设置）
-- 注意：某些变量是只读的，只能在启动时设置
-- SET GLOBAL innodb_buffer_pool_size = 268435456;  -- 256MB (只读变量)
-- SET GLOBAL innodb_log_file_size = 67108864;      -- 64MB (只读变量)
SET GLOBAL innodb_flush_log_at_trx_commit = 2;   -- 性能优化
SET GLOBAL sync_binlog = 0;                      -- 测试环境性能优化

-- 显示配置信息
SELECT 'MySQL Integration Test Environment Initialized' as status;
SELECT VERSION() as mysql_version;
SHOW VARIABLES LIKE 'innodb_buffer_pool_size';
SHOW VARIABLES LIKE 'max_connections';