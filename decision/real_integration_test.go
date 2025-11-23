package decision

import (
	"fmt"
	"nofx/market"
	"nofx/mcp"
	"strings"
	"testing"
	"time"
)

// ⚠️ 配置区：请填入您的真实 API Keys
// ⚠️ 注意：此文件仅用于本地测试，不要提交到代码库
const (
	// LLM API 配置
	LLM_BASE_URL = "https://api.deepseek.com/v1"         // 填入 LLM Base URL，例如: https://api.deepseek.com/v1
	LLM_API_KEY  = "sk-bacb6afe04d34fdfbd391c1afc18abd1" // 填入 LLM API Key
	LLM_MODEL    = "deepseek-chat"                       // 填入模型名，例如: deepseek-chat
)

// TestRealIntegration_FullDecisionFlow 完整决策流程的真实环境集成测试
//
// 此测试会：
// 1. 通过币安 API 获取真实市场数据
// 2. 调用真实 LLM（DeepSeek/Qwen）获取决策
// 3. 解析并验证决策结果
// 4. 输出详细的日志和结果
//
// 运行方式：
//
//	go test ./decision -v -run TestRealIntegration
func TestRealIntegration_FullDecisionFlow(t *testing.T) {
	// 检查是否配置了 API Keys
	if LLM_API_KEY == "" {
		t.Skip("⚠️  请先在测试文件顶部配置 LLM_API_KEY 才能运行此测试")
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🚀 开始真实环境完整决策流程测试")
	fmt.Println(strings.Repeat("=", 80))

	// ========================================================================
	// 步骤 1: 准备测试账户数据（Mock）
	// ========================================================================
	fmt.Println("\n💰 [1/6] 准备测试账户数据...")

	// 注意：为避免循环依赖，这里使用 Mock 账户数据
	// 如果需要真实账户数据，请在 trader 包中创建独立的集成测试
	mockAccountEquity := 10000.0
	mockAvailableBalance := 8000.0

	fmt.Printf("✅ 账户数据准备完成\n")
	fmt.Printf("   测试账户净值: %.2f USDT (Mock)\n", mockAccountEquity)
	fmt.Printf("   测试可用余额: %.2f USDT (Mock)\n", mockAvailableBalance)

	// ========================================================================
	// 步骤 2: 初始化市场数据监控
	// ========================================================================
	fmt.Println("\n📊 [2/6] 初始化市场数据监控...")

	testSymbols := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}

	// 初始化 WebSocket 监控客户端
	monitor := market.NewWSMonitor(len(testSymbols))
	if err := monitor.Initialize(testSymbols); err != nil {
		t.Fatalf("❌ 市场监控初始化失败: %v", err)
	}

	// 测试结束时关闭监控
	defer func() {
		fmt.Println("\n🔌 关闭市场监控...")
		monitor.Close()
	}()

	// 等待数据稳定
	fmt.Println("   等待市场数据稳定...")
	time.Sleep(5 * time.Second)

	fmt.Println("✅ 市场监控初始化完成")

	// 获取市场数据
	fmt.Println("\n📊 [2.5/6] 获取真实市场数据...")
	marketDataMap := make(map[string]*market.Data)

	for _, symbol := range testSymbols {
		data, err := market.Get(symbol)
		if err != nil {
			t.Logf("⚠️  获取 %s 数据失败: %v", symbol, err)
			continue
		}
		marketDataMap[symbol] = data
		fmt.Printf("   %s: $%.2f (1h: %+.2f%%) MACD:%.2f RSI:%.2f\n",
			symbol,
			data.CurrentPrice,
			data.PriceChange1h,
			data.CurrentMACD,
			data.CurrentRSI7,
		)
	}

	if len(marketDataMap) == 0 {
		t.Fatal("❌ 未能获取任何市场数据")
	}

	// ========================================================================
	// 步骤 3: 构建决策上下文
	// ========================================================================
	fmt.Println("\n🔧 [3/6] 构建决策上下文...")

	ctx := &Context{
		CurrentTime: time.Now().Format("2006-01-02 15:04:05"),
		Account: AccountInfo{
			TotalEquity:      mockAccountEquity,
			AvailableBalance: mockAvailableBalance,
		},
		Positions: []PositionInfo{}, // Mock：无持仓
		CandidateCoins: []CandidateCoin{
			{Symbol: "BTCUSDT", Sources: []string{"default"}},
			{Symbol: "ETHUSDT", Sources: []string{"default"}},
			{Symbol: "SOLUSDT", Sources: []string{"default"}},
		},
		MarketDataMap: marketDataMap,
	}

	// Mock：无持仓
	fmt.Printf("   当前持仓数量: 0 (Mock)\n")

	fmt.Printf("   候选币种数量: %d\n", len(ctx.CandidateCoins))
	fmt.Printf("   市场数据完整度: %d/%d\n", len(marketDataMap), len(testSymbols))

	// ========================================================================
	// 步骤 4: 初始化 LLM 客户端
	// ========================================================================
	fmt.Println("\n🤖 [4/6] 初始化 LLM 客户端...")

	// 使用通用客户端
	llmClient := mcp.New()
	llmClient.SetAPIKey(LLM_API_KEY, LLM_BASE_URL, LLM_MODEL)
	llmClient.SetTimeout(120 * time.Second)

	fmt.Printf("   使用 MCP 通用客户端\n")
	fmt.Printf("   Base URL: %s\n", LLM_BASE_URL)
	fmt.Printf("   Model: %s\n", LLM_MODEL)

	// ========================================================================
	// 步骤 5: 调用完整决策流程
	// ========================================================================
	fmt.Println("\n⚡ [5/6] 调用完整决策流程...")
	fmt.Println("   正在获取 AI 决策，请稍候...")

	startTime := time.Now()

	fullDecision, err := GetFullDecisionWithCustomPrompt(
		ctx,
		llmClient,
		"v_5_5_1", // customPrompt - 使用默认
		false,     // overrideBasePrompt
		"default",
	)

	elapsed := time.Since(startTime)

	if err != nil {
		t.Fatalf("❌ 决策流程失败: %v", err)
	}

	fmt.Printf("✅ 决策完成，总耗时: %.2fs\n", elapsed.Seconds())
	if fullDecision.AIRequestDurationMs > 0 {
		fmt.Printf("   其中 LLM 调用耗时: %dms\n", fullDecision.AIRequestDurationMs)
	}

	// ========================================================================
	// 步骤 6: 输出详细结果
	// ========================================================================
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("📋 测试结果")
	fmt.Println(strings.Repeat("=", 80))

	// 6.1 市场数据快照
	fmt.Println("\n📊 市场数据快照:")
	fmt.Println(strings.Repeat("-", 80))
	for symbol, data := range marketDataMap {
		fmt.Printf("  %s:\n", symbol)
		fmt.Printf("    当前价格: $%.2f\n", data.CurrentPrice)
		fmt.Printf("    1小时涨跌: %+.2f%%\n", data.PriceChange1h)
		fmt.Printf("    4小时涨跌: %+.2f%%\n", data.PriceChange4h)
		fmt.Printf("    MACD: %.2f\n", data.CurrentMACD)
		fmt.Printf("    RSI(7): %.2f\n", data.CurrentRSI7)
		fmt.Printf("    资金费率: %.6f\n", data.FundingRate)
		if data.OpenInterest != nil {
			fmt.Printf("    持仓量: %.2f (平均: %.2f)\n", data.OpenInterest.Latest, data.OpenInterest.Average)
		}
		fmt.Println()
	}

	// 6.2 LLM 原始响应（完整内容）
	fmt.Println("\n📜 LLM 原始响应 (完整内容):")
	fmt.Println(strings.Repeat("-", 80))

	// 重新构建完整响应（包含思维链和决策）
	fullResponse := ""
	if fullDecision.CoTTrace != "" {
		fullResponse += fullDecision.CoTTrace + "\n\n"
	}

	// 注意：原始 JSON 响应在解析后无法完整恢复，这里展示解析后的结构
	fmt.Println(fullResponse)

	// 如果响应很长，截断显示
	if len(fullResponse) > 2000 {
		fmt.Printf("%s\n... (响应过长，已截断，完整内容共 %d 字符)\n", fullResponse[:2000], len(fullResponse))
	} else {
		fmt.Println(fullResponse)
	}

	// 6.3 解析后的思维链
	fmt.Println("\n💭 解析后的思维链:")
	fmt.Println(strings.Repeat("-", 80))
	if fullDecision.CoTTrace != "" {
		// 如果思维链很长，只显示前500字符
		cot := fullDecision.CoTTrace
		if len(cot) > 500 {
			fmt.Printf("%s\n... (思维链过长，已截断，完整内容共 %d 字符)\n", cot[:500], len(cot))
		} else {
			fmt.Println(cot)
		}
	} else {
		fmt.Println("  (无思维链内容)")
	}

	// 6.4 解析后的决策列表
	fmt.Println("\n📋 解析后的决策列表:")
	fmt.Println(strings.Repeat("-", 80))
	if len(fullDecision.Decisions) == 0 {
		fmt.Println("  (无决策，可能触发了 wait 或 SafeFallback)")
	} else {
		for i, d := range fullDecision.Decisions {
			fmt.Printf("\n  [决策 %d]\n", i+1)
			fmt.Printf("    币种: %s\n", d.Symbol)
			fmt.Printf("    操作: %s\n", d.Action)

			if d.Action == "open_long" || d.Action == "open_short" {
				fmt.Printf("    杠杆: %dx\n", d.Leverage)
				fmt.Printf("    仓位大小: %.2f USDT\n", d.PositionSizeUSD)
				fmt.Printf("    止损价格: %.2f\n", d.StopLoss)
				fmt.Printf("    止盈价格: %.2f\n", d.TakeProfit)
				if d.Confidence > 0 {
					fmt.Printf("    信心度: %d%%\n", d.Confidence)
				}
				if d.RiskUSD > 0 {
					fmt.Printf("    风险金额: %.2f USDT\n", d.RiskUSD)
				}
			} else if d.Action == "update_stop_loss" {
				fmt.Printf("    新止损价格: %.2f\n", d.NewStopLoss)
			} else if d.Action == "update_take_profit" {
				fmt.Printf("    新止盈价格: %.2f\n", d.NewTakeProfit)
			} else if d.Action == "partial_close" {
				fmt.Printf("    平仓比例: %.2f%%\n", d.ClosePercentage)
			}

			fmt.Printf("    理由: %s\n", d.Reasoning)
		}
	}

	// 6.5 性能指标
	fmt.Println("\n⏱️  性能指标:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("  总耗时: %.2fs\n", elapsed.Seconds())
	if fullDecision.AIRequestDurationMs > 0 {
		fmt.Printf("  LLM 调用耗时: %dms (%.1f%%)\n",
			fullDecision.AIRequestDurationMs,
			float64(fullDecision.AIRequestDurationMs)/float64(elapsed.Milliseconds())*100,
		)
	}
	fmt.Printf("  决策数量: %d\n", len(fullDecision.Decisions))
	fmt.Printf("  候选币种数量: %d\n", len(ctx.CandidateCoins))

	// ========================================================================
	// 验证结果
	// ========================================================================
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("✅ 验证结果")
	fmt.Println(strings.Repeat("=", 80))

	// 验证基本字段
	if fullDecision.CoTTrace == "" {
		t.Error("❌ 思维链为空")
	} else {
		fmt.Println("✅ 思维链提取成功")
	}

	if len(fullDecision.Decisions) == 0 {
		fmt.Println("⚠️  决策列表为空（可能是 wait 决策或 SafeFallback）")
	} else {
		fmt.Printf("✅ 决策列表提取成功，共 %d 个决策\n", len(fullDecision.Decisions))
	}

	// 验证决策格式
	for i, d := range fullDecision.Decisions {
		if d.Symbol == "" {
			t.Errorf("❌ 决策 #%d Symbol 为空", i+1)
		}
		if d.Action == "" {
			t.Errorf("❌ 决策 #%d Action 为空", i+1)
		}
		if d.Reasoning == "" {
			t.Errorf("❌ 决策 #%d Reasoning 为空", i+1)
		}
	}

	// 验证 parseFullDecisionResponse 修复
	fmt.Println("\n🔍 验证 parseFullDecisionResponse 修复:")
	fmt.Println(strings.Repeat("-", 80))

	// 检查是否成功处理了可能的格式问题
	validationPassed := true

	if len(fullDecision.Decisions) > 0 {
		// 检查决策是否被正确解析
		for _, d := range fullDecision.Decisions {
			if d.Symbol != "" && d.Action != "" {
				fmt.Println("✅ 决策字段解析正确")
				break
			}
		}
	}

	// 检查思维链是否被提取
	if fullDecision.CoTTrace != "" {
		fmt.Println("✅ 思维链提取成功（支持大小写不敏感的 XML 标签）")
	}

	// 检查是否触发了 SafeFallback
	if len(fullDecision.Decisions) == 1 &&
		fullDecision.Decisions[0].Symbol == "ALL" &&
		fullDecision.Decisions[0].Action == "wait" {
		fmt.Println("⚠️  触发了 SafeFallback 机制（LLM 未输出有效 JSON）")
		validationPassed = false
	}

	if validationPassed {
		fmt.Println("✅ 所有验证通过")
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🎉 测试完成")
	fmt.Println(strings.Repeat("=", 80) + "\n")
}

// TestRealIntegration_ParseDeepSeekResponse 专门测试 DeepSeek 响应格式的解析
func TestRealIntegration_ParseDeepSeekResponse(t *testing.T) {
	// 这个测试不需要 API Key，直接使用 Mock 响应

	fmt.Println("\n🔍 测试 DeepSeek 响应格式解析...")

	// 模拟 DeepSeek 可能返回的各种格式
	testCases := []struct {
		name     string
		response string
	}{
		{
			name: "标准格式",
			response: `<reasoning>分析内容</reasoning>
<decision>
` + "```json" + `
[{"symbol":"BTCUSDT","action":"wait","reasoning":"观望"}]
` + "```" + `
</decision>`,
		},
		{
			name: "大写标签",
			response: `<REASONING>分析内容</REASONING>
<DECISION>
` + "```json" + `
[{"symbol":"BTCUSDT","action":"wait","reasoning":"观望"}]
` + "```" + `
</DECISION>`,
		},
		{
			name: "全角字符",
			response: `<reasoning>分析</reasoning>
<decision>
［｛"symbol"："BTCUSDT"，"action"："wait"，"reasoning"："观望"｝］
</decision>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			decision, err := parseFullDecisionResponse(tc.response, 10000, 10, 5)

			if err != nil {
				t.Errorf("❌ 解析失败: %v", err)
				return
			}

			if len(decision.Decisions) == 0 {
				t.Error("❌ 决策列表为空")
				return
			}

			fmt.Printf("✅ %s 格式解析成功\n", tc.name)
		})
	}
}
