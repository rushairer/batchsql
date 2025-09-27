# BatchSQL 压力测试报告

## 测试环境
- 时间: 2025年 9月27日 星期六 13时11分20秒 CST
- Go 版本: go version go1.25.1 darwin/arm64
- 系统: Darwin AbendeMacBook-Pro.local 25.0.0 Darwin Kernel Version 25.0.0: Mon Aug 25 21:12:01 PDT 2025; root:xnu-12377.1.9~3/RELEASE_ARM64_T8132 arm64

## 测试场景

### 场景1: 小批量高频率
- 批次数: 100
- 每批记录数: 10
- 并发数: 5
- 总记录数: 1,000

### 场景2: 大批量低频率
- 批次数: 10
- 每批记录数: 1,000
- 并发数: 2
- 总记录数: 10,000

### 场景3: 高并发
- 批次数: 50
- 每批记录数: 100
- 并发数: 10
- 总记录数: 5,000

### 场景4: 极限测试
- 批次数: 100
- 每批记录数: 500
- 并发数: 20
- 总记录数: 50,000

## 详细结果

查看各场景的详细日志:
- 场景1: small_batch_high_freq.log
- 场景2: large_batch_low_freq.log
- 场景3: high_concurrency.log
- 场景4: extreme_test.log
