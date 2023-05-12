package tokenizer

import (
	"bufio"
	"bytes"
	"fmt"

	"github.com/blevesearch/segment"
	"github.com/rylans/getlang"
)

func CalcTokens(text string) int {
	scanner := bufio.NewScanner(bytes.NewReader([]byte(text)))
	scanner.Split(segment.SplitWords)
	words := []byte{}
	for scanner.Scan() {
		tokenBytes := scanner.Bytes()
		words = append(words, tokenBytes...)
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("计算错误: %v", err)
		return 0
	}
	return len(words)
}

func WhatLang(s string) string {
	info := getlang.FromString(s)
	return info.LanguageCode()
}
