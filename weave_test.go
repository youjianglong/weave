package weave

import (
	"fmt"
	"strings"
	"testing"
)

// 测试用的上下文结构
type TestContext struct {
	Config string
}

// 测试用的服务结构
type ServiceA struct {
	Name string
}

type ServiceB struct {
	Name     string
	ServiceA *ServiceA
}

type ServiceC struct {
	Name     string
	ServiceA *ServiceA
	ServiceB *ServiceB
}

type ServiceD struct {
	Name     string
	ServiceC *ServiceC
}

func TestDI_Basic(t *testing.T) {
	di := New[TestContext]()
	ctx := &TestContext{Config: "test"}
	di.SetCtx(ctx)

	// 注册服务A
	Provide(di, "serviceA", func(ctx *TestContext) *ServiceA {
		return &ServiceA{Name: "ServiceA"}
	})

	// 构建服务
	err := di.Build()
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}

	// 获取服务
	serviceA := MustMake[TestContext, ServiceA](di, "serviceA")
	if serviceA.Name != "ServiceA" {
		t.Errorf("期望服务名称为 'ServiceA'，实际为 '%s'", serviceA.Name)
	}
}

func TestDI_WithDependencies(t *testing.T) {
	di := New[TestContext]()
	ctx := &TestContext{Config: "test"}
	di.SetCtx(ctx)

	// 注册服务A
	Provide(di, "serviceA", func(ctx *TestContext) *ServiceA {
		return &ServiceA{Name: "ServiceA"}
	})

	// 注册服务B，依赖服务A
	Provide(di, "serviceB", func(ctx *TestContext) *ServiceB {
		serviceA := MustMake[TestContext, ServiceA](di, "serviceA")
		return &ServiceB{
			Name:     "ServiceB",
			ServiceA: serviceA,
		}
	})

	// 注册服务C，依赖服务A和B
	Provide(di, "serviceC", func(ctx *TestContext) *ServiceC {
		serviceA := MustMake[TestContext, ServiceA](di, "serviceA")
		serviceB := MustMake[TestContext, ServiceB](di, "serviceB")
		return &ServiceC{
			Name:     "ServiceC",
			ServiceA: serviceA,
			ServiceB: serviceB,
		}
	})

	// 构建所有服务
	err := di.Build()
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}

	// 验证服务C
	serviceC := MustMake[TestContext, ServiceC](di, "serviceC")
	if serviceC.Name != "ServiceC" {
		t.Errorf("期望服务名称为 'ServiceC'，实际为 '%s'", serviceC.Name)
	}
	if serviceC.ServiceA == nil {
		t.Error("ServiceC的ServiceA依赖为空")
	}
	if serviceC.ServiceB == nil {
		t.Error("ServiceC的ServiceB依赖为空")
	}
	if serviceC.ServiceA.Name != "ServiceA" {
		t.Errorf("期望依赖的ServiceA名称为 'ServiceA'，实际为 '%s'", serviceC.ServiceA.Name)
	}
}

func TestDI_GetDependencyGraph(t *testing.T) {
	di := New[TestContext]()
	ctx := &TestContext{Config: "test"}
	di.SetCtx(ctx)

	// 注册服务
	Provide(di, "serviceA", func(ctx *TestContext) *ServiceA {
		return &ServiceA{Name: "ServiceA"}
	})

	Provide(di, "serviceB", func(ctx *TestContext) *ServiceB {
		serviceA := MustMake[TestContext, ServiceA](di, "serviceA")
		return &ServiceB{
			Name:     "ServiceB",
			ServiceA: serviceA,
		}
	})

	Provide(di, "serviceC", func(ctx *TestContext) *ServiceC {
		serviceA := MustMake[TestContext, ServiceA](di, "serviceA")
		serviceB := MustMake[TestContext, ServiceB](di, "serviceB")
		return &ServiceC{
			Name:     "ServiceC",
			ServiceA: serviceA,
			ServiceB: serviceB,
		}
	})

	// 构建服务以生成依赖关系
	err := di.Build()
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}

	// 获取依赖图谱
	graph := di.GetDependencyGraph()

	// 验证依赖关系
	if len(graph.Dependencies["serviceA"]) != 0 {
		t.Errorf("ServiceA不应该有依赖，实际依赖: %v", graph.Dependencies["serviceA"])
	}

	if len(graph.Dependencies["serviceB"]) != 1 || graph.Dependencies["serviceB"][0] != "serviceA" {
		t.Errorf("ServiceB应该依赖ServiceA，实际依赖: %v", graph.Dependencies["serviceB"])
	}

	if len(graph.Dependencies["serviceC"]) != 2 {
		t.Errorf("ServiceC应该有2个依赖，实际依赖: %v", graph.Dependencies["serviceC"])
	}

	// 验证被依赖关系
	if len(graph.Dependents["serviceA"]) != 2 {
		t.Errorf("ServiceA应该被2个服务依赖，实际被依赖: %v", graph.Dependents["serviceA"])
	}
}

func TestDI_CircularDependencyDetection(t *testing.T) {
	di := New[TestContext]()
	ctx := &TestContext{Config: "test"}
	di.SetCtx(ctx)

	// 创建循环依赖：A -> B -> A
	Provide(di, "serviceA", func(ctx *TestContext) *ServiceA {
		// 这里会造成循环依赖
		MustMake[TestContext, ServiceB](di, "serviceB")
		return &ServiceA{Name: "ServiceA"}
	})

	Provide(di, "serviceB", func(ctx *TestContext) *ServiceB {
		serviceA := MustMake[TestContext, ServiceA](di, "serviceA")
		return &ServiceB{
			Name:     "ServiceB",
			ServiceA: serviceA,
		}
	})

	// 尝试构建 - 应该成功（DI系统支持循环依赖）
	err := di.Build()
	if err != nil {
		t.Error("依赖构建失败:", err)
	}

	// 测试循环依赖检测
	hasCycle, cycle := di.HasCircularDependency()
	if !hasCycle {
		t.Error("应该检测到循环依赖")
	}
	if len(cycle) == 0 {
		t.Error("循环路径不应该为空")
	}
	t.Logf("检测到循环依赖: %v", cycle)
}

func TestDI_ComplexCircularDependencies(t *testing.T) {
	di := New[TestContext]()
	ctx := &TestContext{Config: "test"}
	di.SetCtx(ctx)

	// 创建复杂的循环依赖网络
	// A -> B -> C -> A (大循环)
	// B -> D -> B (小循环)
	Provide(di, "serviceA", func(ctx *TestContext) *ServiceA {
		MustMake[TestContext, ServiceB](di, "serviceB")
		return &ServiceA{Name: "ServiceA"}
	})

	Provide(di, "serviceB", func(ctx *TestContext) *ServiceB {
		MustMake[TestContext, ServiceC](di, "serviceC")
		MustMake[TestContext, ServiceD](di, "serviceD")
		return &ServiceB{Name: "ServiceB"}
	})

	Provide(di, "serviceC", func(ctx *TestContext) *ServiceC {
		MustMake[TestContext, ServiceA](di, "serviceA")
		return &ServiceC{Name: "ServiceC"}
	})

	Provide(di, "serviceD", func(ctx *TestContext) *ServiceD {
		MustMake[TestContext, ServiceB](di, "serviceB")
		return &ServiceD{Name: "ServiceD"}
	})

	err := di.Build()
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}

	// 获取所有循环依赖
	allCycles := di.GetAllCircularDependencies()

	if len(allCycles) == 0 {
		t.Error("应该检测到循环依赖")
	}

	t.Logf("检测到 %d 个循环依赖:", len(allCycles))
	for i, cycle := range allCycles {
		t.Logf("  循环 %d: %v", i+1, cycle)
	}

	// 验证至少检测到两个循环
	if len(allCycles) < 2 {
		t.Errorf("应该检测到至少2个循环，实际检测到 %d 个", len(allCycles))
	}
}

func TestDI_DependencyGraphCategories(t *testing.T) {
	di := New[TestContext]()
	ctx := &TestContext{Config: "test"}
	di.SetCtx(ctx)

	// 创建不同类型的服务：根、叶、中间
	Provide(di, "rootService", func(ctx *TestContext) *ServiceA {
		return &ServiceA{Name: "Root"}
	})

	Provide(di, "middleService", func(ctx *TestContext) *ServiceB {
		root := MustMake[TestContext, ServiceA](di, "rootService")
		return &ServiceB{Name: "Middle", ServiceA: root}
	})

	Provide(di, "leafService", func(ctx *TestContext) *ServiceC {
		middle := MustMake[TestContext, ServiceB](di, "middleService")
		root := MustMake[TestContext, ServiceA](di, "rootService")
		return &ServiceC{Name: "Leaf", ServiceA: root, ServiceB: middle}
	})

	err := di.Build()
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}

	// 获取依赖图谱输出
	output := di.PrintDependencyGraph()
	t.Logf("分类依赖图谱:\n%s", output)

	// 验证输出包含分类信息
	if !strings.Contains(output, "根服务") {
		t.Error("输出应该包含根服务分类")
	}
	if !strings.Contains(output, "叶服务") {
		t.Error("输出应该包含叶服务分类")
	}
	if !strings.Contains(output, "中间服务") {
		t.Error("输出应该包含中间服务分类")
	}
}

func TestDI_CircularDependencyInPrintOutput(t *testing.T) {
	di := New[TestContext]()
	ctx := &TestContext{Config: "test"}
	di.SetCtx(ctx)

	// 创建循环依赖
	Provide(di, "serviceA", func(ctx *TestContext) *ServiceA {
		MustMake[TestContext, ServiceB](di, "serviceB")
		return &ServiceA{Name: "ServiceA"}
	})

	Provide(di, "serviceB", func(ctx *TestContext) *ServiceB {
		MustMake[TestContext, ServiceA](di, "serviceA")
		return &ServiceB{Name: "ServiceB"}
	})

	err := di.Build()
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}

	// 获取依赖图谱输出
	output := di.PrintDependencyGraph()
	t.Logf("循环依赖图谱输出:\n%s", output)

	// 验证输出包含循环依赖警告
	if !strings.Contains(output, "检测到循环依赖") {
		t.Error("输出应该包含循环依赖警告")
	}
	if !strings.Contains(output, "循环") {
		t.Error("输出应该显示循环路径")
	}
}

func TestDI_NormalizeCycle(t *testing.T) {
	di := New[TestContext]()

	// 测试循环规范化
	testCases := []struct {
		input    []string
		expected []string
	}{
		{
			input:    []string{"B", "C", "A"},
			expected: []string{"A", "B", "C"},
		},
		{
			input:    []string{"C", "A", "B"},
			expected: []string{"A", "B", "C"},
		},
		{
			input:    []string{"A", "B", "C"},
			expected: []string{"A", "B", "C"},
		},
	}

	for _, tc := range testCases {
		result := di.normalizeCycle(tc.input)
		if !equalSlices(result, tc.expected) {
			t.Errorf("normalizeCycle(%v) = %v, 期望 %v", tc.input, result, tc.expected)
		}
	}
}

// 辅助函数：比较两个字符串切片是否相等
func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDI_GenerateDOTGraph(t *testing.T) {
	di := New[TestContext]()
	ctx := &TestContext{Config: "test"}
	di.SetCtx(ctx)

	// 注册简单的依赖关系
	Provide(di, "serviceA", func(ctx *TestContext) *ServiceA {
		return &ServiceA{Name: "ServiceA"}
	})

	Provide(di, "serviceB", func(ctx *TestContext) *ServiceB {
		serviceA := MustMake[TestContext, ServiceA](di, "serviceA")
		return &ServiceB{
			Name:     "ServiceB",
			ServiceA: serviceA,
		}
	})

	// 构建服务
	err := di.Build()
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}

	// 生成DOT图
	dot := di.GenerateDOTGraph()
	t.Logf("DOT图:\n%s", dot)

	// 验证DOT图包含必要的元素
	if !strings.Contains(dot, "digraph DependencyGraph") {
		t.Error("DOT图应该包含图名称")
	}
	if !strings.Contains(dot, "serviceA") {
		t.Error("DOT图应该包含serviceA")
	}
	if !strings.Contains(dot, "serviceB") {
		t.Error("DOT图应该包含serviceB")
	}
	if !strings.Contains(dot, "serviceA\" -> \"serviceB") {
		t.Error("DOT图应该包含从serviceA到serviceB的依赖关系")
	}
}

func TestDI_GenerateDOTGraphWithCircularDependencies(t *testing.T) {
	di := New[TestContext]()
	ctx := &TestContext{Config: "test"}
	di.SetCtx(ctx)

	// 创建循环依赖：A -> B -> A
	Provide(di, "serviceA", func(ctx *TestContext) *ServiceA {
		MustMake[TestContext, ServiceB](di, "serviceB")
		return &ServiceA{Name: "ServiceA"}
	})

	Provide(di, "serviceB", func(ctx *TestContext) *ServiceB {
		MustMake[TestContext, ServiceA](di, "serviceA")
		return &ServiceB{Name: "ServiceB"}
	})

	err := di.Build()
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}

	// 生成DOT图（无需构建，依赖分析会自动进行）
	dot := di.GenerateDOTGraph()
	t.Logf("循环依赖DOT图:\n%s", dot)

	// 验证循环依赖的特殊标记
	if !strings.Contains(dot, "lightcoral") {
		t.Error("循环依赖节点应该用红色(lightcoral)标记")
	}
	if !strings.Contains(dot, "⚠️") {
		t.Error("DOT图应该包含循环依赖警告符号")
	}
	if !strings.Contains(dot, "color=red") {
		t.Error("循环依赖边应该用红色标记")
	}
	if !strings.Contains(dot, "penwidth=2.0") {
		t.Error("循环依赖边应该用粗线显示")
	}
	if !strings.Contains(dot, "图例") {
		t.Error("应该包含图例说明")
	}

	// 验证基本的DOT结构
	if !strings.Contains(dot, "digraph DependencyGraph") {
		t.Error("DOT图应该包含图名称")
	}
	if !strings.Contains(dot, "serviceA") {
		t.Error("DOT图应该包含serviceA")
	}
	if !strings.Contains(dot, "serviceB") {
		t.Error("DOT图应该包含serviceB")
	}
}

func TestDI_GenerateDOTGraphWithComplexTopology(t *testing.T) {
	di := New[TestContext]()
	ctx := &TestContext{Config: "test"}
	di.SetCtx(ctx)

	// 创建复杂的拓扑：根节点、中间节点、叶节点
	Provide(di, "rootService", func(ctx *TestContext) *ServiceA {
		return &ServiceA{Name: "Root"}
	})

	Provide(di, "middleService", func(ctx *TestContext) *ServiceB {
		root := MustMake[TestContext, ServiceA](di, "rootService")
		return &ServiceB{Name: "Middle", ServiceA: root}
	})

	Provide(di, "leafService", func(ctx *TestContext) *ServiceC {
		middle := MustMake[TestContext, ServiceB](di, "middleService")
		root := MustMake[TestContext, ServiceA](di, "rootService")
		return &ServiceC{Name: "Leaf", ServiceA: root, ServiceB: middle}
	})

	err := di.Build()
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}

	// 生成DOT图
	dot := di.GenerateDOTGraph()
	t.Logf("复杂拓扑DOT图:\n%s", dot)

	// 验证不同类型节点的颜色标记
	if !strings.Contains(dot, "lightgreen") {
		t.Error("根节点应该用绿色(lightgreen)标记")
	}
	if !strings.Contains(dot, "🌱") {
		t.Error("根节点应该有🌱标记")
	}
	if !strings.Contains(dot, "lightyellow") {
		t.Error("叶节点应该用黄色(lightyellow)标记")
	}
	if !strings.Contains(dot, "🍃") {
		t.Error("叶节点应该有🍃标记")
	}
	if !strings.Contains(dot, "lightblue") {
		t.Error("中间节点应该用蓝色(lightblue)标记")
	}

	// 验证没有循环依赖警告
	if strings.Contains(dot, "图例") {
		t.Error("无循环依赖时不应该显示图例")
	}
	if strings.Contains(dot, "color=red") {
		t.Error("无循环依赖时不应该有红色边")
	}
}

func TestDI_PrintDependencyGraph(t *testing.T) {
	di := New[TestContext]()
	ctx := &TestContext{Config: "test"}
	di.SetCtx(ctx)

	// 注册服务
	Provide(di, "serviceA", func(ctx *TestContext) *ServiceA {
		return &ServiceA{Name: "ServiceA"}
	})

	Provide(di, "serviceB", func(ctx *TestContext) *ServiceB {
		serviceA := MustMake[TestContext, ServiceA](di, "serviceA")
		return &ServiceB{
			Name:     "ServiceB",
			ServiceA: serviceA,
		}
	})

	// 构建服务
	err := di.Build()
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}

	// 打印依赖图谱
	output := di.PrintDependencyGraph()
	t.Logf("依赖图谱输出:\n%s", output)

	// 验证输出包含必要信息
	if !strings.Contains(output, "依赖图谱") {
		t.Error("输出应该包含标题")
	}
	if !strings.Contains(output, "serviceA") {
		t.Error("输出应该包含serviceA")
	}
	if !strings.Contains(output, "serviceB") {
		t.Error("输出应该包含serviceB")
	}
}

func TestDI_TryMake(t *testing.T) {
	di := New[TestContext]()
	ctx := &TestContext{Config: "test"}
	di.SetCtx(ctx)

	// 注册服务
	Provide(di, "serviceA", func(ctx *TestContext) *ServiceA {
		return &ServiceA{Name: "ServiceA"}
	})

	// 构建服务
	err := di.Build()
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}

	// 测试存在的服务
	serviceA, ok := TryMake[TestContext, ServiceA](di, "serviceA")
	if !ok {
		t.Error("应该成功获取serviceA")
	}
	if serviceA == nil {
		t.Error("serviceA不应该为nil")
	}

	// 测试不存在的服务
	_, ok = TryMake[TestContext, ServiceA](di, "nonexistent")
	if ok {
		t.Error("不存在的服务应该返回false")
	}
}

func TestDI_MustMakePanic(t *testing.T) {
	di := New[TestContext]()
	ctx := &TestContext{Config: "test"}
	di.SetCtx(ctx)

	// 测试MustMake对不存在服务的panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustMake应该对不存在的服务panic")
		}
	}()

	MustMake[TestContext, ServiceA](di, "nonexistent")
}

func TestDI_BuildTwice(t *testing.T) {
	di := New[TestContext]()
	ctx := &TestContext{Config: "test"}
	di.SetCtx(ctx)

	// 注册服务
	Provide(di, "serviceA", func(ctx *TestContext) *ServiceA {
		return &ServiceA{Name: "ServiceA"}
	})

	// 第一次构建
	err := di.Build()
	if err != nil {
		t.Fatalf("第一次构建失败: %v", err)
	}

	// 第二次构建应该成功（不重复构建）
	err = di.Build()
	if err != nil {
		t.Fatalf("第二次构建失败: %v", err)
	}

	// 验证服务仍然可用
	serviceA := MustMake[TestContext, ServiceA](di, "serviceA")
	if serviceA.Name != "ServiceA" {
		t.Errorf("期望服务名称为 'ServiceA'，实际为 '%s'", serviceA.Name)
	}
}

// 基准测试
func BenchmarkDI_Build(b *testing.B) {
	for i := 0; i < b.N; i++ {
		di := New[TestContext]()
		ctx := &TestContext{Config: "test"}
		di.SetCtx(ctx)

		// 注册多个服务
		for j := 0; j < 10; j++ {
			serviceName := "service" + string(rune('A'+j))
			Provide(di, serviceName, func(ctx *TestContext) *ServiceA {
				return &ServiceA{Name: serviceName}
			})
		}

		di.Build()
	}
}

func BenchmarkDI_GetDependencyGraph(b *testing.B) {
	di := New[TestContext]()
	ctx := &TestContext{Config: "test"}
	di.SetCtx(ctx)

	// 注册服务
	Provide(di, "serviceA", func(ctx *TestContext) *ServiceA {
		return &ServiceA{Name: "ServiceA"}
	})

	Provide(di, "serviceB", func(ctx *TestContext) *ServiceB {
		serviceA := MustMake[TestContext, ServiceA](di, "serviceA")
		return &ServiceB{
			Name:     "ServiceB",
			ServiceA: serviceA,
		}
	})

	di.Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		di.GetDependencyGraph()
	}
}

func TestDI_ProvideMethod(t *testing.T) {
	di := New[TestContext]()
	ctx := &TestContext{Config: "test"}
	di.SetCtx(ctx)

	// 使用新的Register方法注册服务
	Provide(di, "userService", func(ctx *TestContext) *ServiceA {
		return &ServiceA{Name: "User"}
	})

	Provide(di, "orderService", func(ctx *TestContext) *ServiceB {
		user := MustMake[TestContext, ServiceA](di, "userService")
		return &ServiceB{
			Name:     "Order",
			ServiceA: user,
		}
	})

	// 构建服务
	err := di.Build()
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}

	// 验证服务
	userService := MustMake[TestContext, ServiceA](di, "userService")
	if userService.Name != "User" {
		t.Errorf("期望用户服务名称为 'User'，实际为 '%s'", userService.Name)
	}

	orderService := MustMake[TestContext, ServiceB](di, "orderService")
	if orderService.Name != "Order" {
		t.Errorf("期望订单服务名称为 'Order'，实际为 '%s'", orderService.Name)
	}
	if orderService.ServiceA.Name != "User" {
		t.Errorf("期望依赖的用户服务名称为 'User'，实际为 '%s'", orderService.ServiceA.Name)
	}

	t.Log("✅ Provide方法测试通过")
}

func ExampleProvide() {
	// 创建DI容器
	di := New[TestContext]()
	ctx := &TestContext{Config: "production"}
	di.SetCtx(ctx)

	// 使用Register方法注册服务（推荐方式）
	Provide(di, "database", func(ctx *TestContext) *ServiceA {
		return &ServiceA{Name: "PostgreSQL"}
	})

	Provide(di, "userService", func(ctx *TestContext) *ServiceB {
		db := MustMake[TestContext, ServiceA](di, "database")
		return &ServiceB{
			Name:     "UserService",
			ServiceA: db,
		}
	})

	// 构建所有服务
	di.Build()

	// 获取服务
	userService := MustMake[TestContext, ServiceB](di, "userService")
	fmt.Printf("服务名称: %s\n", userService.Name)
	fmt.Printf("数据库: %s\n", userService.ServiceA.Name)

	// Output:
	// 服务名称: UserService
	// 数据库: PostgreSQL
}

func TestDI_Extract(t *testing.T) {
	di := New[TestContext]()
	ctx := &TestContext{Config: "test"}
	di.SetCtx(ctx)

	// 注册服务
	Provide(di, "serviceA", func(ctx *TestContext) *ServiceA {
		return &ServiceA{Name: "ServiceA"}
	})

	Provide(di, "serviceB", func(ctx *TestContext) *ServiceB {
		serviceA := MustMake[TestContext, ServiceA](di, "serviceA")
		return &ServiceB{
			Name:     "ServiceB",
			ServiceA: serviceA,
		}
	})

	// 构建服务
	err := di.Build()
	if err != nil {
		t.Fatalf("构建失败: %v", err)
	}

	// 提取服务注册表
	registry := di.Extract()

	// 验证可以从注册表获取服务
	valA, ok := registry.Get("serviceA")
	if !ok {
		t.Error("应该成功获取serviceA")
	}
	serviceA, ok := valA.(*ServiceA)
	if !ok {
		t.Error("serviceA应该是一个*ServiceA")
	}
	if serviceA.Name != "ServiceA" {
		t.Errorf("期望服务名称为 'ServiceA'，实际为 '%s'", serviceA.Name)
	}

	serviceB, ok := TryGetFromRegistry[ServiceB](registry, "serviceB")
	if !ok {
		t.Error("应该成功获取serviceB")
	}
	if serviceB.Name != "ServiceB" {
		t.Errorf("期望服务名称为 'ServiceB'，实际为 '%s'", serviceB.Name)
	}
	if serviceB.ServiceA.Name != "ServiceA" {
		t.Errorf("期望依赖的服务名称为 'ServiceA'，实际为 '%s'", serviceB.ServiceA.Name)
	}

	t.Log("✅ Extract功能测试通过")
}
