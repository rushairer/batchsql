-- SQLite 初始化脚本
-- SQLite 特性：轻量级、嵌入式、无服务器架构

-- 启用 WAL 模式以提高并发性能
PRAGMA journal_mode = WAL;

-- 优化 SQLite 性能配置
PRAGMA synchronous = NORMAL;        -- 平衡性能和安全性
PRAGMA cache_size = 10000;          -- 10MB 缓存
PRAGMA temp_store = MEMORY;         -- 临时表存储在内存中
PRAGMA mmap_size = 268435456;       -- 256MB 内存映射
PRAGMA optimize;                    -- 启用查询优化器

-- 创建测试表
CREATE TABLE IF NOT EXISTS integration_test (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    data TEXT,
    value REAL,
    is_active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_integration_test_name ON integration_test(name);
CREATE INDEX IF NOT EXISTS idx_integration_test_email ON integration_test(email);
CREATE INDEX IF NOT EXISTS idx_integration_test_created_at ON integration_test(created_at);

-- 创建性能测试表
CREATE TABLE IF NOT EXISTS performance_test (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    batch_id TEXT NOT NULL,
    worker_id INTEGER NOT NULL,
    record_data TEXT,  -- SQLite 没有原生 JSON 类型，使用 TEXT 存储 JSON
    large_text TEXT,
    numeric_value REAL,
    timestamp_field DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 创建性能测试表索引
CREATE INDEX IF NOT EXISTS idx_performance_test_batch_id ON performance_test(batch_id);
CREATE INDEX IF NOT EXISTS idx_performance_test_worker_id ON performance_test(worker_id);
CREATE INDEX IF NOT EXISTS idx_performance_test_timestamp ON performance_test(timestamp_field);

-- 创建内存测试表（大量列）
CREATE TABLE IF NOT EXISTS memory_test (
    id INTEGER PRIMARY KEY,
    col_1 TEXT, col_2 TEXT, col_3 TEXT, col_4 TEXT, col_5 TEXT,
    col_6 TEXT, col_7 TEXT, col_8 TEXT, col_9 TEXT, col_10 TEXT,
    col_11 TEXT, col_12 TEXT, col_13 TEXT, col_14 TEXT, col_15 TEXT,
    num_1 REAL, num_2 REAL, num_3 REAL, num_4 REAL, num_5 REAL,
    bool_1 INTEGER, bool_2 INTEGER, bool_3 INTEGER, bool_4 INTEGER, bool_5 INTEGER,
    date_1 DATETIME, date_2 DATETIME, date_3 DATETIME, date_4 DATETIME, date_5 DATETIME
);

-- 创建复合索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_memory_test_composite ON memory_test(col_1, col_2, num_1);

-- 创建用于监控的视图（SQLite 版本）
CREATE VIEW IF NOT EXISTS test_stats AS
SELECT 
    name as table_name,
    sql as table_definition
FROM sqlite_master 
WHERE type = 'table' 
AND name NOT LIKE 'sqlite_%';

-- 清理现有测试数据
DELETE FROM integration_test;
DELETE FROM performance_test;
DELETE FROM memory_test;

-- 重置自增序列
DELETE FROM sqlite_sequence WHERE name IN ('performance_test');

-- 执行 VACUUM 以优化数据库
VACUUM;

-- 分析表以更新统计信息
ANALYZE integration_test;
ANALYZE performance_test;
ANALYZE memory_test;

-- 显示初始化信息
SELECT 'SQLite Integration Test Environment Initialized' as status;
SELECT sqlite_version() as sqlite_version;

-- 显示当前配置
PRAGMA journal_mode;
PRAGMA synchronous;
PRAGMA cache_size;
PRAGMA temp_store;

-- 显示表信息
SELECT 
    name as table_name,
    CASE 
        WHEN name = 'integration_test' THEN 'Main test table for basic operations'
        WHEN name = 'performance_test' THEN 'Performance and throughput testing'
        WHEN name = 'memory_test' THEN 'Memory pressure and large column testing'
        ELSE 'Other table'
    END as description
FROM sqlite_master 
WHERE type = 'table' 
AND name NOT LIKE 'sqlite_%'
ORDER BY name;