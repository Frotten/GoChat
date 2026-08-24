package evaluation

import (
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"GopherAI/common/aihelper"
	"GopherAI/common/rag"
)

// Score 根据评分策略对实际输出打分，返回 score(0~1) 与是否通过。
func Score(scoreType ScoreType, actual, expected string, keywords []string) (float64, bool) {
	actual = strings.TrimSpace(actual)
	expected = strings.TrimSpace(expected)

	switch scoreType {
	case ScoreTypeExact:
		if actual == "" {
			return 0, false
		}
		passed := strings.EqualFold(actual, expected)
		if passed {
			return 1, true
		}
		return 0, false
	case ScoreTypeContains:
		if actual == "" || expected == "" {
			return 0, false
		}
		passed := strings.Contains(strings.ToLower(actual), strings.ToLower(expected))
		if passed {
			return 1, true
		}
		return 0, false
	case ScoreTypeKeywords:
		if actual == "" || len(keywords) == 0 {
			return 0, false
		}
		lower := strings.ToLower(actual)
		for _, kw := range keywords {
			kw = strings.TrimSpace(kw)
			if kw == "" {
				continue
			}
			if !strings.Contains(lower, strings.ToLower(kw)) {
				return 0, false
			}
		}
		return 1, true
	default:
		// none：只要 Agent 正常返回非空内容即视为通过（冒烟测试）
		if actual == "" {
			return 0, false
		}
		return 1, true
	}
}

func ScoreToolCalls(expected []ToolCallExpectation, forbidden []string, actual []aihelper.ToolInvocation) ToolMetrics {
	m := ToolMetrics{ExpectedCalls: len(expected), ActualCalls: len(actual)}
	if len(expected) == 0 {
		m.ExactMatch = len(actual) == 0
		if m.ExactMatch {
			m.NamePrecision, m.NameRecall = 1, 1
			m.CallPrecision, m.CallRecall, m.CallF1 = 1, 1, 1
			m.ArgumentAccuracy, m.OrderAccuracy = 1, 1
		}
		return m
	}

	forbiddenSet := stringSet(forbidden)
	for _, call := range actual {
		if forbiddenSet[strings.ToLower(strings.TrimSpace(call.Name))] {
			m.ForbiddenCalls++
		}
	}

	usedForName := make([]bool, len(actual))
	usedForCall := make([]bool, len(actual))
	for _, want := range expected {
		for i, got := range actual {
			if !usedForName[i] && strings.EqualFold(strings.TrimSpace(want.Name), strings.TrimSpace(got.Name)) {
				usedForName[i] = true
				m.MatchedNames++
				break
			}
		}
		for i, got := range actual {
			if usedForCall[i] || !strings.EqualFold(strings.TrimSpace(want.Name), strings.TrimSpace(got.Name)) {
				continue
			}
			if argumentsMatch(want.Arguments, got.Arguments) {
				usedForCall[i] = true
				m.MatchedCalls++
				break
			}
		}
	}

	m.NamePrecision = ratio(m.MatchedNames, len(actual))
	m.NameRecall = ratio(m.MatchedNames, len(expected))
	m.CallPrecision = ratio(m.MatchedCalls, len(actual))
	m.CallRecall = ratio(m.MatchedCalls, len(expected))
	m.CallF1 = harmonicMean(m.CallPrecision, m.CallRecall)
	m.ArgumentAccuracy = ratio(m.MatchedCalls, len(expected))
	m.OrderAccuracy = toolOrderAccuracy(expected, actual)
	m.ExactMatch = m.MatchedCalls == len(expected) && len(actual) == len(expected) && m.ForbiddenCalls == 0 && m.OrderAccuracy == 1
	return m
}

func ScoreRAG(expectedSources []string, hits []rag.RetrievalHit, k int) RAGMetrics {
	if k <= 0 {
		k = len(hits)
	}
	m := RAGMetrics{K: k}
	wanted := stringSet(expectedSources)
	if len(wanted) == 0 {
		return m
	}
	seenRelevantSources := make(map[string]bool)
	relevantHits := 0
	limit := k
	if limit > len(hits) {
		limit = len(hits)
	}
	for i := 0; i < limit; i++ {
		source := normalizeSource(hits[i].Source)
		if sourceMatches(wanted, source) {
			relevantHits++
			seenRelevantSources[source] = true
			if !m.Hit {
				m.Hit = true
				m.MRR = 1 / float64(i+1)
			}
		}
	}
	// Count each expected document once even when multiple chunks are retrieved.
	matchedExpected := 0
	for want := range wanted {
		for got := range seenRelevantSources {
			if sourceEqual(want, got) {
				matchedExpected++
				break
			}
		}
	}
	m.RecallAtK = ratio(matchedExpected, len(wanted))
	m.PrecisionAtK = ratio(relevantHits, k)
	return m
}

func argumentsMatch(expected map[string]any, rawActual string) bool {
	if len(expected) == 0 {
		return true
	}
	decoder := json.NewDecoder(strings.NewReader(rawActual))
	decoder.UseNumber()
	var actual map[string]any
	if err := decoder.Decode(&actual); err != nil {
		return false
	}
	return valueSubset(expected, actual)
}

func valueSubset(expected, actual any) bool {
	switch want := expected.(type) {
	case map[string]any:
		got, ok := actual.(map[string]any)
		if !ok {
			return false
		}
		for key, value := range want {
			actualValue, exists := got[key]
			if !exists || !valueSubset(value, actualValue) {
				return false
			}
		}
		return true
	case []any:
		got, ok := actual.([]any)
		if !ok || len(want) != len(got) {
			return false
		}
		for i := range want {
			if !valueSubset(want[i], got[i]) {
				return false
			}
		}
		return true
	case json.Number:
		return numericEqual(want.String(), actual)
	case float64:
		return numericEqual(want, actual)
	case float32:
		return numericEqual(float64(want), actual)
	case int:
		return numericEqual(float64(want), actual)
	default:
		return reflect.DeepEqual(want, actual)
	}
}

func numericEqual(expected any, actual any) bool {
	want, ok := toFloat(expected)
	if !ok {
		return false
	}
	got, ok := toFloat(actual)
	return ok && math.Abs(want-got) < 1e-9
}

func toFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case json.Number:
		n, err := v.Float64()
		return n, err == nil
	case string:
		n, err := json.Number(v).Float64()
		return n, err == nil
	default:
		return 0, false
	}
}

func toolOrderAccuracy(expected []ToolCallExpectation, actual []aihelper.ToolInvocation) float64 {
	if len(expected) == 0 {
		if len(actual) == 0 {
			return 1
		}
		return 0
	}
	want := make([]string, len(expected))
	got := make([]string, len(actual))
	for i := range expected {
		want[i] = strings.ToLower(strings.TrimSpace(expected[i].Name))
	}
	for i := range actual {
		got[i] = strings.ToLower(strings.TrimSpace(actual[i].Name))
	}
	return float64(longestCommonSubsequence(want, got)) / float64(len(want))
}

func longestCommonSubsequence(a, b []string) int {
	row := make([]int, len(b)+1)
	for _, left := range a {
		previous := 0
		for j, right := range b {
			saved := row[j+1]
			if left == right {
				row[j+1] = previous + 1
			} else if row[j] > row[j+1] {
				row[j+1] = row[j]
			}
			previous = saved
		}
	}
	return row[len(b)]
}

func sourceMatches(wanted map[string]bool, actual string) bool {
	for want := range wanted {
		if sourceEqual(want, actual) {
			return true
		}
	}
	return false
}

func sourceEqual(a, b string) bool {
	a, b = normalizeSource(a), normalizeSource(b)
	return a == b || filepath.Base(a) == filepath.Base(b)
}

func normalizeSource(source string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(source), "\\", "/"))
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		value = normalizeSource(value)
		if value != "" {
			out[value] = true
		}
	}
	return out
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func harmonicMean(a, b float64) float64 {
	if a+b == 0 {
		return 0
	}
	return 2 * a * b / (a + b)
}

// StableMetricKeys is useful to consumers that render metrics dynamically.
func StableMetricKeys(metrics map[string]float64) []string {
	keys := make([]string, 0, len(metrics))
	for key := range metrics {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
