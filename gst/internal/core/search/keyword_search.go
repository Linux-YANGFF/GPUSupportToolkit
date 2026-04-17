package search

import (
	"regexp"
	"strings"

	"gst/internal/core"
)

// KeywordSearch 关键字检索
type KeywordSearch struct {
	results []core.SearchResult
}

func NewKeywordSearch() *KeywordSearch {
	return &KeywordSearch{}
}

// Search 执行关键字搜索
// keywords: 关键字列表，AND 关系（所有关键字都匹配才返回）
// lines: 日志行内容
// 返回: 匹配的结果
func (ks *KeywordSearch) Search(keywords []string, lines []string) []core.SearchResult {
	if len(keywords) == 0 || len(lines) == 0 {
		return nil
	}

	// Compile regex patterns (case-insensitive)
	patterns := make([]*regexp.Regexp, 0, len(keywords))
	for _, kw := range keywords {
		if kw == "" {
			continue
		}
		re, err := regexp.Compile(`(?i)` + regexp.QuoteMeta(kw))
		if err != nil {
			continue
		}
		patterns = append(patterns, re)
	}

	if len(patterns) == 0 {
		return nil
	}

	var results []core.SearchResult
	for i, line := range lines {
		if ks.matchAll(line, patterns) {
			results = append(results, core.SearchResult{
				LineNum: i + 1, // Line numbers start at 1
				Content: line,
				PageNum: 0, // Pagination calculated later
			})
		}
	}

	// Calculate pagination
	const pageSize = 1000
	for i := range results {
		results[i].PageNum = i / pageSize
	}

	return results
}

func (ks *KeywordSearch) matchAll(line string, patterns []*regexp.Regexp) bool {
	for _, p := range patterns {
		if !p.MatchString(line) {
			return false
		}
	}
	return true
}

// SearchWithPagination 带分页的搜索
func (ks *KeywordSearch) SearchWithPagination(keywords []string, lines []string, page, pageSize int) ([]core.SearchResult, int) {
	allResults := ks.Search(keywords, lines)
	total := len(allResults)

	if page < 0 {
		page = 0
	}
	if pageSize <= 0 {
		pageSize = 1000
	}

	start := page * pageSize
	end := start + pageSize

	if start >= total {
		return nil, total
	}
	if end > total {
		end = total
	}

	return allResults[start:end], total
}

// KeywordSearchSimple Simple keyword search without regex (case-insensitive substring match)
func KeywordSearchSimple(keywords []string, lines []string) []core.SearchResult {
	if len(keywords) == 0 || len(lines) == 0 {
		return nil
	}

	// Convert keywords to lowercase, filtering out empty strings
	lowerKeywords := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		if kw == "" {
			continue
		}
		lowerKeywords = append(lowerKeywords, strings.ToLower(kw))
	}

	if len(lowerKeywords) == 0 {
		return nil
	}

	var results []core.SearchResult
	for i, line := range lines {
		lowerLine := strings.ToLower(line)
		allMatch := true
		for _, kw := range lowerKeywords {
			if !strings.Contains(lowerLine, kw) {
				allMatch = false
				break
			}
		}
		if allMatch {
			results = append(results, core.SearchResult{
				LineNum: i + 1,
				Content: line,
				PageNum: len(results) / 1000,
			})
		}
	}

	return results
}