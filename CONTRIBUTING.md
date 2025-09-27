# Contributing to BatchSQL

Thank you for your interest in contributing to BatchSQL! This document provides guidelines and information for contributors.

## 🚀 Getting Started

### Prerequisites
- Go 1.20 or later
- Docker and Docker Compose (for integration tests)
- Git

### Development Setup
1. **Fork and Clone**
   ```bash
   git clone https://github.com/your-username/batchsql.git
   cd batchsql
   ```

2. **Install Dependencies**
   ```bash
   go mod download
   ```

3. **Verify Setup**
   ```bash
   make test-unit
   make lint
   ```

## 📋 Development Workflow

### 1. Create a Branch
```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-number
```

### 2. Make Changes
- Write clean, well-documented code
- Follow Go best practices and project conventions
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes
```bash
# Run unit tests
make test-unit

# Run linting
make lint

# Run integration tests (optional but recommended)
make docker-sqlite-test
make docker-mysql-test
make docker-postgres-test
```

### 4. Commit Changes
```bash
git add .
git commit -m "feat: add new feature description"
# or
git commit -m "fix: resolve issue description"
```

**Commit Message Format:**
- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `test:` - Test additions or modifications
- `refactor:` - Code refactoring
- `perf:` - Performance improvements
- `chore:` - Maintenance tasks

### 5. Push and Create PR
```bash
git push origin your-branch-name
```
Then create a Pull Request on GitHub.

## 🧪 Testing Guidelines

### Unit Tests
- Write tests for all new functions and methods
- Aim for at least 80% code coverage
- Use table-driven tests where appropriate
- Mock external dependencies

**Example:**
```go
func TestBatchSQL_Submit(t *testing.T) {
    tests := []struct {
        name    string
        request *Request
        wantErr bool
    }{
        {
            name:    "valid request",
            request: NewRequest(schema).SetString("name", "test"),
            wantErr: false,
        },
        // Add more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Integration Tests
- Test real database interactions
- Verify performance characteristics
- Test error handling and edge cases
- Use Docker containers for consistent environments

### Performance Tests
- Add benchmarks for performance-critical code
- Monitor memory allocations
- Test with realistic data volumes

**Example:**
```go
func BenchmarkBatchSQL_Submit(b *testing.B) {
    batch, _ := NewBatchSQLWithMock(ctx, config)
    request := NewRequest(schema).SetString("name", "test")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        batch.Submit(ctx, request)
    }
}
```

## 📝 Code Style Guidelines

### Go Code Style
- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names
- Write clear, concise comments
- Keep functions small and focused
- Handle errors appropriately

### Documentation
- Add GoDoc comments for public functions and types
- Update README.md for significant changes
- Include code examples in documentation
- Document configuration options and their effects

### Error Handling
```go
// Good: Specific error types
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error in field %s: %s", e.Field, e.Message)
}

// Good: Contextual error wrapping
if err := validateRequest(req); err != nil {
    return fmt.Errorf("failed to validate request: %w", err)
}
```

## 🏗️ Architecture Guidelines

### Adding New Database Drivers
1. **Create driver directory**: `drivers/newdb/`
2. **Implement interfaces**:
   ```go
   type NewDBDriver struct{}
   
   func (d *NewDBDriver) GenerateInsertSQL(schema *Schema, batchSize int) string {
       // Implementation
   }
   
   type NewDBExecutor struct{}
   
   func (e *NewDBExecutor) ExecuteBatch(ctx context.Context, requests []*Request) error {
       // Implementation
   }
   ```
3. **Add factory method**:
   ```go
   func NewNewDBBatchSQL(ctx context.Context, db *sql.DB, config PipelineConfig) *BatchSQL {
       // Implementation
   }
   ```
4. **Add tests and documentation**

### Performance Considerations
- Use pointer receivers for methods
- Minimize memory allocations in hot paths
- Consider using sync.Pool for frequently allocated objects
- Profile code to identify bottlenecks

## 🐛 Bug Reports and Feature Requests

### Reporting Bugs
1. Check existing issues first
2. Use the bug report template
3. Provide minimal reproduction case
4. Include environment details
5. Add relevant logs and error messages

### Requesting Features
1. Use the feature request template
2. Explain the use case and problem
3. Propose a solution
4. Consider backwards compatibility
5. Discuss API design implications

## 🔄 Review Process

### Pull Request Requirements
- [ ] All tests pass
- [ ] Code coverage maintained or improved
- [ ] Documentation updated
- [ ] No linting errors
- [ ] Backwards compatibility preserved (unless breaking change is justified)

### Review Criteria
- **Functionality**: Does the code work as intended?
- **Performance**: Are there any performance regressions?
- **Security**: Are there any security implications?
- **Maintainability**: Is the code easy to understand and maintain?
- **Testing**: Are there adequate tests?
- **Documentation**: Is the documentation clear and complete?

## 📊 CI/CD Pipeline

### Automated Checks
- Code formatting (`go fmt`)
- Linting (`golangci-lint`)
- Unit tests with coverage
- Integration tests (MySQL, PostgreSQL, SQLite)
- Performance benchmarks

### Manual Testing
- Test with different Go versions
- Verify on different operating systems
- Test with various database versions
- Performance testing under load

## 🎯 Project Priorities

### Current Focus Areas
1. **Performance Optimization**: Improving throughput and reducing latency
2. **Error Handling**: Better error messages and recovery mechanisms
3. **Documentation**: Comprehensive guides and examples
4. **Testing**: Increasing test coverage and reliability

### Future Roadmap
- Additional database support (TiDB, ClickHouse)
- Monitoring and metrics integration
- Connection pool optimization
- Advanced batching strategies

## 🤝 Community Guidelines

### Code of Conduct
- Be respectful and inclusive
- Provide constructive feedback
- Help newcomers get started
- Focus on technical merit
- Maintain professional communication

### Getting Help
- Check existing documentation first
- Search closed issues for similar problems
- Ask questions in GitHub Discussions
- Provide context and examples when asking for help

## 📚 Resources

### Documentation
- [README.md](README.md) - Project overview and basic usage
- [CONFIG.md](CONFIG.md) - Configuration options
- [README-INTEGRATION-TESTS.md](README-INTEGRATION-TESTS.md) - Integration testing guide

### Development Tools
- [golangci-lint](https://golangci-lint.run/) - Go linting
- [Docker](https://www.docker.com/) - Containerization
- [Make](https://www.gnu.org/software/make/) - Build automation

### Learning Resources
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Testing](https://golang.org/pkg/testing/)

## 📞 Contact

- **Issues**: [GitHub Issues](https://github.com/rushairer/batchsql/issues)
- **Discussions**: [GitHub Discussions](https://github.com/rushairer/batchsql/discussions)
- **Security**: Report security issues privately via email

---

Thank you for contributing to BatchSQL! Your efforts help make this project better for everyone. 🙏