// Package sensitive 快速检测敏感词（只支持大小写字母及数字）
//
//go:generate mockgen -source=$GOFILE -destination=./mock/filter_mock.go -package=sensitive
package sensitive

import (
	"bufio"
	"github.com/zeromicro/go-zero/core/logx"
	"os"
	"shortener/pkg/errorx"
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
	ascll       = 256
)

type trieNode struct {
	children *[charSetSize]*trieNode // 0-9: 数字(0-9), 10-35: 字母(a-z)
	mask     uint64                  //用于校验子树是否存在的掩码 1 存在 0 不存在
	isEnd    bool
}

type filter struct {
	root         *trieNode
	mu           sync.RWMutex
	charIndex    [ascll]int    // 字符快速索引
	similarMap   map[rune]rune // 单字符替换映射(区分大小写，因为大小写字母形态发生很大变化)
	replaceRules []replaceRule // 多字符替换规则(区分大小写，因为大小写字母形态发生很大变化)
	nodePool     sync.Pool     // 节点对象池
	bufferPool   sync.Pool     // 字符串处理缓冲池
}

type replaceRule struct {
	from string
	to   string
}

func NewFilter(sensitivePath, similarPath, replacePath string) (Filter, error) {
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
	for c := 'A'; c <= 'Z'; c++ {
		f.charIndex[c] = 10 + int(c-'A') // 与a-z共享索引
	}

	// 加载配置文件
	var lastErr error
	if similarPath != "" {
		if err := f.loadSimilarMap(similarPath); err != nil {
			lastErr = errorx.Wrap(err, errorx.CodeSystemError, "failed to load similar map")
			// 继续加载其他配置
		}
	}

	if replacePath != "" {
		if err := f.loadReplaceRules(replacePath); err != nil {
			lastErr = errorx.Wrap(err, errorx.CodeSystemError, "failed to load replace rules")
			// 继续加载其他配置
		}
	}

	if sensitivePath != "" {
		if err := f.loadSensitiveWords(sensitivePath); err != nil {
			lastErr = errorx.Wrap(err, errorx.CodeSystemError, "failed to load sensitive words")
		}
	}

	// 预处理多字符替换规则
	f.preprocessReplaceRules()

	return f, lastErr
}

// 优化节点重置方式
func (n *trieNode) reset() {
	// 只重置必要字段，减少内存操作
	n.isEnd = false
	n.mask = 0
	// 子节点会在使用时重新分配，无需遍历清空
}

func (f *filter) ContainsBadWord(input string) bool {
	// 优化边界条件检查
	if input == "" || len(input) < 2 {
		return false
	}

	processed := f.preprocessText(input)
	if len(processed) == 0 {
		return false
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	// 使用位掩码更快速地遍历Trie
	for i := 0; i < len(processed); i++ {
		current := f.root
		for j := i; j < len(processed); j++ {
			c := processed[j]
			// 检查字符是否在有效范围内
			if int(c) >= ascll {
				break
			}

			idx := f.charIndex[c]
			if idx == invalidChar {
				break
			}

			// 使用一致的方式计算mask
			mask := uint64(1) << uint(idx)
			if (current.mask & mask) == 0 {
				break
			}

			current = current.children[idx]
			if current.isEnd {
				return true // 立即返回，无需继续检查
			}
		}
	}
	return false
}

// 预处理替换规则，应用单字符规范化
func (f *filter) preprocessReplaceRules() {
	processedRules := make([]replaceRule, 0, len(f.replaceRules))

	// 1. 先进行单字符规范化
	for _, rule := range f.replaceRules {
		from := f.normalizeSingleChars(rule.from)
		to := f.normalizeSingleChars(rule.to)

		// 过滤空规则和自映射规则
		if from == "" || from == to {
			continue
		}

		processedRules = append(processedRules, replaceRule{
			from: from,
			to:   to,
		})
	}

	// 2. 按规范化后的from长度降序排序
	sort.Slice(processedRules, func(i, j int) bool {
		if len(processedRules[i].from) == len(processedRules[j].from) {
			return processedRules[i].from > processedRules[j].from // 相同长度时按字典序倒排
		}
		return len(processedRules[i].from) > len(processedRules[j].from)
	})
	f.replaceRules = processedRules
}

// 整合所有字符处理过程
// 说明：
// 由于多字符替换中涉及字母大小写的区别
// 所以并不能提前进行小写转换
// 如L和l的形态差异较大，在我看来并不算形近字
// 所以虽然cl可以被识别成d，但是cL不需要被识别成d
func (f *filter) preprocessText(s string) string {
	if len(s) == 0 {
		return s
	}

	builder := f.bufferPool.Get().(*strings.Builder)
	builder.Reset()
	defer f.bufferPool.Put(builder)

	// 预分配一定容量减少内存重分配
	builder.Grow(len(s))

	// 单次循环完成字符过滤和单字符替换
	for _, c := range s {
		// 只保留有效字符
		if ('0' <= c && c <= '9') || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') {
			// 处理可能的字符越界
			if int(c) < ascll {
				// 应用单字符替换
				if similar, ok := f.similarMap[c]; ok {
					builder.WriteRune(similar)
				} else {
					builder.WriteRune(c)
				}
			}
		}
	}

	result := builder.String()

	// 应用多字符替换并转小写
	for _, rule := range f.replaceRules {
		result = strings.ReplaceAll(result, rule.from, rule.to)
	}
	return strings.ToLower(result)
}

// 高效处理单个形近字
func (f *filter) normalizeSingleChars(s string) string {
	if len(s) == 0 || len(f.similarMap) == 0 {
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

// 文件加载逻辑
func (f *filter) loadSimilarMap(path string) error {
	if path == "" {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return errorx.NewWithCause(errorx.CodeSystemError, "open the similar map file failed", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logx.Errorf("failed to close similar map file %s: %v", path, err)
		}
	}()

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

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1]) // 直接获取等号右侧值

		if len(key) != 1 || len(value) != 1 {
			continue // 跳过无效行
		}

		// 映射关系：原始字符 → 目标字符
		// 说明：不能直接盲目转大小写
		//如：i和1和I和l可以视为同组
		//但是L不能差异过大不应该纳入此组
		f.similarMap[rune(key[0])] = rune(value[0])
	}

	if err := scanner.Err(); err != nil {
		return errorx.NewWithCause(errorx.CodeSystemError, "reading similar map file failed", err)
	}
	return nil
}

func (f *filter) loadReplaceRules(path string) error {
	if path == "" {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return errorx.NewWithCause(errorx.CodeSystemError, "open the replace rule file failed", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logx.Errorf("failed to close replace rules file %s: %v", path, err)
		}
	}()

	var rules []replaceRule
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.ReplaceAll(line, " ", "") // 移除空格
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 移除所有空格并确保格式一致性
		line = strings.ReplaceAll(line, " ", "")
		parts := strings.Split(line, "=")
		if len(parts) == 2 {
			from := parts[0]
			to := parts[1]
			if from != "" { // 确保from不为空
				rules = append(rules, replaceRule{
					from: from,
					to:   to,
				})
			}
		}
	}

	f.replaceRules = rules

	if err := scanner.Err(); err != nil {
		return errorx.NewWithCause(errorx.CodeSystemError, "reading replace rules file failed", err)
	}
	return nil
}

func (f *filter) loadSensitiveWords(path string) error {
	if path == "" {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return errorx.NewWithCause(errorx.CodeSystemError, "open the sensitive words file failed", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logx.Errorf("failed to close sensitive words file %s: %v", path, err)
		}
	}()

	var words []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" && !strings.HasPrefix(line, "#") {
			words = append(words, line)
		}
	}

	// 先检查扫描是否有错误
	if err := scanner.Err(); err != nil {
		return errorx.NewWithCause(errorx.CodeSystemError, "reading sensitive words file failed", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	for _, word := range words {
		processed := f.preprocessText(word)
		if len(processed) < 2 {
			continue
		}

		current := f.root
		for _, c := range processed {
			// 确保字符索引不越界
			if int(c) >= ascll {
				continue
			}

			idx := f.charIndex[c]
			if idx == invalidChar {
				continue
			}

			// 使用 mask 位判断子节点是否存在，保持类型一致
			mask := uint64(1) << uint(idx)
			if (current.mask & mask) == 0 {
				// 子节点不存在，获取或创建新节点
				node := f.nodePool.Get().(*trieNode)
				// 使用 reset 方法重置节点状态而不是重新创建
				node.reset()
				current.children[idx] = node
				// 设置 mask 位标记子节点存在，使用同样的方式计算 mask
				current.mask |= mask
			}
			current = current.children[idx]
		}
		current.isEnd = true
	}

	return nil
}
