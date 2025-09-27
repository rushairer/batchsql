# 贡献指南

感谢您对 BatchSQL 项目的关注！我们欢迎各种形式的贡献。

## 🚀 快速开始

### 开发环境设置

1. **Fork 项目**
   ```bash
   git clone https://github.com/your-username/batchsql.git
   cd batchsql
   ```

2. **安装依赖**
   ```bash
   go mod download
   ```

3. **运行测试**
   ```bash
   make test
   ```

## 📋 贡献类型

### 🐛 Bug 报告
- 使用 [Bug Report 模板](https://github.com/rushairer/batchsql/issues/new?template=bug_report.md)
- 提供详细的复现步骤
- 包含环境信息和错误日志

### ✨ 功能请求
- 使用 [Feature Request 模板](https://github.com/rushairer/batchsql/issues/new?template=feature_request.md)
- 描述使用场景和预期行为
- 考虑向后兼容性

### 🔧 代码贡献
- 遵循项目的代码规范
- 添加相应的测试用例
- 更新相关文档

## 🛠️ 开发流程

### 1. 创建分支
```bash
git checkout -b feature/your-feature-name
# 或
git checkout -b fix/your-bug-fix
```

### 2. 开发和测试
```bash
# 运行所有测试
make test

# 运行代码检查
make lint

# 运行压力测试
make test-stress
```

### 3. 提交代码
```bash
git add .
git commit -m "feat: add new feature description"
```

#### 提交信息规范
使用 [Conventional Commits](https://www.conventionalcommits.org/) 格式：

- `feat:` 新功能
- `fix:` Bug 修复
- `docs:` 文档更新
- `style:` 代码格式化
- `refactor:` 代码重构
- `test:` 测试相关
- `chore:` 构建过程或辅助工具的变动

### 4. 推送和创建 PR
```bash
git push origin feature/your-feature-name
```

然后在 GitHub 上创建 Pull Request。

## 📝 代码规范

### Go 代码风格
- 遵循 `gofmt` 格式化标准
- 使用 `golangci-lint` 进行代码检查
- 函数和方法需要有注释
- 导出的类型和函数必须有文档注释

### 测试要求
- 新功能必须包含单元测试
- 测试覆盖率不应降低
- 使用表驱动测试模式
- Mock 外部依赖

### 文档要求
- 更新相关的 README 文档
- 添加代码示例
- 更新 API 文档

## 🧪 测试指南

### 运行测试
```bash
# 快速测试
make test

# 单元测试 + 覆盖率
make test-unit

# 集成测试
make test-integration

# 压力测试
make test-stress

# 基准测试
make bench
```

### 编写测试
- 白盒测试：`package batchsql`
- 黑盒测试：`package batchsql_test`
- 使用 Mock 对象测试接口
- 测试边界条件和错误情况

## 🔍 代码审查

### PR 检查清单
- [ ] 代码遵循项目规范
- [ ] 包含适当的测试
- [ ] 文档已更新
- [ ] CI 检查通过
- [ ] 没有破坏性变更（或已标明）

### 审查标准
- 代码质量和可读性
- 性能影响
- 安全性考虑
- 向后兼容性
- 测试覆盖率

## 🏗️ 架构指南

### 添加新数据库驱动
1. 实现 `DatabaseDriver` 接口
2. 添加相应的测试
3. 更新文档和示例
4. 考虑性能和错误处理

### 扩展核心功能
1. 保持接口的向后兼容性
2. 考虑对现有驱动的影响
3. 添加全面的测试覆盖
4. 更新相关文档

## 📊 性能考虑

### 基准测试
- 新功能应包含基准测试
- 避免性能回归
- 考虑内存使用和 GC 压力

### 优化原则
- 批量操作优于单个操作
- 避免不必要的内存分配
- 合理使用缓存和连接池

## 🤝 社区

### 沟通渠道
- GitHub Issues：Bug 报告和功能请求
- GitHub Discussions：一般讨论和问题
- Pull Requests：代码贡献

### 行为准则
- 尊重他人观点
- 建设性的反馈
- 包容和友好的态度

## 📋 发布流程

### 版本管理
- 遵循 [Semantic Versioning](https://semver.org/)
- 主要版本：破坏性变更
- 次要版本：新功能
- 补丁版本：Bug 修复

### 发布检查
- [ ] 所有测试通过
- [ ] 文档已更新
- [ ] CHANGELOG 已更新
- [ ] 版本号已更新

## 🙏 致谢

感谢所有为 BatchSQL 项目做出贡献的开发者！

---

如有任何问题，请随时创建 Issue 或联系维护者。