# Weave - Go 依赖注入容器 / Go Dependency Injection Container

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.18-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

[English](#english) | [中文](#中文)

---

## 中文

### 📖 项目简介

Weave 是一个轻量级、高性能的 Go 语言依赖注入容器，支持泛型、线程安全，并具备依赖关系分析和可视化功能。名字来源于"编织"的含义，形象地描述了服务之间复杂的依赖关系网络。

### ✨ 主要特性

- 🔧 **泛型支持** - 完全支持 Go 1.18+ 泛型，类型安全
- 🔒 **线程安全** - 内置并发安全机制
- 📊 **依赖分析** - 构建后分析服务依赖关系
- 🔍 **循环检测** - 智能检测和报告循环依赖
- 📈 **可视化** - 支持文本和 DOT 格式的依赖图谱
- 🚀 **Ready 回调** - 支持构建完成后的回调机制
- 💾 **服务提取** - 支持构建后提取服务实例
- ⚡ **高性能** - 优化的构建和查找算法
- 🎯 **简单易用** - 直观的 API 设计

### 📦 安装

```bash
go get github.com/youjianglong/weave
```

### 🚀 快速开始

```go
package main

import (
    "fmt"
    "github.com/youjianglong/weave"
)

// 定义上下文
type AppContext struct {
    Config string
}

// 定义服务
type Database struct {
    Name string
}

type UserService struct {
    DB *Database
}

func main() {
    // 创建 Weave 容器
    w := weave.New[AppContext]()
    ctx := &AppContext{Config: "production"}
    w.SetCtx(ctx)

    // 注册服务
    weave.Provide(w, "database", func(ctx *AppContext) *Database {
        return &Database{Name: "PostgreSQL"}
    })

    weave.Provide(w, "userService", func(ctx *AppContext) *UserService {
        db := weave.MustMake[AppContext, Database](w, "database")
        return &UserService{DB: db}
    })

    // 添加构建完成回调
    w.Ready(func() {
        fmt.Println("所有服务构建完成！")
    })

    // 构建服务
    if err := w.Build(); err != nil {
        panic(err)
    }

    // 使用服务
    userService := weave.MustMake[AppContext, UserService](w, "userService")
    fmt.Printf("用户服务已启动，使用数据库: %s\n", userService.DB.Name)
}
```

### 📚 API 文档

#### 核心 API

```go
// 创建新的 Weave 容器
func New[T any]() *Weave[T]

// 设置上下文
func (w *Weave[T]) SetCtx(ctx *T)

// 注册服务
func Provide[T any, R any](w *Weave[T], name string, builder func(*T) *R)

// 构建所有服务
func (w *Weave[T]) Build() error

// 添加构建完成回调
func (w *Weave[T]) Ready(fn func())

// 获取服务（必须存在）
func MustMake[T any, R any](w *Weave[T], name string) *R

// 安全获取服务
func TryMake[T any, R any](w *Weave[T], name string) (*R, bool)

// 获取服务（返回错误）
func (w *Weave[T]) GetService(name string) (any, error)
```

#### 服务提取 API

```go
// 提取所有已构建的服务实例
func (w *Weave[T]) Extract() *Map[string, any]

// 压缩容器，释放构建时数据
func (w *Weave[T]) Compact()

// 从服务映射获取服务（必须存在）
func MustGetFromRegistry[T any](registry *Map[string, any], name string) *T

// 从服务映射安全获取服务
func TryGetFromRegistry[T any](registry *Map[string, any], name string) (*T, bool)
```

#### 依赖分析 API

```go
// 获取依赖图谱（需要先Build）
func (w *Weave[T]) GetDependencyGraph() *DependencyGraph

// 检测循环依赖
func (w *Weave[T]) HasCircularDependency() (bool, []string)

// 获取所有循环依赖
func (w *Weave[T]) GetAllCircularDependencies() [][]string

// 打印依赖图谱
func (w *Weave[T]) PrintDependencyGraph() string

// 生成 DOT 格式图谱
func (w *Weave[T]) GenerateDOTGraph() string
```

### 🔧 高级功能

#### Ready 回调机制

Ready 回调允许您在所有服务构建完成后执行特定的逻辑：

```go
w := weave.New[AppContext]()

// 添加多个回调
w.Ready(func() {
    fmt.Println("初始化日志系统...")
})

w.Ready(func() {
    fmt.Println("启动HTTP服务器...")
})

w.Ready(func() {
    fmt.Println("应用程序启动完成！")
})

// 构建服务时会按顺序执行所有回调
w.Build()
```

#### 服务提取

Weave 支持在构建完成后提取服务实例，实现与容器的解耦：

```go
func extractServices() {
    w := weave.New[AppContext]()
    // ... 注册服务 ...
    
    // 构建所有服务
    w.Build()
    
    // 提取服务映射
    services := w.Extract()
    
    // 现在可以独立使用服务映射
    userService := weave.MustGetFromRegistry[UserService](services, "userService")
    database := weave.MustGetFromRegistry[Database](services, "database")
    
    // 或者安全获取
    if logger, ok := weave.TryGetFromRegistry[Logger](services, "logger"); ok {
        logger.Info("服务提取完成")
    }
}
```

**使用场景**：
- 🎯 **服务解耦**: 将服务实例与容器分离
- 🔄 **生命周期管理**: 在不同阶段使用不同的服务访问方式
- 📦 **模块化**: 将服务实例传递给其他模块
- 🧪 **测试**: 在测试中更灵活地控制服务实例

#### 内存优化

Weave 提供 `Compact()` 方法来释放构建期数据，减少内存占用：

```go
w := weave.New[AppContext]()
// ... 注册服务 ...

// 构建所有服务
w.Build()

// 压缩容器，释放构建期数据（构建函数、回调函数等）
w.Compact()

// 服务仍然可用，但内存占用显著减少
userService := weave.MustMake[AppContext, UserService](w, "userService")

// 注意：调用 Compact() 后无法再获取依赖图谱
// graph := w.GetDependencyGraph() // 这会导致不准确的结果
```

**使用场景**：
- 🔋 **生产环境**: 服务构建完成后释放不必要的内存
- 📱 **资源受限环境**: 嵌入式系统或移动应用
- ☁️ **云函数**: 优化冷启动后的内存使用
- 🚀 **长期运行服务**: 减少应用程序的内存占用

#### 使用注意事项

在使用 Weave 时，请注意以下几点：

**1. 服务方法调用限制**

在 `Provide` 的 builder 函数中，不能直接调用已注入服务的方法。如果需要调用服务方法，应该在 `Ready` 回调中进行：

```go
// ❌ 错误用法：在 builder 中直接调用服务方法
weave.Provide(w, "userService", func(ctx *AppContext) *UserService {
    db := weave.MustMake[AppContext, Database](w, "database")
    db.Connect() // ❌ 不能在这里调用方法
    return &UserService{DB: db}
})

// ✅ 正确用法：在 Ready 回调中调用服务方法
weave.Provide(w, "userService", func(ctx *AppContext) *UserService {
    db := weave.MustMake[AppContext, Database](w, "database")
    return &UserService{DB: db} // ✅ 只注入依赖，不调用方法
})

w.Ready(func() {
    // ✅ 在 Ready 回调中调用服务方法
    db := weave.MustMake[AppContext, Database](w, "database")
    db.Connect()
})
```

**2. 依赖图谱分析限制**

必须在调用 `Build()` 之后才能获取准确的依赖图谱：

```go
w := weave.New[AppContext]()
// ... 注册服务 ...

// ❌ 错误：构建前获取依赖图谱
// graph := w.GetDependencyGraph() // 结果不准确

// ✅ 正确：先构建再获取依赖图谱
w.Build()
graph := w.GetDependencyGraph() // ✅ 准确的依赖关系

// ⚠️  注意：调用 Compact() 后依赖图谱信息会被清理
w.Compact()
// graph = w.GetDependencyGraph() // 依赖信息不完整
```

**3. Extract 和 Compact 的使用时机**

- `Extract()`: 可以在 `Build()` 后的任何时候调用
- `Compact()`: 只能在 `Build()` 后调用，且会影响依赖图谱功能

```go
w.Build()           // ✅ 必须先构建
services := w.Extract()  // ✅ 可以提取服务
w.Compact()         // ✅ 可以压缩容器
// w.GetDependencyGraph() // ⚠️  压缩后依赖信息不完整
```

#### 依赖关系分析

在服务构建完成后，可以分析依赖关系：

```go
// 构建服务
if err := w.Build(); err != nil {
    panic(err)
}

// 获取依赖图谱
graph := w.GetDependencyGraph()

// 检查循环依赖
if hasCycle, cycle := w.HasCircularDependency(); hasCycle {
    fmt.Printf("检测到循环依赖: %v\n", cycle)
}

// 打印可读的依赖图谱
fmt.Println(w.PrintDependencyGraph())
```

#### 可视化依赖图谱

```go
// 生成 DOT 格式（用于 Graphviz）
dotGraph := w.GenerateDOTGraph()

// 保存到文件
ioutil.WriteFile("dependencies.dot", []byte(dotGraph), 0644)

// 使用 Graphviz 生成图片
// dot -Tpng dependencies.dot -o dependencies.png
```

### 📋 完整示例

```go
package main

import (
    "fmt"
    "log"
    "github.com/youjianglong/weave"
)

type AppContext struct {
    Environment string
}

type Logger struct {
    Level string
}

type Database struct {
    DSN    string
    Logger *Logger
}

type UserRepository struct {
    DB *Database
}

type UserService struct {
    Repo   *UserRepository
    Logger *Logger
}

func main() {
    w := weave.New[AppContext]()
    ctx := &AppContext{Environment: "production"}
    w.SetCtx(ctx)

    // 注册服务
    weave.Provide(w, "logger", func(ctx *AppContext) *Logger {
        level := "info"
        if ctx.Environment == "development" {
            level = "debug"
        }
        return &Logger{Level: level}
    })

    weave.Provide(w, "database", func(ctx *AppContext) *Database {
        logger := weave.MustMake[AppContext, Logger](w, "logger")
        return &Database{
            DSN:    "postgres://localhost/myapp",
            Logger: logger,
        }
    })

    weave.Provide(w, "userRepo", func(ctx *AppContext) *UserRepository {
        db := weave.MustMake[AppContext, Database](w, "database")
        return &UserRepository{DB: db}
    })

    weave.Provide(w, "userService", func(ctx *AppContext) *UserService {
        repo := weave.MustMake[AppContext, UserRepository](w, "userRepo")
        logger := weave.MustMake[AppContext, Logger](w, "logger")
        return &UserService{
            Repo:   repo,
            Logger: logger,
        }
    })

    // 添加启动完成回调
    w.Ready(func() {
        log.Println("数据库连接已建立")
    })

    w.Ready(func() {
        log.Println("用户服务已启动")
    })

    w.Ready(func() {
        log.Println("应用程序启动完成！")
    })

    // 构建所有服务
    if err := w.Build(); err != nil {
        panic(fmt.Sprintf("构建失败: %v", err))
    }

    // 打印依赖图谱
    fmt.Println(w.PrintDependencyGraph())

    // 使用服务
    userService := weave.MustMake[AppContext, UserService](w, "userService")
    fmt.Printf("用户服务已启动 (日志级别: %s)\n", userService.Logger.Level)

    // 提取服务供其他模块使用
    services := w.Extract()
    logger := weave.MustGetFromRegistry[Logger](services, "logger")
    fmt.Printf("提取的日志级别: %s\n", logger.Level)
}
```

### 🤝 贡献

欢迎提交 Issue 和 Pull Request！

### 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

---

## English

### 📖 Introduction

Weave is a lightweight, high-performance dependency injection container for Go, featuring generics support, thread safety, and dependency analysis and visualization capabilities. The name "weave" metaphorically describes the intricate network of service dependencies.

### ✨ Features

- 🔧 **Generics Support** - Full support for Go 1.18+ generics with type safety
- 🔒 **Thread Safe** - Built-in concurrency safety mechanisms  
- 📊 **Dependency Analysis** - Post-build service dependency analysis
- 🔍 **Circular Detection** - Smart detection and reporting of circular dependencies
- 📈 **Visualization** - Support for text and DOT format dependency graphs
- 🚀 **Ready Callbacks** - Support for post-build callback mechanisms
- 💾 **Service Extraction** - Support for extracting service instances after build
- ⚡ **High Performance** - Optimized build and lookup algorithms
- 🎯 **Easy to Use** - Intuitive API design

### 📦 Installation

```bash
go get github.com/youjianglong/weave
```

### 🚀 Quick Start

```go
package main

import (
    "fmt"
    "github.com/youjianglong/weave"
)

// Define context
type AppContext struct {
    Config string
}

// Define services
type Database struct {
    Name string
}

type UserService struct {
    DB *Database
}

func main() {
    // Create Weave container
    w := weave.New[AppContext]()
    ctx := &AppContext{Config: "production"}
    w.SetCtx(ctx)

    // Register services
    weave.Provide(w, "database", func(ctx *AppContext) *Database {
        return &Database{Name: "PostgreSQL"}
    })

    weave.Provide(w, "userService", func(ctx *AppContext) *UserService {
        db := weave.MustMake[AppContext, Database](w, "database")
        return &UserService{DB: db}
    })

    // Add ready callback
    w.Ready(func() {
        fmt.Println("All services built successfully!")
    })

    // Build services
    if err := w.Build(); err != nil {
        panic(err)
    }

    // Use services
    userService := weave.MustMake[AppContext, UserService](w, "userService")
    fmt.Printf("User service started with database: %s\n", userService.DB.Name)
}
```

### 📚 API Documentation

#### Core API

```go
// Create new Weave container
func New[T any]() *Weave[T]

// Set context
func (w *Weave[T]) SetCtx(ctx *T)

// Register service
func Provide[T any, R any](w *Weave[T], name string, builder func(*T) *R)

// Build all services
func (w *Weave[T]) Build() error

// Add ready callback
func (w *Weave[T]) Ready(fn func())

// Get service (must exist)
func MustMake[T any, R any](w *Weave[T], name string) *R

// Safe get service
func TryMake[T any, R any](w *Weave[T], name string) (*R, bool)

// Get service with error
func (w *Weave[T]) GetService(name string) (any, error)
```

#### Service Extraction API

```go
// Extract all built service instances
func (w *Weave[T]) Extract() *Map[string, any]

// Compact container, release data built during
func (w *Weave[T]) Compact()

// Get service from service map (must exist)
func MustGetFromRegistry[T any](registry *Map[string, any], name string) *T

// Safe get service from service map
func TryGetFromRegistry[T any](registry *Map[string, any], name string) (*T, bool)
```

#### Dependency Analysis API

```go
// Get dependency graph (requires Build first)
func (w *Weave[T]) GetDependencyGraph() *DependencyGraph

// Detect circular dependencies
func (w *Weave[T]) HasCircularDependency() (bool, []string)

// Get all circular dependencies
func (w *Weave[T]) GetAllCircularDependencies() [][]string

// Print dependency graph
func (w *Weave[T]) PrintDependencyGraph() string

// Generate DOT format graph
func (w *Weave[T]) GenerateDOTGraph() string
```

### 🔧 Advanced Features

#### Ready Callback Mechanism

Ready callbacks allow you to execute specific logic after all services are built:

```go
w := weave.New[AppContext]()

// Add multiple callbacks
w.Ready(func() {
    fmt.Println("Initializing logging system...")
})

w.Ready(func() {
    fmt.Println("Starting HTTP server...")
})

w.Ready(func() {
    fmt.Println("Application startup complete!")
})

// All callbacks will be executed in order during Build()
w.Build()
```

#### Service Extraction

Weave supports extracting service instances after build completion, enabling decoupling from the container:

```go
func extractServices() {
    w := weave.New[AppContext]()
    // ... register services ...
    
    // Build all services
    w.Build()
    
    // Extract service map
    services := w.Extract()
    
    // Now you can use the service map independently
    userService := weave.MustGetFromRegistry[UserService](services, "userService")
    database := weave.MustGetFromRegistry[Database](services, "database")
    
    // Or safely get services
    if logger, ok := weave.TryGetFromRegistry[Logger](services, "logger"); ok {
        logger.Info("Service extraction completed")
    }
}
```

**Use Cases**:
- 🎯 **Service Decoupling**: Separate service instances from container
- 🔄 **Lifecycle Management**: Use different service access patterns in different phases
- 📦 **Modularization**: Pass service instances to other modules
- 🧪 **Testing**: More flexible control over service instances in tests

#### Memory Optimization

Weave provides the `Compact()` method to release build-time data and reduce memory usage:

```go
w := weave.New[AppContext]()
// ... register services ...

// Build all services
w.Build()

// Compact container, release data built (builder functions, callbacks, etc.)
w.Compact()

// Services are still available, but memory usage is significantly reduced
userService := weave.MustMake[AppContext, UserService](w, "userService")

// Note: After calling Compact(), you cannot get the dependency graph again
// graph := w.GetDependencyGraph() // This will result in inaccurate results
```

**Use Cases**:
- 🔋 **Production Environment**: Release unnecessary memory after service build
- 📱 **Resource-constrained Environments**: Embedded systems or mobile applications
- ☁️ **Cloud Functions**: Optimize memory usage after cold start
- 🚀 **Long-running Services**: Reduce application memory usage

#### Usage Notes

When using Weave, please note the following:

**1. Service Method Call Limitation**

In the `Provide` builder function, you cannot directly call methods of the injected service. If you need to call service methods, you should do it in a `Ready` callback:

```go
// ❌ Incorrect usage: Calling service methods directly in the builder
weave.Provide(w, "userService", func(ctx *AppContext) *UserService {
    db := weave.MustMake[AppContext, Database](w, "database")
    db.Connect() // ❌ Cannot call methods here
    return &UserService{DB: db}
})

// ✅ Correct usage: Calling service methods in a Ready callback
weave.Provide(w, "userService", func(ctx *AppContext) *UserService {
    db := weave.MustMake[AppContext, Database](w, "database")
    return &UserService{DB: db} // ✅ Only inject dependencies, do not call methods
})

w.Ready(func() {
    // ✅ Calling service methods in a Ready callback
    db := weave.MustMake[AppContext, Database](w, "database")
    db.Connect()
})
```

**2. Dependency Graph Analysis Limitation**

You can only get an accurate dependency graph after calling `Build()`:

```go
w := weave.New[AppContext]()
// ... register services ...

// ❌ Incorrect: Getting dependency graph before build
// graph := w.GetDependencyGraph() // Results are inaccurate

// ✅ Correct: Build first, then get dependency graph
w.Build()
graph := w.GetDependencyGraph() // ✅ Accurate dependency relationships

// ⚠️   Note: After calling Compact(), dependency graph information will be cleared
w.Compact()
// graph = w.GetDependencyGraph() // Incomplete dependency information
```

**3. When to Use Extract and Compact**

- `Extract()`: Can be called at any time after `Build()`
- `Compact()`: Can only be called after `Build()`, and it will affect dependency graph functionality

```go
w.Build()           // ✅ Must build first
services := w.Extract()  // ✅ Can extract services
w.Compact()         // ✅ Can compact container
// w.GetDependencyGraph() // ⚠️   Incomplete dependency information after compression
```

#### Dependency Analysis

After services are built, you can analyze dependencies:

```go
// Build services
if err := w.Build(); err != nil {
    panic(err)
}

// Get dependency graph
graph := w.GetDependencyGraph()

// Check circular dependencies
if hasCycle, cycle := w.HasCircularDependency(); hasCycle {
    fmt.Printf("Circular dependency detected: %v\n", cycle)
}

// Print readable dependency graph
fmt.Println(w.PrintDependencyGraph())
```

#### Visualize Dependency Graph

```go
// Generate DOT format (for Graphviz)
dotGraph := w.GenerateDOTGraph()

// Save to file
ioutil.WriteFile("dependencies.dot", []byte(dotGraph), 0644)

// Generate image using Graphviz
// dot -Tpng dependencies.dot -o dependencies.png
```

### 🤝 Contributing

Issues and Pull Requests are welcome!

### 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 