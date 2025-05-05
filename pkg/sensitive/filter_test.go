package sensitive

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilter_BasicDetection(t *testing.T) {
	// 初始化测试环境
	tmpDir := t.TempDir()

	// 创建测试文件
	createTestFile := func(content string) string {
		path := filepath.Join(tmpDir, "testfile.txt")
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
		return path
	}

	t.Run("直接敏感词匹配", func(t *testing.T) {
		f := NewFilter(
			createTestFile("admin\npassword"),
			"",
			"",
		)
		require.True(t, f.ContainsBadWord("user admin"))
		require.False(t, f.ContainsBadWord("normal text"))
	})

	t.Run("大小写不敏感检测", func(t *testing.T) {
		f := NewFilter(
			createTestFile("Admin"),
			"",
			"",
		)
		require.True(t, f.ContainsBadWord("ADMIN"))
	})

	t.Run("特殊字符过滤", func(t *testing.T) {
		f := NewFilter(
			createTestFile("admin"),
			"",
			"",
		)
		require.True(t, f.ContainsBadWord("a-d_m+i[n]"))
	})
}

func TestFilter_CharacterReplacement(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFile := func(content string, name string) string {
		path := filepath.Join(tmpDir, "test"+"_"+name+".txt")
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
		return path
	}

	// 测试数据
	similarFile := "1=i\nv=u"   // 单字符替换规则
	replaceFile := "vv=w\nph=f" // 多字符替换规则（使用=分隔符）

	t.Run("单字符替换规则", func(t *testing.T) {
		f := NewFilter(
			"", // 无敏感词
			createTestFile(similarFile, "similar_file"),
			createTestFile(replaceFile, "replace_file"),
		).(*filter)

		// 验证相似字符映射
		require.Equal(t, 'i', f.similarMap['1'])
		require.Equal(t, 'u', f.similarMap['v'])

		// 测试替换逻辑
		require.Equal(t, "iu", f.normalizeSingleChars("1v"))
	})

	t.Run("多字符替换顺序", func(t *testing.T) {
		// 敏感词文件包含"wolf"
		f := NewFilter(
			createTestFile("wolf", "sensitive_words"),
			createTestFile(similarFile, "similar_file"),
			createTestFile(replaceFile, "replace_file"),
		)

		// 输入经过：vvolph → 替换vv→w → wolph → 替换ph→f → wolf
		require.True(t, f.ContainsBadWord("vvolph"))
	})
}

func TestSimilarChars(t *testing.T) {
	// 初始化测试环境
	tmpDir := t.TempDir()

	// 创建测试文件
	createTestFile := func(content string) string {
		path := filepath.Join(tmpDir, "testfile.txt")
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
		return path
	}

	similarCharsContent := `
  # 相似字符映射
  0=o
  1=i
  3=e
  4=a
  5=s
  7=t
  8=b
  `

	t.Run("加载相似字符映射", func(t *testing.T) {
		f := NewFilter("", createTestFile(similarCharsContent), "").(*filter)
		require.Equal(t, 'o', f.similarMap['0'])
		require.Equal(t, 'i', f.similarMap['1'])
		require.Equal(t, 'e', f.similarMap['3'])
		require.Equal(t, 'a', f.similarMap['4'])
	})

	t.Run("单字符替换应用", func(t *testing.T) {
		f := NewFilter("", createTestFile(similarCharsContent), "").(*filter)
		require.Equal(t, "password", f.normalizeSingleChars("p4ssw0rd"))
		require.Equal(t, "test", f.normalizeSingleChars("7es7"))
		require.Equal(t, "admin", f.normalizeSingleChars("4dm1n"))
	})

	t.Run("空映射处理", func(t *testing.T) {
		f := NewFilter("", "", "").(*filter)
		require.Equal(t, "test", f.normalizeSingleChars("test"))
	})
}

func TestFilter_PreprocessFlow(t *testing.T) {
	// 初始化测试环境
	tmpDir := t.TempDir()

	// 创建测试文件
	createTestFile := func(content string) string {
		path := filepath.Join(tmpDir, "testfile.txt")
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
		return path
	}

	// 创建必要的规则文件
	similarFile := "1=i\nv=u"
	replaceFile := "vv=w\nPh=f"

	testCases := []struct {
		name   string
		input  string
		expect string
	}{
		{
			"完整处理流程",
			"vv23!",
			"w23", // Vv→w, 1→i保留，过滤特殊字符!
		},
		{
			"多阶段处理",
			"Ph0ne",
			"f0ne", // Ph→f (需要多字符替换规则)
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 使用规则文件创建过滤器
			f := NewFilter(
				"",
				createTestFile(similarFile),
				createTestFile(replaceFile),
			).(*filter)
			require.Equal(t, tc.expect, f.preprocessText(tc.input))
		})
	}
}

func TestFilter_FileLoading(t *testing.T) {
	// 初始化测试环境
	tmpDir := t.TempDir()

	// 创建测试文件
	createTestFile := func(content string) string {
		path := filepath.Join(tmpDir, "testfile.txt")
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
		return path
	}

	t.Run("敏感词文件加载", func(t *testing.T) {
		f := NewFilter(
			createTestFile("# 注释\nadmin\n\npassword"),
			"",
			"",
		)
		require.True(t, f.ContainsBadWord("admin"))
		require.True(t, f.ContainsBadWord("password"))
	})

	t.Run("空文件处理", func(t *testing.T) {
		f := NewFilter(
			createTestFile(""),
			createTestFile(""),
			createTestFile(""),
		)
		require.False(t, f.ContainsBadWord("any"))
	})
}

func TestFilter_Concurrency(t *testing.T) {
	// 初始化测试环境
	tmpDir := t.TempDir()

	// 创建测试文件
	createTestFile := func(content string) string {
		path := filepath.Join(tmpDir, "testfile.txt")
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
		return path
	}

	f := NewFilter("", "", "")
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			err := f.(*filter).loadSensitiveWords(createTestFile("concurrent"))
			if err != nil {
				panic(err)
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			f.ContainsBadWord("concurrent")
		}
	}()

	wg.Wait()
	require.True(t, f.ContainsBadWord("concurrent"))
}

func TestFilter_EdgeCases(t *testing.T) {
	// 初始化测试环境
	tmpDir := t.TempDir()

	// 创建测试文件
	createTestFile := func(content string) string {
		path := filepath.Join(tmpDir, "testfile.txt")
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
		return path
	}

	f := NewFilter(createTestFile("short"), "", "")

	t.Run("超短输入", func(t *testing.T) {
		require.False(t, f.ContainsBadWord("a"))
	})

	t.Run("空字符串", func(t *testing.T) {
		require.False(t, f.ContainsBadWord(""))
	})

	t.Run("全特殊字符", func(t *testing.T) {
		require.False(t, f.ContainsBadWord("!@#$%"))
	})
}
