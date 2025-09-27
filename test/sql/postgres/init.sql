-- PostgreSQL 初始化脚本

-- 创建测试表
CREATE TABLE IF NOT EXISTS integration_test (
    id BIGINT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    data TEXT,
    value DECIMAL(10,2),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_integration_test_name ON integration_test(name);
CREATE INDEX IF NOT EXISTS idx_integration_test_email ON integration_test(email);
CREATE INDEX IF NOT EXISTS idx_integration_test_created_at ON integration_test(created_at);

-- 创建性能测试表
CREATE TABLE IF NOT EXISTS performance_test (
    id BIGSERIAL PRIMARY KEY,
    batch_id VARCHAR(50) NOT NULL,
    worker_id INTEGER NOT NULL,
    record_data JSONB,
    large_text TEXT,
    numeric_value DECIMAL(15,4),
    timestamp_field TIMESTAMP(6) DEFAULT CURRENT_TIMESTAMP
);

-- 创建性能测试表索引
CREATE INDEX IF NOT EXISTS idx_performance_test_batch_id ON performance_test(batch_id);
CREATE INDEX IF NOT EXISTS idx_performance_test_worker_id ON performance_test(worker_id);
CREATE INDEX IF NOT EXISTS idx_performance_test_timestamp ON performance_test(timestamp_field);
CREATE INDEX IF NOT EXISTS idx_performance_test_record_data ON performance_test USING GIN(record_data);

-- 创建内存测试表（大量列）
CREATE TABLE IF NOT EXISTS memory_test (
    id BIGINT PRIMARY KEY,
    col_1 VARCHAR(100), col_2 VARCHAR(100), col_3 VARCHAR(100), col_4 VARCHAR(100), col_5 VARCHAR(100),
    col_6 VARCHAR(100), col_7 VARCHAR(100), col_8 VARCHAR(100), col_9 VARCHAR(100), col_10 VARCHAR(100),
    col_11 TEXT, col_12 TEXT, col_13 TEXT, col_14 TEXT, col_15 TEXT,
    num_1 DECIMAL(10,2), num_2 DECIMAL(10,2), num_3 DECIMAL(10,2), num_4 DECIMAL(10,2), num_5 DECIMAL(10,2),
    bool_1 BOOLEAN, bool_2 BOOLEAN, bool_3 BOOLEAN, bool_4 BOOLEAN, bool_5 BOOLEAN,
    date_1 TIMESTAMP, date_2 TIMESTAMP, date_3 TIMESTAMP, date_4 TIMESTAMP, date_5 TIMESTAMP
);

-- 优化 PostgreSQL 配置
ALTER SYSTEM SET shared_buffers = '128MB';
ALTER SYSTEM SET effective_cache_size = '256MB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;

-- 重新加载配置
SELECT pg_reload_conf();

-- 创建用于监控的视图
CREATE OR REPLACE VIEW test_stats AS
SELECT 
    schemaname,
    relname as tablename,
    n_tup_ins as inserts,
    n_tup_upd as updates,
    n_tup_del as deletes,
    n_live_tup as live_tuples,
    n_dead_tup as dead_tuples,
    last_vacuum,
    last_autovacuum,
    last_analyze,
    last_autoanalyze
FROM pg_stat_user_tables 
WHERE schemaname = 'public';

-- 显示初始化信息
SELECT 'PostgreSQL Integration Test Environment Initialized' as status;
SELECT version() as postgresql_version;
SHOW shared_buffers;
SHOW max_connections;

-- 清理现有测试数据
TRUNCATE TABLE integration_test;
TRUNCATE TABLE performance_test RESTART IDENTITY;
TRUNCATE TABLE memory_test;

-- 分析表以更新统计信息
ANALYZE integration_test;
ANALYZE performance_test;
ANALYZE memory_test;