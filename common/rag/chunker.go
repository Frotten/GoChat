package rag

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var sentenceRegexp = regexp.MustCompile(`[^。！？；!?]+[。！？；!?]?`) //编译正则表达式
//正则NB：匹配不为指定符号的N个字符，并以0个或1个指定符号结尾

func splitText(text string, chunkSize, overlap int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if utf8.RuneCountInString(text) <= chunkSize {
		return []string{text}
	}
	var chunks []string
	// 按段落切
	paragraphs := strings.Split(text, "\n")
	var current []string
	currentLen := 0

	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		// 按句子切
		sentences := sentenceRegexp.FindAllString(paragraph, -1)
		if len(sentences) == 0 {
			sentences = []string{paragraph}
		}
		for _, sentence := range sentences {
			sentence = strings.TrimSpace(sentence)
			if sentence == "" {
				continue
			}
			l := utf8.RuneCountInString(sentence)
			// 一句话太长，继续按字符切
			if l > chunkSize {
				if len(current) > 0 {
					chunks = append(chunks, strings.Join(current, "\n"))
					current = nil
					currentLen = 0
				}
				runes := []rune(sentence)
				for start := 0; start < len(runes); {
					end := start + chunkSize
					if end > len(runes) {
						end = len(runes)
					}
					chunks = append(chunks, string(runes[start:end]))
					if end == len(runes) {
						break
					}
					start = end - overlap
					if start < 0 {
						start = 0
					}
				}
				continue
			}
			// 当前Chunk满了
			if currentLen+l > chunkSize {
				chunks = append(chunks, strings.Join(current, "\n"))
				if overlap > 0 && len(current) > 0 {
					var keep []string
					length := 0
					for i := len(current) - 1; i >= 0; i-- {
						length += utf8.RuneCountInString(current[i])
						keep = append([]string{current[i]}, keep...)
						if length >= overlap {
							break
						}
					}
					current = keep
					currentLen = length
				} else {
					current = nil
					currentLen = 0
				}
			}
			current = append(current, sentence)
			currentLen += l
		}
	}
	if len(current) > 0 {
		chunks = append(chunks, strings.Join(current, "\n"))
	}
	return chunks
}
