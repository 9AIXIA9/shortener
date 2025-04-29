package sensitive

import (
	"os"
	"strings"
	"testing"
)

// 创建临时测试文件
func createTempFile(content string) (string, error) {
	tmpFile, err := os.CreateTemp("", "test")
	if err != nil {
		return "", err
	}

	if _, err := tmpFile.Write([]byte(content)); err != nil {
		return "", err
	}

	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

func TestNewFilter(t *testing.T) {
	// 创建测试文件
	sensitiveContent := "test\nbad\nword\n"
	sensitiveFile, err := createTempFile(sensitiveContent)
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(sensitiveFile); err != nil {
			t.Fatalf("删除临时文件失败：%v", err)
		}
	}()

	similarCharsContent := "a=b,c\n"
	similarCharsFile, err := createTempFile(similarCharsContent)
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(similarCharsFile); err != nil {
			t.Fatalf("删除临时文件失败：%v", err)
		}
	}()

	replaceRulesContent := "1,one\n2,two\n"
	replaceRulesFile, err := createTempFile(replaceRulesContent)
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(replaceRulesFile); err != nil {
			t.Fatalf("删除临时文件失败: %v", err)
		}
	}()

	// 测试创建过滤器
	filter := NewFilter(sensitiveFile, similarCharsFile, replaceRulesFile)
	if filter == nil {
		t.Fatal("创建敏感词过滤器失败")
	}
}

func TestContainsBadWord(t *testing.T) {
	// 创建测试文件
	sensitiveContent := `# 敏感词示例
fuck
sex
porn
anal
ass
asshole
dick
bastard
bitch
`
	sensitiveFile, err := createTempFile(sensitiveContent)
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(sensitiveFile); err != nil {
			t.Fatalf("删除临时文件失败：%v", err)
		}
	}()

	similarCharsContent := `# 形近字映射
a=4,@
e=3,€
i=1,!
o=0,*
s=5,$
`
	similarCharsFile, err := createTempFile(similarCharsContent)
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(similarCharsFile); err != nil {
			t.Fatalf("删除临时文件失败：%v", err)
		}
	}()

	replaceRulesContent := `# 替换规则
**,o
++,t
__,i
`
	replaceRulesFile, err := createTempFile(replaceRulesContent)
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(replaceRulesFile); err != nil {
			t.Fatalf("删除临时文件失败: %v", err)
		}
	}()

	// 创建过滤器
	filter := NewFilter(sensitiveFile, similarCharsFile, replaceRulesFile)

	// 测试用例
	testCases := []struct {
		name   string
		input  string
		expect bool
	}{
		{"敏感词-明确", "this text contains fuck word", true},
		{"敏感词-子字符串", "this is an assembly line", true}, // 包含 "ass"
		{"敏感词-大小写", "This Is SeX education", true},     // 大写敏感词
		{"敏感词-形近字", "This is p0rn", true},              // 形近字替换: 'o' -> '0'
		{"敏感词-形近字2", "This is 5ex", true},              // 形近字替换: 's' -> '5'
		{"敏感词-替换规则", "This is p**rn", true},            // 替换规则: '**' -> 'o'
		{"敏感词-组合", "This is p**rn and d!ck", true},     // 形近字+替换规则
		{"非敏感词", "This is a clean text", false},
		{"空字符串", "", false},
		{"只有空格", "    ", false},
		{"边界-单个字符", "a", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := filter.ContainsBadWord(tc.input)
			if result != tc.expect {
				t.Errorf("ContainsBadWord(%q) = %v, 期望 %v", tc.input, result, tc.expect)
			}
		})
	}
}

func TestSimilarCharsDetection(t *testing.T) {
	// 创建测试文件
	sensitiveContent := "abc\n"
	sensitiveFile, err := createTempFile(sensitiveContent)
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(sensitiveFile); err != nil {
			t.Fatalf("删除临时文件失败：%v", err)
		}
	}()

	similarCharsContent := "a=4,@\nb=8,6\nc=k,<\n"
	similarCharsFile, err := createTempFile(similarCharsContent)
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(similarCharsFile); err != nil {
			t.Fatalf("删除临时文件失败：%v", err)
		}
	}()

	replaceRulesFile, err := createTempFile("")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(replaceRulesFile); err != nil {
			t.Fatalf("删除临时文件失败: %v", err)
		}
	}()

	// 创建过滤器
	filter := NewFilter(sensitiveFile, similarCharsFile, replaceRulesFile)

	// 测试形近字检测
	testCases := []struct {
		input  string
		expect bool
	}{
		{"abc", true},    // 原始敏感词
		{"4bc", true},    // a -> 4
		{"@bc", true},    // a -> @
		{"a8c", true},    // b -> 8
		{"ab<", true},    // c -> <
		{"48<", true},    // 多个替换
		{"xyz", false},   // 无关内容
		{"a b c", false}, // 分开的字符
		{"4b<", true},    // 部分替换
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := filter.ContainsBadWord(tc.input)
			if result != tc.expect {
				t.Errorf("ContainsBadWord(%q) = %v, 期望 %v", tc.input, result, tc.expect)
			}
		})
	}
}

func BenchmarkContainsBadWord(b *testing.B) {
	// 创建测试文件
	sensitiveContent := `# 敏感词示例
fuck
sex
porn
anal
ass
asshole
dick
cock
bastard
bitch
shit
`
	sensitiveFile, err := createTempFile(sensitiveContent)
	if err != nil {
		b.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(sensitiveFile); err != nil {
			b.Fatalf("删除临时文件失败：%v", err)
		}
	}()

	similarCharsContent := `a=4,@
e=3,€
i=1,!
o=0,*
s=5,$
`
	similarCharsFile, err := createTempFile(similarCharsContent)
	if err != nil {
		b.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(similarCharsFile); err != nil {
			b.Fatalf("删除临时文件失败：%v", err)
		}
	}()

	replaceRulesContent := `**,o
++,t
__,i
`
	replaceRulesFile, err := createTempFile(replaceRulesContent)
	if err != nil {
		b.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(replaceRulesFile); err != nil {
			b.Fatalf("删除临时文件失败: %v", err)
		}
	}()

	filter := NewFilter(sensitiveFile, similarCharsFile, replaceRulesFile)

	// 各种测试文本
	texts := []string{
		"这是一段正常文本，不包含任何敏感词。",
		"这段文本包含敏感词: sex。",
		"这段文本隐藏了敏感词: f*ck。",
		"这段文本使用了形近字: 5h!t。",
		"这是一段很长的文本，可能在某处包含敏感词。看看能否找到像 a55hole 这样伪装在段落中间的词。即使使用各种技巧也应该被检测到。",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, text := range texts {
			filter.ContainsBadWord(text)
		}
	}
}

func TestReplaceRules(t *testing.T) {
	// 创建测试文件
	sensitiveContent := "sensitive\ntest\n"
	sensitiveFile, err := createTempFile(sensitiveContent)
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(sensitiveFile); err != nil {
			t.Fatalf("删除临时文件失败：%v", err)
		}
	}()

	similarCharsFile, err := createTempFile("")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(similarCharsFile); err != nil {
			t.Fatalf("删除临时文件失败：%v", err)
		}
	}()

	replaceRulesContent := `s e n s i t i v e,sensitive
t-e-s-t,test
+,t
#,e
$,s
`
	replaceRulesFile, err := createTempFile(replaceRulesContent)
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(replaceRulesFile); err != nil {
			t.Fatalf("删除临时文件失败: %v", err)
		}
	}()

	filter := NewFilter(sensitiveFile, similarCharsFile, replaceRulesFile)

	testCases := []struct {
		name   string
		input  string
		expect bool
	}{
		{"原文本", "sensitive", true},
		{"原文本2", "test", true},
		{"空格替换", "s e n s i t i v e", true},
		{"短线替换", "t-e-s-t", true},
		{"部分替换", "sen$itive", true}, // $ -> s
		{"无关文本", "normal", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := filter.ContainsBadWord(tc.input)
			if result != tc.expect {
				t.Errorf("ContainsBadWord(%q) = %v, 期望 %v", tc.input, result, tc.expect)
			}
		})
	}
}

func TestRealSensitiveWordsList(t *testing.T) {
	// 创建包含敏感词的临时文件
	sensitiveContent := `# 敏感词列表
ass
penis
fuck
`
	sensitiveFile, err := createTempFile(sensitiveContent)
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(sensitiveFile); err != nil {
			t.Fatalf("删除临时文件失败：%v", err)
		}
	}()

	// 使用临时文件创建过滤器
	filter := NewFilter(sensitiveFile, "", "")

	testCases := []struct {
		input  string
		expect bool
	}{
		{"short-url-ass-example", true},
		{"my-penis-site", true},
		{"fuck-this-url", true},
		{"normal-website-url", false},
		{"programming-url-example", false},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := filter.ContainsBadWord(tc.input)
			if result != tc.expect {
				t.Errorf("ContainsBadWord(%q) = %v, 期望 %v", tc.input, result, tc.expect)
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	// 创建测试文件
	sensitiveContent := "a\nab\nabc\n"
	sensitiveFile, err := createTempFile(sensitiveContent)
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(sensitiveFile); err != nil {
			t.Fatalf("删除临时文件失败：%v", err)
		}
	}()

	filter := NewFilter(sensitiveFile, "", "")

	// 准备一个非常长的文本，在最后包含敏感词
	longText := strings.Repeat("正常文本", 1000) + "abc"

	testCases := []struct {
		name   string
		input  string
		expect bool
	}{
		{"单字符敏感词", "a", true},
		{"单字符敏感词在文本中", "这段文本包含 a。", true},
		{"短敏感词", "ab", true},
		{"长敏感词", "abc", true},
		{"空字符串", "", false},
		{"只有空格", "    ", false},
		{"标点符号", ".,;:!?", false},
		{"特殊字符", "\t\n\r", false},
		{"中文字符", "你好世界", false},
		{"包含敏感词的中文", "你好a世界", true},
		{"超长文本", longText, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := filter.ContainsBadWord(tc.input)
			if result != tc.expect {
				t.Errorf("ContainsBadWord(%q) = %v, 期望 %v", tc.input, result, tc.expect)
			}
		})
	}
}

func TestInvalidFileFormat(t *testing.T) {
	// 创建格式错误的文件
	invalidSimilarContent := "invalid format\na=b=c\n=d\n"
	similarCharsFile, err := createTempFile(invalidSimilarContent)
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(similarCharsFile); err != nil {
			t.Fatalf("删除临时文件失败：%v", err)
		}
	}()

	invalidReplaceContent := "invalid,one,extra\nmissing\n"
	replaceRulesFile, err := createTempFile(invalidReplaceContent)
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(replaceRulesFile); err != nil {
			t.Fatalf("删除临时文件失败: %v", err)
		}
	}()

	sensitiveContent := "bad\nword\n"
	sensitiveFile, err := createTempFile(sensitiveContent)
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer func() {
		if err := os.Remove(sensitiveFile); err != nil {
			t.Fatalf("删除临时文件失败：%v", err)
		}
	}()

	// 过滤器应该仍然能创建
	filter := NewFilter(sensitiveFile, similarCharsFile, replaceRulesFile)

	// 测试基本功能是否仍然工作
	if !filter.ContainsBadWord("bad") {
		t.Error("没有检测到敏感词")
	}
}
