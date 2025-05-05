package sensitive

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// 创建临时测试文件
func createBenchmarkFile(b *testing.B, content string, name string) string {
	path := filepath.Join(b.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		b.Fatalf("无法创建临时文件: %v", err)
	}
	return path
}

// BenchmarkContainsBadWord_NoMatch 测试无敏感词匹配的场景
func BenchmarkContainsBadWord_NoMatch(b *testing.B) {
	words := "admin\npassword\nbad\nword"
	wordFile := createBenchmarkFile(b, words, "words.txt")

	f := NewFilter(wordFile, "", "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.ContainsBadWord("good text without any sensitive content")
	}
}

// BenchmarkContainsBadWord_Match 测试包含敏感词的场景
func BenchmarkContainsBadWord_Match(b *testing.B) {
	words := "admin\npassword\nbad\nword"
	wordFile := createBenchmarkFile(b, words, "words.txt")

	f := NewFilter(wordFile, "", "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.ContainsBadWord("this text contains admin which is bad")
	}
}

// BenchmarkContainsBadWord_SimilarChar 测试形近字替换
func BenchmarkContainsBadWord_SimilarChar(b *testing.B) {
	words := "admin\npassword\nbad\nword"
	wordFile := createBenchmarkFile(b, words, "words.txt")

	similarChars := "4=a\n1=i\n0=o"
	similarFile := createBenchmarkFile(b, similarChars, "similar.txt")

	f := NewFilter(wordFile, similarFile, "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.ContainsBadWord("this text contains 4dm1n which is b4d")
	}
}

// BenchmarkContainsBadWord_MultiChar 测试多字符替换
func BenchmarkContainsBadWord_MultiChar(b *testing.B) {
	words := "admin\npassword\nwolf\ndog"
	wordFile := createBenchmarkFile(b, words, "words.txt")

	replaceRules := "vv=w\nph=f"
	replaceFile := createBenchmarkFile(b, replaceRules, "replace.txt")

	f := NewFilter(wordFile, "", replaceFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.ContainsBadWord("this text contains vvolf")
	}
}

// BenchmarkPreprocessText 测试文本预处理性能
func BenchmarkPreprocessText(b *testing.B) {
	similarChars := "4=a\n1=i\n0=o"
	similarFile := createBenchmarkFile(b, similarChars, "similar.txt")

	replaceRules := "vv=w\nph=f"
	replaceFile := createBenchmarkFile(b, replaceRules, "replace.txt")

	f := NewFilter("", similarFile, replaceFile).(*filter)

	text := "This 1s 4 c0mpl3x t3xt vvith special ch4r4ct3rs like &*()_+|}{[]\\:;\"'<>,.?/~`!@#$%^"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.preprocessText(text)
	}
}

// BenchmarkLoadSensitiveWords 测试加载大量敏感词的性能
func BenchmarkLoadSensitiveWords(b *testing.B) {
	// 生成10000个敏感词
	var words string
	for i := 0; i < 10000; i++ {
		// 使用字符拼接方式生成不同的敏感词(例如：wordaaa, wordaab, ...)
		words += fmt.Sprintf("word%c%c%c\n", 'a'+i%26, 'a'+(i/26)%26, 'a'+(i/676)%26)
	}
	wordFile := createBenchmarkFile(b, words, "words.txt")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f := NewFilter("", "", "")
		err := f.(*filter).loadSensitiveWords(wordFile)
		if err != nil {
			panic(err)
		}
	}
}

// BenchmarkLongTextProcessing 测试处理长文本的性能
func BenchmarkLongTextProcessing(b *testing.B) {
	words := "admin\npassword\nbad\nword"
	wordFile := createBenchmarkFile(b, words, "words.txt")

	f := NewFilter(wordFile, "", "")

	// 创建一个较长的文本（重复10次）
	longText := ""
	for i := 0; i < 10; i++ {
		longText += "This is a long paragraph of text that contains no sensitive words. "
		longText += "It is designed to test the performance of the filter when processing large amounts of text. "
		longText += "The filter should be able to quickly determine that this text is safe. "
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.ContainsBadWord(longText)
	}
}

// BenchmarkWithFullConfiguration 测试完整配置
func BenchmarkWithFullConfiguration(b *testing.B) {
	// 创建所有配置文件
	words := "admin\npassword\nbad\nword\nwolf\ndog"
	wordFile := createBenchmarkFile(b, words, "words.txt")

	similarChars := "4=a\n1=i\n0=o\n5=s\n7=t"
	similarFile := createBenchmarkFile(b, similarChars, "similar.txt")

	replaceRules := "vv=w\nph=f\nrn=m\ncl=d"
	replaceFile := createBenchmarkFile(b, replaceRules, "replace.txt")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f := NewFilter(wordFile, similarFile, replaceFile)
		f.ContainsBadWord("7hi5 i5 4 7ex7 wi7h vvolf and phake w0rd5")
	}
}

// BenchmarkParallel 测试并行处理性能
func BenchmarkParallel(b *testing.B) {
	words := "admin\npassword\nbad\nword\nwolf\ndog"
	wordFile := createBenchmarkFile(b, words, "words.txt")

	similarChars := "4=a\n1=i\n0=o"
	similarFile := createBenchmarkFile(b, similarChars, "similar.txt")

	replaceRules := "vv=w\nph=f"
	replaceFile := createBenchmarkFile(b, replaceRules, "replace.txt")

	f := NewFilter(wordFile, similarFile, replaceFile)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			f.ContainsBadWord("this text contains 4dm1n and vvolf")
		}
	})
}
