//go:generate mockgen -source=$GOFILE -destination=./mock/filter_mock.go -package=sensitive
package sensitive

import (
	"bufio"
	"github.com/zeromicro/go-zero/core/logx"
	"os"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
)

type Filter interface {
	ContainsBadWord(input string) bool
}

type TrieNode struct {
	children map[rune]*TrieNode
	isEnd    bool
}

type filterBasedOnTrie struct {
	root *TrieNode
	// 形近字映射表
	similarChars map[rune][]rune
	// 替换规则结构（按长度降序排序）
	replaceRules []struct {
		from string
		to   string
	}
	mu       sync.RWMutex
	triePool sync.Pool
}

func NewFilter(sensitiveWordsPath, similarCharsPath, replaceRulesPath string) Filter {
	f := &filterBasedOnTrie{
		root: &TrieNode{children: make(map[rune]*TrieNode)},
		triePool: sync.Pool{
			New: func() interface{} {
				return &TrieNode{children: make(map[rune]*TrieNode)}
			},
		},
		similarChars: make(map[rune][]rune),
		replaceRules: []struct {
			from string
			to   string
		}{},
	}

	// 从文件加载敏感词
	if err := f.loadSensitiveWordsFromFile(sensitiveWordsPath); err != nil {
		logx.Severef("failed to load sensitive words: %v", err)
	}

	// 从文件加载形近字映射
	if err := f.loadSimilarChars(similarCharsPath); err != nil {
		logx.Severef("failed to load the geomorphic mapping: %v", err)
	}

	// 从文件加载替换规则
	if err := f.loadReplaceRules(replaceRulesPath); err != nil {
		logx.Severef("failed to load the substitution rule: %v", err)
	}

	// 预排序替换规则
	sort.Slice(f.replaceRules, func(i, j int) bool {
		return len(f.replaceRules[i].from) > len(f.replaceRules[j].from)
	})

	return f
}

func (f *filterBasedOnTrie) ContainsBadWord(input string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	processed := f.unifyWords(input)
	current := f.root

	for i := 0; i < len(processed); {
		r, size := utf8.DecodeRuneInString(processed[i:])
		found := false

		for testR := range current.children {
			if r == testR || f.isSimilar(r, testR) {
				current = current.children[testR]
				if current.isEnd {
					return true
				}
				found = true
				break
			}
		}

		if !found {
			current = f.root
		}
		i += size
	}
	return false
}

// loadSimilarChars 从文件加载形近字映射
func (f *filterBasedOnTrie) loadSimilarChars(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logx.Errorf("load similar chars close file failed,err:%v", err)
		}
	}()

	// 创建新的映射表
	newSimilarChars := make(map[rune][]rune)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, "=")
		if len(parts) != 2 {
			continue
		}

		if len([]rune(parts[0])) != 1 {
			continue
		}

		char := []rune(parts[0])[0]
		var similarList []rune
		for _, s := range strings.Split(parts[1], ",") {
			s = strings.TrimSpace(s)
			if len([]rune(s)) == 1 {
				similarList = append(similarList, []rune(s)[0])
			}
		}

		if len(similarList) > 0 {
			newSimilarChars[char] = similarList
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// 只有成功读取文件后才替换全局变量
	if len(newSimilarChars) > 0 {
		f.similarChars = newSimilarChars
	}

	return nil
}

// loadReplaceRules 从文件加载替换规则
func (f *filterBasedOnTrie) loadReplaceRules(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logx.Errorf("load replace rules close file failed,err:%v", err)
		}
	}()

	var newReplaceRules []struct {
		from string
		to   string
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) != 2 {
			continue
		}

		from := strings.TrimSpace(parts[0])
		to := strings.TrimSpace(parts[1])
		if from != "" && to != "" {
			newReplaceRules = append(newReplaceRules, struct {
				from string
				to   string
			}{
				from: from,
				to:   to,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// 只有成功读取文件后才替换全局变量
	if len(newReplaceRules) > 0 {
		f.replaceRules = newReplaceRules
	}

	return nil
}

func (f *filterBasedOnTrie) loadSensitiveWordsFromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logx.Errorf("load sensitivie words close file failed,err:%v", err)
		}
	}()

	scanner := bufio.NewScanner(file)
	var words []string
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" && !strings.HasPrefix(line, "#") {
			words = append(words, line)
		}
	}

	f.addBadWords(words...)
	return scanner.Err()
}

func (f *filterBasedOnTrie) addBadWords(words ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, word := range words {
		processed := f.unifyWords(word)
		if processed != "" {
			f.insertWithVariants(processed)
		}
	}
}

func (f *filterBasedOnTrie) insertWithVariants(word string) {
	runes := []rune(word)
	queue := []struct {
		node  *TrieNode
		index int
	}{{f.root, 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.index == len(runes) {
			current.node.isEnd = true
			continue
		}

		char := runes[current.index]
		similar := f.getSimilarChars(char)

		for _, c := range similar {
			if current.node.children[c] == nil {
				current.node.children[c] = f.triePool.Get().(*TrieNode)
			}
			queue = append(queue, struct {
				node  *TrieNode
				index int
			}{current.node.children[c], current.index + 1})
		}
	}
}

// unifyWords 统一化处理单词
func (f *filterBasedOnTrie) unifyWords(word string) string {
	word = strings.ToLower(word)
	for _, rule := range f.replaceRules {
		word = strings.ReplaceAll(word, rule.from, rule.to)
	}
	return word
}

func (f *filterBasedOnTrie) getSimilarChars(r rune) []rune {
	chars := []rune{r}
	if similar, ok := f.similarChars[r]; ok {
		chars = append(chars, similar...)
	}
	return chars
}

func (f *filterBasedOnTrie) isSimilar(a, b rune) bool {
	if a == b {
		return true
	}

	// 检查Trie节点的字符（b）的形近字是否包含输入字符（a）
	if similar, ok := f.similarChars[b]; ok {
		for _, c := range similar {
			if c == a {
				return true
			}
		}
	}
	return false
}
