package embed

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Chunk represents a piece of content to be embedded.
type Chunk struct {
	Content  string
	Metadata map[string]interface{}
}

// Approximation: target maximum words per chunk
const maxWordsPerChunk = 256 // Reduced for better context sizing

// Markdown heading level to split sections (Level 2 => ##)
const headingLevelToSplit = 2

// ChunkMarkdown precisely splits Markdown content by iterating through its main blocks.
func ChunkMarkdown(content string) []Chunk {
	mdParser := goldmark.New()
	reader := text.NewReader([]byte(content))
	docAST := mdParser.Parser().Parse(reader)

	var chunks []Chunk
	var currentChunk bytes.Buffer
	var currentHeadings []string

	// Instead of ast.Walk, we iterate through the document's main blocks. This is safer.
	for node := docAST.FirstChild(); node != nil; node = node.NextSibling() {
		// Check if the current block is a heading of the level we want to split by.
		if heading, ok := node.(*ast.Heading); ok && heading.Level == headingLevelToSplit {
			// If the current chunk has content, finalize it and add it to our list.
			if currentChunk.Len() > 0 {
				chunks = append(chunks, splitSectionByWords(currentChunk.String(), currentHeadings)...)
			}
			// Reset the buffer and update the current heading context.
			currentChunk.Reset()
			currentHeadings = []string{string(heading.Text(reader.Source()))}
		}

		// Append the raw text of the current block to the chunk buffer.
		// This is safe because we are only iterating over block-level nodes.
		start := node.Lines().At(0).Start
		end := node.Lines().At(node.Lines().Len() - 1).Stop
		currentChunk.Write(reader.Source()[start:end])
		currentChunk.WriteString("\n\n") // Add double newline for paragraph separation
	}

	// Don't forget the very last chunk in the file.
	if currentChunk.Len() > 0 {
		chunks = append(chunks, splitSectionByWords(currentChunk.String(), currentHeadings)...)
	}

	return chunks
}

// splitSectionByWords splits a section if it exceeds maxWordsPerChunk.
func splitSectionByWords(section string, headings []string) []Chunk {
	var finalChunks []Chunk
	words := strings.Fields(section)

	if len(words) == 0 {
		return finalChunks
	}

	if len(words) <= maxWordsPerChunk {
		finalChunks = append(finalChunks, Chunk{
			Content:  section,
			Metadata: map[string]interface{}{"headings": strings.Join(headings, " > ")},
		})
	} else {
		var buf strings.Builder
		currentWordCount := 0
		for _, word := range words {
			buf.WriteString(word)
			buf.WriteString(" ")
			currentWordCount++

			if currentWordCount >= maxWordsPerChunk {
				finalChunks = append(finalChunks, Chunk{
					Content:  strings.TrimSpace(buf.String()),
					Metadata: map[string]interface{}{"headings": strings.Join(headings, " > ")},
				})
				buf.Reset()
				currentWordCount = 0
			}
		}
		if buf.Len() > 0 {
			finalChunks = append(finalChunks, Chunk{
				Content:  strings.TrimSpace(buf.String()),
				Metadata: map[string]interface{}{"headings": strings.Join(headings, " > ")},
			})
		}
	}
	return finalChunks
}

// chunkCodeFile treats code files as one chunk for now.
func chunkCodeFile(content string) []Chunk {
	// TODO: Later, you can implement a proper AST-based chunker for code here.
	return []Chunk{{Content: content}}
}
