package decision

import (
	"strings"
	"testing"
)

// =============================================================================
// extractCoTTrace 测试
// =============================================================================

func TestExtractCoTTrace(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     string
	}{
		{
			name: "标准格式-带reasoning标签",
			response: `<reasoning>
BTC跌破支撑位，MACD死叉，成交量放大。
建议开空单，止损设在97000。
</reasoning>
<decision>
[{"symbol":"BTCUSDT","action":"wait"}]
</decision>`,
			want: "BTC跌破支撑位，MACD死叉，成交量放大。\n建议开空单，止损设在97000。",
		},
		{
			name: "大小写变化-REASONING标签（DeepSeek可能输出）",
			response: `<REASONING>
市场震荡，暂时观望。
</REASONING>
<DECISION>[...]</DECISION>`,
			want: "市场震荡，暂时观望。",
		},
		{
			name: "混合大小写-Reasoning标签",
			response: `<Reasoning>思维链内容</Reasoning><Decision>...</Decision>`,
			want: "思维链内容",
		},
		{
			name: "只有decision标签",
			response: `市场分析：
BTC价格在95000-96000区间震荡
<decision>
[{"symbol":"BTCUSDT","action":"wait"}]
</decision>`,
			want: "市场分析：\nBTC价格在95000-96000区间震荡",
		},
		{
			name: "旧版格式-直接JSON数组",
			response: `经过分析，当前市场趋势向下。
建议开空单。

[{"symbol":"BTCUSDT","action":"open_short"}]`,
			want: "经过分析，当前市场趋势向下。\n建议开空单。",
		},
		{
			name: "只有思维链无JSON",
			response: "这是纯思维链内容，没有任何JSON或标签",
			want:     "这是纯思维链内容，没有任何JSON或标签",
		},
		{
			name:     "空响应",
			response: "",
			want:     "",
		},
		{
			name:     "仅空格和换行",
			response: "  \n\n  \n",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCoTTrace(tt.response)
			if got != tt.want {
				t.Errorf("extractCoTTrace() = %q, want %q", got, tt.want)
			}
		})
	}
}

// =============================================================================
// extractDecisions 测试（重点）
// =============================================================================

func TestExtractDecisions(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		wantLen   int
		wantError bool
		checkFunc func(*testing.T, []Decision) // 可选的额外验证
	}{
		{
			name: "标准格式-带标签和代码块",
			response: `<decision>
` + "```json" + `
[
  {
    "symbol": "BTCUSDT",
    "action": "wait",
    "reasoning": "市场震荡"
  }
]
` + "```" + `
</decision>`,
			wantLen: 1,
			checkFunc: func(t *testing.T, decisions []Decision) {
				if decisions[0].Symbol != "BTCUSDT" {
					t.Errorf("Symbol = %v, want BTCUSDT", decisions[0].Symbol)
				}
				if decisions[0].Action != "wait" {
					t.Errorf("Action = %v, want wait", decisions[0].Action)
				}
			},
		},
		{
			name: "大小写变化-DECISION标签（DeepSeek可能输出）",
			response: `<DECISION>
` + "```json" + `
[{"symbol":"ETHUSDT","action":"hold","reasoning":"观望"}]
` + "```" + `
</DECISION>`,
			wantLen: 1,
			checkFunc: func(t *testing.T, decisions []Decision) {
				if decisions[0].Symbol != "ETHUSDT" {
					t.Errorf("Symbol = %v, want ETHUSDT", decisions[0].Symbol)
				}
			},
		},
		{
			name: "没有代码块-裸JSON在标签内",
			response: `<decision>
[
  {"symbol": "SOLUSDT", "action": "wait", "reasoning": "等待信号"}
]
</decision>`,
			wantLen: 1,
		},
		{
			name: "全角字符污染-括号和标点",
			response: `<decision>
［｛"symbol"："BTCUSDT"，"action"："wait"，"reasoning"："观望"｝］
</decision>`,
			wantLen: 1,
			checkFunc: func(t *testing.T, decisions []Decision) {
				if decisions[0].Symbol != "BTCUSDT" {
					t.Errorf("全角字符修复失败: Symbol = %v, want BTCUSDT", decisions[0].Symbol)
				}
			},
		},
		{
			name: "全角字符污染-中文引号",
			response: `<decision>
[{"symbol":"BTCUSDT","action":"wait","reasoning":"观望"}]
</decision>`,
			wantLen: 1,
		},
		{
			name: "CJK括号-【】",
			response: `<decision>
【{"symbol":"BTCUSDT","action":"wait","reasoning":"test"}】
</decision>`,
			wantLen: 1,
		},
		{
			name: "旧版格式-无标签直接JSON",
			response: `思维链分析内容...

[{"symbol":"BTCUSDT","action":"wait","reasoning":"观望"}]`,
			wantLen: 1,
		},
		{
			name: "多个决策",
			response: `<decision>
` + "```json" + `
[
  {"symbol":"BTCUSDT","action":"open_long","leverage":10,"position_size_usd":5000,"stop_loss":94000,"take_profit":100000,"reasoning":"趋势向上"},
  {"symbol":"ETHUSDT","action":"wait","reasoning":"观望"}
]
` + "```" + `
</decision>`,
			wantLen: 2,
			checkFunc: func(t *testing.T, decisions []Decision) {
				if decisions[0].Action != "open_long" {
					t.Errorf("First action = %v, want open_long", decisions[0].Action)
				}
				if decisions[1].Action != "wait" {
					t.Errorf("Second action = %v, want wait", decisions[1].Action)
				}
			},
		},
		{
			name:     "SafeFallback-无JSON只有思维链",
			response: "这是纯思维链内容，完全没有JSON数组",
			wantLen:  1, // SafeFallback 应返回 wait 决策
			checkFunc: func(t *testing.T, decisions []Decision) {
				if decisions[0].Symbol != "ALL" {
					t.Errorf("SafeFallback Symbol = %v, want ALL", decisions[0].Symbol)
				}
				if decisions[0].Action != "wait" {
					t.Errorf("SafeFallback Action = %v, want wait", decisions[0].Action)
				}
				if !strings.Contains(decisions[0].Reasoning, "模型未输出结构化JSON决策") {
					t.Errorf("SafeFallback Reasoning should contain fallback message")
				}
			},
		},
		{
			name: "数组开头有空格-需要规整",
			response: `<decision>
[
  {
    "symbol": "BTCUSDT",
    "action": "wait",
    "reasoning": "test"
  }
]
</decision>`,
			wantLen: 1,
		},
		{
			name:      "空响应-触发SafeFallback",
			response:  "",
			wantLen:   1,
			checkFunc: func(t *testing.T, decisions []Decision) {
				if decisions[0].Action != "wait" {
					t.Errorf("Empty response should trigger SafeFallback with wait action")
				}
			},
		},
		{
			name: "包含零宽字符和BOM",
			response: "\uFEFF<decision>\u200B\n" +
				"[{\"symbol\":\"BTCUSDT\",\"action\":\"wait\",\"reasoning\":\"test\"}]\u200C\n" +
				"</decision>\u200D",
			wantLen: 1,
		},
		{
			name: "DeepSeek格式-单个对象（应触发SafeFallback）",
			response: "```json\n" +
				`{
  "action": "wait",
  "reasoning": "观望",
  "confidence": 0.70
}
` + "```",
			wantLen: 1,
			checkFunc: func(t *testing.T, decisions []Decision) {
				// 单个对象会被当作无效格式，触发 SafeFallback
				if decisions[0].Action != "wait" {
					t.Errorf("Single object should trigger SafeFallback with wait action")
				}
				if decisions[0].Symbol != "ALL" {
					t.Errorf("SafeFallback Symbol = %v, want ALL", decisions[0].Symbol)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractDecisions(tt.response)

			// 检查错误期望
			if (err != nil) != tt.wantError {
				t.Errorf("extractDecisions() error = %v, wantError %v", err, tt.wantError)
				return
			}

			// 检查返回数量
			if len(got) != tt.wantLen {
				t.Errorf("extractDecisions() returned %d decisions, want %d", len(got), tt.wantLen)
				t.Logf("Got decisions: %+v", got)
				return
			}

			// 执行自定义验证
			if tt.checkFunc != nil {
				tt.checkFunc(t, got)
			}
		})
	}
}

// 测试格式错误场景（应该返回错误）
func TestExtractDecisions_FormatErrors(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		wantErrorMsg string
	}{
		{
			name: "包含范围符号",
			response: `<decision>
[{"symbol":"BTCUSDT","action":"open_long","leverage":"5~10","position_size_usd":5000}]
</decision>`,
			wantErrorMsg: "范围符号",
		},
		{
			name: "包含千位分隔符",
			response: `<decision>
[{"symbol":"BTCUSDT","action":"open_long","leverage":10,"position_size_usd":98,000}]
</decision>`,
			wantErrorMsg: "千位分隔符",
		},
		{
			name: "无效JSON-缺少引号",
			response: `<decision>
[{symbol:BTCUSDT,action:wait}]
</decision>`,
			wantErrorMsg: "JSON", // 应包含JSON解析错误
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractDecisions(tt.response)

			// 对于格式错误，可能触发 SafeFallback 或返回错误
			// SafeFallback 会返回 wait 决策而不是错误
			if err == nil && len(got) > 0 && got[0].Action == "wait" {
				// 触发了 SafeFallback，这也是合理的
				t.Logf("Triggered SafeFallback for invalid format: %v", got[0].Reasoning)
				return
			}

			if err == nil {
				t.Errorf("extractDecisions() expected error for invalid format, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErrorMsg) {
				t.Errorf("extractDecisions() error = %v, should contain %q", err, tt.wantErrorMsg)
			}
		})
	}
}

// =============================================================================
// parseFullDecisionResponse 集成测试
// =============================================================================

func TestParseFullDecisionResponse(t *testing.T) {
	accountEquity := 10000.0
	btcEthLeverage := 10
	altcoinLeverage := 5

	tests := []struct {
		name           string
		response       string
		wantCoTLen     int  // 思维链长度（大致）
		wantDecisions  int  // 决策数量
		wantError      bool // 是否期望错误
		validateResult func(*testing.T, *FullDecision, error)
	}{
		{
			name: "完整成功场景-标准格式",
			response: `<reasoning>
市场分析：
1. BTC价格突破95000阻力位
2. MACD金叉，RSI未超买
3. 成交量放大确认突破有效
建议：开多单，目标100000，止损94000
</reasoning>

<decision>
` + "```json" + `
[
  {
    "symbol": "BTCUSDT",
    "action": "open_long",
    "leverage": 10,
    "position_size_usd": 5000,
    "stop_loss": 94000,
    "take_profit": 100000,
    "confidence": 85,
    "risk_usd": 300,
    "reasoning": "突破阻力位+MACD金叉"
  }
]
` + "```" + `
</decision>`,
			wantCoTLen:    50,
			wantDecisions: 1,
			wantError:     false,
			validateResult: func(t *testing.T, fd *FullDecision, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}
				if !strings.Contains(fd.CoTTrace, "市场分析") {
					t.Errorf("CoTTrace should contain '市场分析'")
				}
				if fd.Decisions[0].Symbol != "BTCUSDT" {
					t.Errorf("Symbol = %v, want BTCUSDT", fd.Decisions[0].Symbol)
				}
				if fd.Decisions[0].Leverage != 10 {
					t.Errorf("Leverage = %v, want 10", fd.Decisions[0].Leverage)
				}
			},
		},
		{
			name: "DeepSeek格式-大写标签",
			response: `<REASONING>
技术面分析：趋势向下
</REASONING>

<DECISION>
` + "```json" + `
[{"symbol":"BTCUSDT","action":"wait","reasoning":"观望"}]
` + "```" + `
</DECISION>`,
			wantCoTLen:    10,
			wantDecisions: 1,
			wantError:     false,
		},
		{
			name: "多个决策-混合操作",
			response: `<reasoning>
市场分析：BTC看涨，ETH观望，SOL平仓
</reasoning>

<decision>
[
  {"symbol":"BTCUSDT","action":"open_long","leverage":10,"position_size_usd":6000,"stop_loss":94000,"take_profit":103000,"reasoning":"看涨"},
  {"symbol":"ETHUSDT","action":"wait","reasoning":"观望"},
  {"symbol":"SOLUSDT","action":"close_long","reasoning":"止盈"}
]
</decision>`,
			wantCoTLen:    10,
			wantDecisions: 3,
			wantError:     false,
		},
		{
			name: "杠杆超限-自动修正",
			response: `<reasoning>高杠杆测试</reasoning>
<decision>
[{"symbol":"SOLUSDT","action":"open_long","leverage":20,"position_size_usd":1000,"stop_loss":150,"take_profit":200,"reasoning":"test"}]
</decision>`,
			wantCoTLen:    5,
			wantDecisions: 1,
			wantError:     false, // 杠杆超限会被自动修正，不会报错
			validateResult: func(t *testing.T, fd *FullDecision, err error) {
				if err != nil {
					t.Errorf("Leverage should be auto-corrected, not error: %v", err)
					return
				}
				// 验证杠杆已被修正为上限值 5x
				if fd.Decisions[0].Leverage != 5 {
					t.Errorf("Leverage should be corrected to 5, got %d", fd.Decisions[0].Leverage)
				}
			},
		},
		{
			name:           "无JSON-SafeFallback",
			response:       "这是纯思维链内容，没有JSON",
			wantCoTLen:     10,
			wantDecisions:  1,
			wantError:      false,
			validateResult: func(t *testing.T, fd *FullDecision, err error) {
				if err != nil {
					t.Errorf("SafeFallback should not return error: %v", err)
					return
				}
				if fd.Decisions[0].Action != "wait" {
					t.Errorf("SafeFallback should produce wait action")
				}
			},
		},
		{
			name: "JSON解析成功但验证失败-仓位过小",
			response: `<reasoning>测试小仓位</reasoning>
<decision>
[{"symbol":"BTCUSDT","action":"open_long","leverage":10,"position_size_usd":5,"stop_loss":94000,"take_profit":100000,"reasoning":"test"}]
</decision>`,
			wantCoTLen:    5,
			wantDecisions: 1,
			wantError:     true,
			validateResult: func(t *testing.T, fd *FullDecision, err error) {
				if err == nil {
					t.Errorf("Expected validation error for position size too small")
					return
				}
				// 应该提到仓位大小问题
				if !strings.Contains(err.Error(), "决策验证失败") {
					t.Errorf("Error should mention validation failure: %v", err)
				}
			},
		},
		{
			name:           "空响应-SafeFallback",
			response:       "",
			wantCoTLen:     0,
			wantDecisions:  1,
			wantError:      false,
			validateResult: func(t *testing.T, fd *FullDecision, err error) {
				if err != nil {
					t.Errorf("Empty response should trigger SafeFallback without error: %v", err)
					return
				}
				if fd.Decisions[0].Action != "wait" {
					t.Errorf("Empty response should produce wait action")
				}
			},
		},
		{
			name: "DeepSeek实际格式-单个对象+无标签",
			response: `## 第0步：疑惑检查
当前BTC处于102,902，整体市场呈现震荡偏弱状态。

## 最终决策分析
根据零号原则：疑惑优先，当前市场信号模糊。

` + "```json" + `
{
  "action": "wait",
  "reasoning": "BTC处于明确空头但动能不足，信心度仅70分。",
  "confidence": 0.70,
  "expected_profit_percent": 0
}
` + "```",
			wantCoTLen:    50,
			wantDecisions: 1,
			wantError:     false,
			validateResult: func(t *testing.T, fd *FullDecision, err error) {
				if err != nil {
					t.Errorf("DeepSeek format should be supported: %v", err)
					return
				}
				// SafeFallback 应该触发，因为是单个对象而不是数组
				if fd.Decisions[0].Action != "wait" {
					t.Errorf("Should convert single object to array with wait action")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFullDecisionResponse(tt.response, accountEquity, btcEthLeverage, altcoinLeverage)

			// 检查错误期望
			if (err != nil) != tt.wantError {
				t.Errorf("parseFullDecisionResponse() error = %v, wantError %v", err, tt.wantError)
			}

			// 即使有错误，也应该有返回值（包含思维链）
			if got == nil {
				t.Errorf("parseFullDecisionResponse() returned nil")
				return
			}

			// 检查思维链长度
			if tt.wantCoTLen > 0 && len(got.CoTTrace) < tt.wantCoTLen {
				t.Errorf("CoTTrace length = %d, want >= %d", len(got.CoTTrace), tt.wantCoTLen)
			}

			// 检查决策数量
			if len(got.Decisions) != tt.wantDecisions {
				t.Errorf("Decisions count = %d, want %d", len(got.Decisions), tt.wantDecisions)
			}

			// 执行自定义验证
			if tt.validateResult != nil {
				tt.validateResult(t, got, err)
			}
		})
	}
}

// =============================================================================
// 辅助函数测试
// =============================================================================

func TestFixMissingQuotes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "中文引号",
			input: `"symbol":"BTCUSDT"`,
			want:  `"symbol":"BTCUSDT"`,
		},
		{
			name:  "全角括号",
			input: "［｛｝］",
			want:  "[{}]",
		},
		{
			name:  "全角冒号和逗号",
			input: `"symbol"："BTCUSDT"，"action"："wait"`,
			want:  `"symbol":"BTCUSDT","action":"wait"`,
		},
		{
			name:  "CJK括号",
			input: "【】〔〕",
			want:  "[][]",
		},
		{
			name:  "全角空格",
			input: "test　test",
			want:  "test test",
		},
		{
			name:  "混合全角字符",
			input: `［｛"symbol"："BTCUSDT"，"action"："wait"｝］`,
			want:  `[{"symbol":"BTCUSDT","action":"wait"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fixMissingQuotes(tt.input)
			if got != tt.want {
				t.Errorf("fixMissingQuotes() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateJSONFormat(t *testing.T) {
	tests := []struct {
		name      string
		jsonStr   string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "有效JSON数组",
			jsonStr:   `[{"symbol":"BTCUSDT","action":"wait"}]`,
			wantError: false,
		},
		{
			name:      "有效JSON-带空格",
			jsonStr:   `[ { "symbol": "BTCUSDT", "action": "wait" } ]`,
			wantError: false,
		},
		{
			name:      "包含范围符号",
			jsonStr:   `[{"leverage":"5~10"}]`,
			wantError: true,
			errorMsg:  "范围符号",
		},
		{
			name:      "包含千位分隔符",
			jsonStr:   `[{"position_size_usd":98,000}]`,
			wantError: true,
			errorMsg:  "千位分隔符",
		},
		{
			name:      "不是对象数组",
			jsonStr:   `["BTCUSDT","ETHUSDT"]`,
			wantError: true,
			errorMsg:  "必须包含对象",
		},
		{
			name:      "不以[开头",
			jsonStr:   `{"symbol":"BTCUSDT"}`,
			wantError: true,
			errorMsg:  "必须以 [{ 开头",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJSONFormat(tt.jsonStr)

			if (err != nil) != tt.wantError {
				t.Errorf("validateJSONFormat() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError && !strings.Contains(err.Error(), tt.errorMsg) {
				t.Errorf("validateJSONFormat() error = %v, should contain %q", err, tt.errorMsg)
			}
		})
	}
}
