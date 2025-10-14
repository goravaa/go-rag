package embed

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Chunk represents a piece of content to be embedded.
type Chunk struct {
	Content     string
	ContentHash string
	Metadata    map[string]interface{}
}

// getContentHash calculates the SHA256 hash of a string.
func getContentHash(content string) string {
	hashBytes := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hashBytes)
}

// Approximation: target maximum words per chunk
const maxWordsPerChunk = 256

// Markdown heading level to split sections (Level 2 => ##)
const headingLevelToSplit = 2

// ChunkMarkdown precisely splits Markdown content and calculates a hash for each chunk.
func ChunkMarkdown(content string) []Chunk {
	mdParser := goldmark.New()
	reader := text.NewReader([]byte(content))
	docAST := mdParser.Parser().Parse(reader)

	var chunks []Chunk
	var currentChunk bytes.Buffer
	var currentHeadings []string

	for node := docAST.FirstChild(); node != nil; node = node.NextSibling() {
		if heading, ok := node.(*ast.Heading); ok && heading.Level == headingLevelToSplit {
			if currentChunk.Len() > 0 {
				chunks = append(chunks, splitSectionByWords(currentChunk.String(), currentHeadings)...)
			}
			currentChunk.Reset()
			currentHeadings = []string{string(heading.Text(reader.Source()))}
		}
		if node.Lines() == nil || node.Lines().Len() == 0 {
			continue
		}

		start := node.Lines().At(0).Start
		end := node.Lines().At(node.Lines().Len() - 1).Stop
		currentChunk.Write(reader.Source()[start:end])
		currentChunk.WriteString("\n\n")
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, splitSectionByWords(currentChunk.String(), currentHeadings)...)
	}

	return chunks
}

// splitSectionByWords splits a section and adds content hashes.
func splitSectionByWords(section string, headings []string) []Chunk {
	var finalChunks []Chunk
	words := strings.Fields(section)

	if len(words) == 0 {
		return finalChunks
	}

	if len(words) <= maxWordsPerChunk {
		content := strings.TrimSpace(section)
		finalChunks = append(finalChunks, Chunk{
			Content:     content,
			ContentHash: getContentHash(content),
			Metadata:    map[string]interface{}{"headings": strings.Join(headings, " > ")},
		})
	} else {
		var buf strings.Builder
		currentWordCount := 0
		for _, word := range words {
			buf.WriteString(word)
			buf.WriteString(" ")
			currentWordCount++

			if currentWordCount >= maxWordsPerChunk {
				content := strings.TrimSpace(buf.String())
				finalChunks = append(finalChunks, Chunk{
					Content:     content,
					ContentHash: getContentHash(content),
					Metadata:    map[string]interface{}{"headings": strings.Join(headings, " > ")},
				})
				buf.Reset()
				currentWordCount = 0
			}
		}
		if buf.Len() > 0 {
			content := strings.TrimSpace(buf.String())
			finalChunks = append(finalChunks, Chunk{
				Content:     content,
				ContentHash: getContentHash(content),
				Metadata:    map[string]interface{}{"headings": strings.Join(headings, " > ")},
			})
		}
	}
	return finalChunks
}

// chunkCodeFile treats code files as one chunk and adds a content hash.
func chunkCodeFile(content string) []Chunk {
	trimmedContent := strings.TrimSpace(content)
	return []Chunk{{
		Content:     trimmedContent,
		ContentHash: getContentHash(trimmedContent),
	}}
}
