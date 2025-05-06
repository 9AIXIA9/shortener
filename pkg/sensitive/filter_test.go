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
		f, err := NewFilter(
			createTestFile("admin\npassword"),
			"",
			"",
		)
		require.NoError(t, err)
		require.True(t, f.ContainsBadWord("user admin"))
		require.False(t, f.ContainsBadWord("normal text"))
	})

	t.Run("大小写不敏感检测", func(t *testing.T) {
		f, err := NewFilter(
			createTestFile("Admin"),
			"",
			"",
		)
		require.NoError(t, err)
		require.True(t, f.ContainsBadWord("ADMIN"))
	})

	t.Run("特殊字符过滤", func(t *testing.T) {
		f, err := NewFilter(
			createTestFile("admin"),
			"",
			"",
		)
		require.NoError(t, err)
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
		f, err := NewFilter(
			"", // 无敏感词
			createTestFile(similarFile, "similar_file"),
			createTestFile(replaceFile, "replace_file"),
		)
		require.NoError(t, err)
		fImpl := f.(*filter)

		// 验证相似字符映射
		require.Equal(t, 'i', fImpl.similarMap['1'])
		require.Equal(t, 'u', fImpl.similarMap['v'])

		// 测试替换逻辑
		require.Equal(t, "iu", fImpl.normalizeSingleChars("1v"))
	})

	t.Run("多字符替换顺序", func(t *testing.T) {
		// 敏感词文件包含"wolf"
		f, err := NewFilter(
			createTestFile("wolf", "sensitive_words"),
			createTestFile(similarFile, "similar_file"),
			createTestFile(replaceFile, "replace_file"),
		)
		require.NoError(t, err)

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
		f, err := NewFilter("", createTestFile(similarCharsContent), "")
		require.NoError(t, err)
		fImpl := f.(*filter)
		require.Equal(t, 'o', fImpl.similarMap['0'])
		require.Equal(t, 'i', fImpl.similarMap['1'])
		require.Equal(t, 'e', fImpl.similarMap['3'])
		require.Equal(t, 'a', fImpl.similarMap['4'])
	})

	t.Run("单字符替换应用", func(t *testing.T) {
		f, err := NewFilter("", createTestFile(similarCharsContent), "")
		require.NoError(t, err)
		fImpl := f.(*filter)
		require.Equal(t, "password", fImpl.normalizeSingleChars("p4ssw0rd"))
		require.Equal(t, "test", fImpl.normalizeSingleChars("7es7"))
		require.Equal(t, "admin", fImpl.normalizeSingleChars("4dm1n"))
	})

	t.Run("空映射处理", func(t *testing.T) {
		f, err := NewFilter("", "", "")
		require.NoError(t, err)
		fImpl := f.(*filter)
		require.Equal(t, "test", fImpl.normalizeSingleChars("test"))
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
			f, err := NewFilter(
				"",
				createTestFile(similarFile),
				createTestFile(replaceFile),
			)
			require.NoError(t, err)
			fImpl := f.(*filter)
			require.Equal(t, tc.expect, fImpl.preprocessText(tc.input))
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
		f, err := NewFilter(
			createTestFile("# 注释\nadmin\n\npassword"),
			"",
			"",
		)
		require.NoError(t, err)
		require.True(t, f.ContainsBadWord("admin"))
		require.True(t, f.ContainsBadWord("password"))
	})

	t.Run("空文件处理", func(t *testing.T) {
		f, err := NewFilter(
			createTestFile(""),
			createTestFile(""),
			createTestFile(""),
		)
		require.NoError(t, err)
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

	f, err := NewFilter("", "", "")
	require.NoError(t, err)
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

	f, err := NewFilter(createTestFile("short"), "", "")
	require.NoError(t, err)

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

func TestFilter_ReplaceRuleProcessing(t *testing.T) {
	// 初始化测试环境
	tmpDir := t.TempDir()

	// 创建测试文件
	createTestFile := func(content, name string) string {
		path := filepath.Join(tmpDir, name)
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
		return path
	}

	sensitiveWords := "badword"
	replaceRules := "aa=a\ncc=a"

	f, err := NewFilter(
		createTestFile(sensitiveWords, "sensitive.txt"),
		"",
		createTestFile(replaceRules, "replace.txt"),
	)
	require.NoError(t, err)

	// 测试替换规则的处理
	require.True(t, f.ContainsBadWord("baadword")) // aa->b 变成 badword
	require.True(t, f.ContainsBadWord("bccdword")) // cc->d 变成 badword

	// 测试大小写处理
	require.False(t, f.ContainsBadWord("BaAdWoRd"))
}

func TestFilter_ErrorHandling(t *testing.T) {
	// 初始化测试环境
	tmpDir := t.TempDir()

	// 创建测试文件
	createTestFile := func(content, name string) string {
		path := filepath.Join(tmpDir, name)
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
		return path
	}

	t.Run("非法文件路径处理", func(t *testing.T) {
		// 制造不存在的路径
		nonExistentPath := filepath.Join(tmpDir, "nonexistent")

		// 所有路径都不存在，应该返回错误但过滤器仍能初始化
		f, err := NewFilter(nonExistentPath, nonExistentPath, nonExistentPath)
		require.Error(t, err) // 应返回错误
		require.NotNil(t, f)  // 过滤器实例仍应有效

		// 过滤器应该能正常工作，只是没有敏感词
		require.False(t, f.ContainsBadWord("test"))
	})

	t.Run("空路径处理", func(t *testing.T) {
		// 所有路径为空，应初始化成功且无错误
		f, err := NewFilter("", "", "")
		require.NoError(t, err)
		require.NotNil(t, f)
	})

	t.Run("部分配置加载失败", func(t *testing.T) {
		// 敏感词文件存在，其他不存在
		sensitiveContent := "badword"
		validPath := createTestFile(sensitiveContent, "sensitive.txt")
		nonExistentPath := filepath.Join(tmpDir, "nonexistent")

		// 应返回错误但过滤器仍能初始化并加载有效的敏感词
		f, err := NewFilter(validPath, nonExistentPath, nonExistentPath)
		require.Error(t, err) // 应返回错误
		require.NotNil(t, f)  // 过滤器实例仍应有效

		// 测试有效配置是否加载成功
		require.True(t, f.ContainsBadWord("badword"))
	})
}

func TestFilter_EmptyInputHandling(t *testing.T) {
	tmpDir := t.TempDir()

	createTestFile := func(content, name string) string {
		path := filepath.Join(tmpDir, name)
		require.NoError(t, os.WriteFile(path, []byte(content), 0644))
		return path
	}

	sensitiveWords := "badword"
	f, err := NewFilter(
		createTestFile(sensitiveWords, "sensitive.txt"),
		"",
		"",
	)
	require.NoError(t, err)

	// 测试各种边界条件的输入
	require.False(t, f.ContainsBadWord(""))    // 空字符串
	require.False(t, f.ContainsBadWord("a"))   // 单字符
	require.False(t, f.ContainsBadWord("!@#")) // 只有特殊字符
}
