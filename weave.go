package weave

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
)

// 服务容器状态
type entry[T any] struct {
	instance  any
	builder   func(T) any
	dependsOn []string // 依赖的服务名称
	built     bool     // 是否已构建
}

type Weave[T any] struct {
	ctx *T

	// 服务容器
	entries *Map[string, *entry[*T]]

	// 准备好后执行的函数
	ready []func()

	// 是否已构建
	built bool

	// 服务获取函数（用于依赖注入）
	getServiceFunc func(name string) (any, error)

	mu sync.RWMutex
}

func New[T any]() *Weave[T] {
	s := new(Weave[T])
	s.entries = NewMap[string, *entry[*T]]()

	// 初始化服务获取函数
	s.getServiceFunc = func(name string) (any, error) {
		if entry, ok := s.entries.Get(name); ok {
			return entry.instance, nil
		}
		return nil, fmt.Errorf("service [%s] not found", name)
	}

	return s
}

func (s *Weave[T]) SetCtx(ctx *T) {
	s.ctx = ctx
}

// GetServiceFunc 获取服务函数供builder使用
func (s *Weave[T]) GetService(name string) (any, error) {
	return s.getServiceFunc(name)
}

func (s *Weave[T]) Ready(fn func()) {
	s.ready = append(s.ready, fn)
}

// Auto 注册服务
func (s *Weave[T]) assign(name string, placeholder any, builder func(*T) any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := &entry[*T]{
		builder:   builder,
		instance:  placeholder,
		dependsOn: []string{},
		built:     false,
	}

	s.entries.Set(name, entry)
	s.built = false // 标记需要重新构建
}

// Build 进行全量分析和构造所有服务
func (s *Weave[T]) Build() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.built {
		return nil // 已经构建过了
	}
	var err error
	s.entries.Range(func(name string, entry *entry[*T]) bool {
		err = s.build(name, entry)
		return err == nil
	})
	if err != nil {
		return err
	}
	s.built = true
	for _, fn := range s.ready {
		fn()
	}
	return nil
}

func (s *Weave[T]) build(name string, entry *entry[*T]) error {
	if entry.built {
		return nil
	}

	originalFunc := s.getServiceFunc

	s.getServiceFunc = func(name string) (any, error) {
		e, ok := s.entries.Get(name)
		if !ok {
			return nil, fmt.Errorf("service [%s] not found", name)
		}
		entry.dependsOn = append(entry.dependsOn, name)
		if !e.built {
			if err := s.build(name, e); err != nil {
				return nil, err
			}
		}
		return e.instance, nil
	}

	entry.built = true
	instance := entry.builder(s.ctx)
	if instance == nil {
		entry.built = false
		return fmt.Errorf("service [%s] build failed", name)
	}

	// 通过反射设置实例
	vo := reflect.ValueOf(instance)
	reflect.ValueOf(entry.instance).Elem().Set(vo.Elem())

	s.getServiceFunc = originalFunc
	return nil
}

func Provide[T any, R any](di *Weave[T], name string, builder func(*T) *R) {
	di.assign(name, new(R), func(ctx *T) any {
		return builder(ctx)
	})
}

// 工具函数
func MustMake[T any, R any](di *Weave[T], name string) *R {
	obj, err := di.GetService(name)
	if err != nil {
		panic(err)
	}
	return obj.(*R)
}

func TryMake[T any, R any](di *Weave[T], name string) (*R, bool) {
	obj, err := di.GetService(name)
	if err != nil {
		return nil, false
	}
	result, ok := obj.(*R)
	return result, ok
}

// DependencyGraph 依赖图谱结构
type DependencyGraph struct {
	// Dependencies 每个服务的依赖列表
	Dependencies map[string][]string
	// Dependents 每个服务的被依赖列表
	Dependents map[string][]string
}

// GetDependencyGraph 获取完整的依赖图谱
func (s *Weave[T]) GetDependencyGraph() *DependencyGraph {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dependencies := make(map[string][]string)
	dependents := make(map[string][]string)

	// 初始化所有服务
	s.entries.Range(func(name string, entry *entry[*T]) bool {
		dependencies[name] = make([]string, len(entry.dependsOn))
		copy(dependencies[name], entry.dependsOn)

		if dependents[name] == nil {
			dependents[name] = []string{}
		}
		return true
	})

	// 构建被依赖关系
	for serviceName, deps := range dependencies {
		for _, dep := range deps {
			if dependents[dep] == nil {
				dependents[dep] = []string{}
			}
			dependents[dep] = append(dependents[dep], serviceName)
		}
	}

	// 排序以确保输出一致性
	for name := range dependencies {
		sort.Strings(dependencies[name])
		sort.Strings(dependents[name])
	}

	return &DependencyGraph{
		Dependencies: dependencies,
		Dependents:   dependents,
	}
}

// HasCircularDependency 检测是否存在循环依赖
func (s *Weave[T]) HasCircularDependency() (bool, []string) {
	graph := s.GetDependencyGraph()
	return s.detectCircularDependency(graph.Dependencies)
}

// GetAllCircularDependencies 获取所有循环依赖路径
func (s *Weave[T]) GetAllCircularDependencies() [][]string {
	graph := s.GetDependencyGraph()
	allCycles := [][]string{}
	visited := make(map[string]bool)

	for node := range graph.Dependencies {
		if !visited[node] {
			cycles := s.findAllCyclesFromNode(node, graph.Dependencies, make(map[string]bool), make(map[string]bool), []string{})
			allCycles = append(allCycles, cycles...)
			visited[node] = true
		}
	}

	return s.deduplicateCycles(allCycles)
}

// findAllCyclesFromNode 从指定节点查找所有循环
func (s *Weave[T]) findAllCyclesFromNode(node string, dependencies map[string][]string, visited, visiting map[string]bool, path []string) [][]string {
	cycles := [][]string{}

	if visiting[node] {
		// 找到循环，构建循环路径
		cycleStart := -1
		for i, n := range path {
			if n == node {
				cycleStart = i
				break
			}
		}
		if cycleStart >= 0 {
			cycle := make([]string, 0, len(path)-cycleStart+1)
			cycle = append(cycle, path[cycleStart:]...)
			cycle = append(cycle, node)
			cycles = append(cycles, cycle)
		}
		return cycles
	}

	if visited[node] {
		return cycles
	}

	visiting[node] = true
	path = append(path, node)

	for _, dep := range dependencies[node] {
		subCycles := s.findAllCyclesFromNode(dep, dependencies, visited, visiting, path)
		cycles = append(cycles, subCycles...)
	}

	visiting[node] = false
	visited[node] = true

	return cycles
}

// deduplicateCycles 去重循环路径
func (s *Weave[T]) deduplicateCycles(cycles [][]string) [][]string {
	seen := make(map[string]bool)
	result := [][]string{}

	for _, cycle := range cycles {
		if len(cycle) <= 1 {
			continue
		}

		// 规范化循环表示（从最小元素开始）
		normalized := s.normalizeCycle(cycle)
		key := strings.Join(normalized, "->")

		if !seen[key] {
			seen[key] = true
			result = append(result, normalized)
		}
	}

	return result
}

// normalizeCycle 规范化循环表示
func (s *Weave[T]) normalizeCycle(cycle []string) []string {
	if len(cycle) <= 1 {
		return cycle
	}

	// 找到最小元素的位置
	minIdx := 0
	for i, item := range cycle {
		if item < cycle[minIdx] {
			minIdx = i
		}
	}

	// 从最小元素开始重新排列
	normalized := make([]string, len(cycle))
	for i := 0; i < len(cycle); i++ {
		normalized[i] = cycle[(minIdx+i)%len(cycle)]
	}

	return normalized
}

// detectCircularDependency 使用DFS检测循环依赖
func (s *Weave[T]) detectCircularDependency(dependencies map[string][]string) (bool, []string) {
	visited := make(map[string]bool)
	visiting := make(map[string]bool)
	path := make([]string, 0)

	var dfs func(string) (bool, []string)
	dfs = func(node string) (bool, []string) {
		if visiting[node] {
			// 找到循环，构建循环路径
			cycle := []string{node}
			for i := len(path) - 1; i >= 0; i-- {
				cycle = append(cycle, path[i])
				if path[i] == node {
					break
				}
			}
			return true, cycle
		}

		if visited[node] {
			return false, nil
		}

		visiting[node] = true
		path = append(path, node)

		for _, dep := range dependencies[node] {
			if hasCycle, cycle := dfs(dep); hasCycle {
				return true, cycle
			}
		}

		visiting[node] = false
		visited[node] = true
		path = path[:len(path)-1]

		return false, nil
	}

	for node := range dependencies {
		if !visited[node] {
			if hasCycle, cycle := dfs(node); hasCycle {
				return true, cycle
			}
		}
	}

	return false, nil
}

// GenerateDOTGraph 生成DOT格式的依赖图，可用于Graphviz可视化
func (s *Weave[T]) GenerateDOTGraph() string {
	graph := s.GetDependencyGraph()

	var builder strings.Builder
	builder.WriteString("digraph DependencyGraph {\n")
	builder.WriteString("  rankdir=TB;\n")
	builder.WriteString("  node [shape=box, style=filled];\n")

	// 检测循环依赖
	hasCycle, _ := s.detectCircularDependency(graph.Dependencies)
	allCycles := [][]string{}
	if hasCycle {
		allCycles = s.GetAllCircularDependencies()
	}

	// 创建循环节点集合
	cycleNodes := make(map[string]bool)
	cycleEdges := make(map[string]bool)

	if len(allCycles) > 0 {
		for _, cycle := range allCycles {
			for i, node := range cycle {
				cycleNodes[node] = true
				if i < len(cycle)-1 {
					edge := fmt.Sprintf("%s->%s", node, cycle[i+1])
					cycleEdges[edge] = true
				}
			}
		}
	}

	// 添加所有节点
	services := make([]string, 0, len(graph.Dependencies))
	for service := range graph.Dependencies {
		services = append(services, service)
	}
	sort.Strings(services)

	builder.WriteString("\n  // 节点定义\n")
	for _, service := range services {
		if cycleNodes[service] {
			// 循环依赖中的节点用红色突出显示
			builder.WriteString(fmt.Sprintf("  \"%s\" [fillcolor=lightcoral, label=\"⚠️ %s\"];\n", service, service))
		} else {
			// 普通节点
			deps := len(graph.Dependencies[service])
			dependents := len(graph.Dependents[service])

			if deps == 0 && dependents > 0 {
				// 根节点（绿色）
				builder.WriteString(fmt.Sprintf("  \"%s\" [fillcolor=lightgreen, label=\"🌱 %s\"];\n", service, service))
			} else if deps > 0 && dependents == 0 {
				// 叶节点（黄色）
				builder.WriteString(fmt.Sprintf("  \"%s\" [fillcolor=lightyellow, label=\"🍃 %s\"];\n", service, service))
			} else {
				// 中间节点（蓝色）
				builder.WriteString(fmt.Sprintf("  \"%s\" [fillcolor=lightblue];\n", service))
			}
		}
	}

	builder.WriteString("\n  // 依赖关系边\n")

	// 添加依赖关系边
	for _, service := range services {
		for _, dep := range graph.Dependencies[service] {
			edge := fmt.Sprintf("%s->%s", dep, service)
			if cycleEdges[edge] {
				// 循环依赖边用红色粗线显示
				builder.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [color=red, penwidth=2.0, label=\"⚠️\"];\n", dep, service))
			} else {
				// 普通依赖边
				builder.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\";\n", dep, service))
			}
		}
	}

	// 如果有循环依赖，添加说明
	if len(allCycles) > 0 {
		builder.WriteString("\n  // 循环依赖说明\n")
		builder.WriteString("  legend [shape=box, style=filled, fillcolor=lightyellow, label=\"")
		builder.WriteString("图例:\\n")
		builder.WriteString("🌱 = 根服务 (无依赖)\\n")
		builder.WriteString("🍃 = 叶服务 (无被依赖)\\n")
		builder.WriteString("⚠️  = 循环依赖节点\\n")
		builder.WriteString("红色边 = 循环依赖关系")
		builder.WriteString("\"];\n")
	}

	builder.WriteString("}\n")
	return builder.String()
}

// PrintDependencyGraph 打印依赖图谱的文本表示
func (s *Weave[T]) PrintDependencyGraph() string {
	graph := s.GetDependencyGraph()

	var builder strings.Builder
	builder.WriteString("依赖图谱:\n")
	builder.WriteString("================\n\n")

	// 检测循环依赖
	hasCycle, firstCycle := s.detectCircularDependency(graph.Dependencies)
	if hasCycle {
		builder.WriteString("⚠️  检测到循环依赖!\n")
		builder.WriteString("第一个循环: ")
		builder.WriteString(strings.Join(firstCycle, " -> "))
		builder.WriteString("\n\n")

		// 获取所有循环依赖
		allCycles := s.GetAllCircularDependencies()
		if len(allCycles) > 1 {
			builder.WriteString("所有循环依赖:\n")
			for i, cycle := range allCycles {
				builder.WriteString(fmt.Sprintf("  循环 %d: %s\n", i+1, strings.Join(cycle, " -> ")))
			}
			builder.WriteString("\n")
		}
	} else {
		builder.WriteString("✅ 无循环依赖\n\n")
	}

	services := make([]string, 0, len(graph.Dependencies))
	for service := range graph.Dependencies {
		services = append(services, service)
	}
	sort.Strings(services)

	// 分类显示服务
	rootServices := []string{}
	leafServices := []string{}
	middleServices := []string{}

	for _, service := range services {
		deps := graph.Dependencies[service]
		dependents := graph.Dependents[service]

		if len(deps) == 0 && len(dependents) > 0 {
			rootServices = append(rootServices, service)
		} else if len(deps) > 0 && len(dependents) == 0 {
			leafServices = append(leafServices, service)
		} else {
			middleServices = append(middleServices, service)
		}
	}

	// 显示根服务（无依赖）
	if len(rootServices) > 0 {
		builder.WriteString("🌱 根服务 (无依赖):\n")
		for _, service := range rootServices {
			builder.WriteString(fmt.Sprintf("  📦 %s -> 被依赖于: %s\n",
				service, strings.Join(graph.Dependents[service], ", ")))
		}
		builder.WriteString("\n")
	}

	// 显示叶服务（无被依赖）
	if len(leafServices) > 0 {
		builder.WriteString("🍃 叶服务 (无被依赖):\n")
		for _, service := range leafServices {
			builder.WriteString(fmt.Sprintf("  📦 %s <- 依赖于: %s\n",
				service, strings.Join(graph.Dependencies[service], ", ")))
		}
		builder.WriteString("\n")
	}

	// 显示中间服务
	if len(middleServices) > 0 {
		builder.WriteString("🔗 中间服务:\n")
		for _, service := range middleServices {
			builder.WriteString(fmt.Sprintf("  📦 %s\n", service))

			if len(graph.Dependencies[service]) > 0 {
				builder.WriteString("    ⬅️  依赖于: ")
				builder.WriteString(strings.Join(graph.Dependencies[service], ", "))
				builder.WriteString("\n")
			}

			if len(graph.Dependents[service]) > 0 {
				builder.WriteString("    ➡️  被依赖于: ")
				builder.WriteString(strings.Join(graph.Dependents[service], ", "))
				builder.WriteString("\n")
			}
			builder.WriteString("\n")
		}
	}

	// 详细的服务信息
	builder.WriteString("详细信息:\n")
	builder.WriteString("================\n")
	for _, service := range services {
		builder.WriteString(fmt.Sprintf("服务: %s\n", service))

		if len(graph.Dependencies[service]) > 0 {
			builder.WriteString("  依赖于: ")
			builder.WriteString(strings.Join(graph.Dependencies[service], ", "))
			builder.WriteString("\n")
		} else {
			builder.WriteString("  依赖于: (无)\n")
		}

		if len(graph.Dependents[service]) > 0 {
			builder.WriteString("  被依赖于: ")
			builder.WriteString(strings.Join(graph.Dependents[service], ", "))
			builder.WriteString("\n")
		} else {
			builder.WriteString("  被依赖于: (无)\n")
		}

		builder.WriteString("\n")
	}

	return builder.String()
}

// Compact 压缩容器，释放构建时数据，节约内存
func (s *Weave[T]) Compact() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.built {
		panic("cannot compact weave before Build() is called")
	}
	s.ctx = nil
	s.ready = nil
	s.entries.Range(func(name string, entry *entry[*T]) bool {
		entry.builder = nil
		entry.dependsOn = nil
		return true
	})
}

// Extract 提取所有已构建的服务实例，返回轻量级服务注册表
// 使用此方法后，可以安全地释放DI容器实例
func (s *Weave[T]) Extract() *Map[string, any] {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.built {
		panic("cannot extract services before Build() is called")
	}

	registry := NewMap[string, any]()

	s.entries.Range(func(name string, entry *entry[*T]) bool {
		if entry.built {
			registry.Set(name, entry.instance)
		}
		return true
	})

	return registry
}

// MustGetFromRegistry 从注册表中获取服务
func MustGetFromRegistry[T any](registry *Map[string, any], name string) *T {
	obj, ok := registry.Get(name)
	if !ok {
		panic(fmt.Errorf("service [%s] not found", name))
	}
	return obj.(*T)
}

// TryGetFromRegistry 从注册表中获取服务
func TryGetFromRegistry[T any](registry *Map[string, any], name string) (*T, bool) {
	obj, ok := registry.Get(name)
	if !ok {
		return nil, false
	}
	result, ok := obj.(*T)
	return result, ok
}
