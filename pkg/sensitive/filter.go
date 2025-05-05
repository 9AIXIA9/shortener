// Package sensitive 快速检测敏感词（只支持大小写字母及数字）
//
//go:generate mockgen -source=$GOFILE -destination=./mock/filter_mock.go -package=sensit
//go:generate mockgen -source=$GOFILE -destination=./mock/filter_mock.go -package=sensitive
package sensitive

import (
	"bufio"
	"github.com/zeromicro/go-zero/core/logx"
	"os"
	"sort"
	"strings"
	"sync"
)

type Filter interface {
	ContainsBadWord(input string) bool
}

const (
	invalidChar = -1
	charSetSize = 36 // 0-9 + a-z
	ascllLen    = 256
)

type trieNode struct {
	children *[charSetSize]*trieNode // 0-9: 数字(0-9), 10-35: 字母(a-z)
	mask     uint64                  //用于校验子树是否存在的掩码 1 存在 0 不存在
	isEnd    bool
}

type filter struct {
	root         *trieNode
	mu           sync.RWMutex
	charIndex    [ascllLen]int // 字符快速索引
	similarMap   map[rune]rune // 单字符替换映射
	replaceRules []replaceRule // 多字符替换规则
	nodePool     sync.Pool     // 节点对象池
	bufferPool   sync.Pool     // 字符串处理缓冲池
}

type replaceRule struct {
	from string
	to   string
}

func NewFilter(sensitivePath, similarPath, replacePath string) Filter {
	f := &filter{
		root: &trieNode{
			children: &[charSetSize]*trieNode{},
			mask:     0, // 初始化 mask
			isEnd:    false,
		},
		similarMap: make(map[rune]rune),
		nodePool: sync.Pool{
			New: func() interface{} {
				return &trieNode{
					children: &[charSetSize]*trieNode{},
					isEnd:    false,
					mask:     0,
				}
			},
		},
		bufferPool: sync.Pool{
			New: func() interface{} {
				return new(strings.Builder)
			},
		},
	}

	// 初始化字符索引为无效值
	for i := 0; i < 256; i++ {
		f.charIndex[i] = invalidChar
	}

	// 设置有效字符索引
	for c := '0'; c <= '9'; c++ {
		f.charIndex[c] = int(c - '0')
	}
	for c := 'a'; c <= 'z'; c++ {
		f.charIndex[c] = 10 + int(c-'a')
	}

	// 加载配置文件
	err := f.loadSimilarMap(similarPath)
	if err != nil && similarPath != "" {
		logx.Severef("Failed to load similar chars: %v", err)
	}

	err = f.loadReplaceRules(replacePath)
	if err != nil && replacePath != "" {
		logx.Severef("Failed to load replace rules: %v", err)
	}

	err = f.loadSensitiveWords(sensitivePath)
	if err != nil && sensitivePath != "" {
		logx.Severef("Failed to load sensitive words: %v", err)
	}

	// 预处理多字符替换规则
	f.preprocessReplaceRules()

	return f
}

// 预处理替换规则，应用单字符规范化
func (f *filter) preprocessReplaceRules() {
	processedRules := make([]replaceRule, 0, len(f.replaceRules))

	// 1. 先进行单字符规范化
	for _, rule := range f.replaceRules {
		from := f.normalizeSingleChars(rule.from)
		to := f.normalizeSingleChars(rule.to)
		processedRules = append(processedRules, replaceRule{
			from: from,
			to:   to,
		})
	}

	// 2. 按规范化后的from长度降序排序
	sort.Slice(processedRules, func(i, j int) bool {
		return len(processedRules[i].from) > len(processedRules[j].from)
	})

	f.replaceRules = processedRules
}

func (f *filter) ContainsBadWord(input string) bool {
	if len(input) < 2 {
		return false
	}

	processed := f.preprocessText(input)
	if len(processed) == 0 {
		return false
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	// 高效 Trie 匹配
	for i := 0; i < len(processed); i++ {
		current := f.root
		for j := i; j < len(processed); j++ {
			c := processed[j]
			idx := f.charIndex[c]
			if idx == invalidChar {
				break
			}

			// 先检查 mask 位，避免访问 children 数组
			if (current.mask & (1 << idx)) == 0 {
				break // 子节点不存在
			}

			current = current.children[idx]
			if current.isEnd {
				return true
			}
		}
	}
	return false
}

// 高效处理单个形近字
func (f *filter) normalizeSingleChars(s string) string {
	if len(s) == 0 {
		return s
	}

	// 从对象池获取 builder
	builder := f.bufferPool.Get().(*strings.Builder)
	builder.Reset()
	defer f.bufferPool.Put(builder)

	for _, c := range s {
		if similar, ok := f.similarMap[c]; ok {
			builder.WriteRune(similar)
		} else {
			builder.WriteRune(c)
		}
	}
	return builder.String()
}

// 处理多形近字替换和小写转换
func (f *filter) normalizeText(s string) string {
	result := s
	for _, rule := range f.replaceRules {
		result = strings.ReplaceAll(result, rule.from, rule.to)
	}
	return strings.ToLower(result) // 确保最后统一转小写
}

// 整合所有字符处理过程
func (f *filter) preprocessText(s string) string {
	// 1. 先过滤非法字符
	cleaned := f.filterInvalidChars(s)

	// 2. 单字符替换
	normalized := f.normalizeSingleChars(cleaned)

	// 3. 多字符替换并转小写
	return f.normalizeText(normalized)
}

func (f *filter) filterInvalidChars(s string) string {
	builder := f.bufferPool.Get().(*strings.Builder)
	defer f.bufferPool.Put(builder)
	builder.Reset()

	for _, c := range s {
		if ('0' <= c && c <= '9') || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') {
			builder.WriteRune(c)
		}
	}
	return builder.String()
}

// 文件加载逻辑
func (f *filter) loadSimilarMap(path string) error {
	if path == "" {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.ReplaceAll(line, " ", "") // 移除所有空格
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, "=")
		if len(parts) != 2 {
			continue
		}

		// 统一转换为小写处理
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1]) // 直接获取等号右侧值

		if len(key) != 1 || len(value) != 1 {
			continue // 跳过无效行
		}

		// 映射关系：原始字符 → 目标字符
		f.similarMap[rune(key[0])] = rune(value[0])
	}
	return scanner.Err()
}

func (f *filter) loadReplaceRules(path string) error {
	if path == "" {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var rules []replaceRule
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.ReplaceAll(line, " ", "") // 移除空格
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, "=")
		if len(parts) == 2 {
			rules = append(rules, replaceRule{
				from: strings.TrimSpace(parts[0]),
				to:   strings.TrimSpace(parts[1]),
			})
		}
	}

	f.replaceRules = rules

	return scanner.Err()
}

func (f *filter) loadSensitiveWords(path string) error {
	if path == "" {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var words []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" && !strings.HasPrefix(line, "#") {
			words = append(words, line)
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	for _, word := range words {
		processed := f.preprocessText(word)
		if len(processed) == 0 {
			continue
		}

		current := f.root
		for _, c := range processed {
			idx := f.charIndex[c]
			if idx == invalidChar {
				continue
			}

			// 使用 mask 位判断子节点是否存在
			if (current.mask & (1 << idx)) == 0 {
				// 子节点不存在，获取或创建新节点
				node := f.nodePool.Get().(*trieNode)
				*node = trieNode{
					children: &[charSetSize]*trieNode{},
					mask:     0,
					isEnd:    false,
				}
				current.children[idx] = node
				// 设置 mask 位标记子节点存在
				current.mask |= 1 << idx
			}
			current = current.children[idx]
		}
		current.isEnd = true
	}
	return scanner.Err()
}
