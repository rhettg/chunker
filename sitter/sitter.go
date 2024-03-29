package sitter

import (
	"context"
	"fmt"
	"unicode"

	"github.com/rhettg/chunker/plaintext"
	sitter "github.com/smacker/go-tree-sitter"
)

type Chunks struct {
	sourceCode []byte
	minSize    uint32
	maxSize    uint32
	c          *sitter.TreeCursor

	offset uint32
}

func New(l *sitter.Language, sourceCode []byte, minSize, maxSize int) (*Chunks, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(l)

	tree, err := parser.ParseCtx(context.Background(), nil, sourceCode)
	if err != nil {
		return nil, err
	}

	c := sitter.NewTreeCursor(tree.RootNode())
	if !c.GoToFirstChild() {
		return nil, fmt.Errorf("no first child")
	}

	ch := Chunks{
		sourceCode: sourceCode,
		minSize:    uint32(minSize),
		maxSize:    uint32(maxSize),
		c:          c,
	}

	return &ch, nil
}

func endOnLines(minSize, maxSize uint32, sourceCode []byte) uint32 {
	for i := uint32(0); i < uint32(len(sourceCode)); i++ {
		if sourceCode[i] == '\n' {
			if i > minSize {
				return i
			}
		}
		if i > maxSize {
			return 0
		}
	}
	return uint32(len(sourceCode))
}

func endOnWhitespace(minSize, maxSize uint32, sourceCode []byte) uint32 {
	for i := uint32(0); i < uint32(len(sourceCode)); i++ {
		if unicode.IsSpace(rune(sourceCode[i])) && i > minSize {
			return i
		}
		if i > maxSize {
			return 0
		}
	}
	return uint32(len(sourceCode))
}

func (c *Chunks) Next() ([]byte, bool) {
	n := c.c.CurrentNode()
	start := n.StartByte()
	end := n.EndByte()

	start += c.offset

	// The natural chunk / node could be larger than our max, we'll have to break it up.
	if end-start > c.maxSize {
		// TODO: I could imagine it being *even* better to futher use the
		// tree-sitter structures to slowly build up more of the chunk.
		// Prioritizing blocks of code, for loops, etc. I'm not quite sure what
		// that would look like though. Especially in a way that would be language agnostic.
		plainSplit := plaintext.FindSplitBounds(c.sourceCode[start:], int(c.minSize), int(c.maxSize))
		end := start + uint32(plainSplit)

		if end >= n.EndByte() {
			c.offset = 0
			return c.sourceCode[start:n.EndByte()], c.c.GoToNextSibling()
		}

		c.offset += end - start
		return c.sourceCode[start:end], true
	}

	c.offset = 0

	more := false
	for c.c.GoToNextSibling() {
		c.offset = 0
		if c.c.CurrentNode().EndByte()-start > c.maxSize {
			more = true
			break
		}

		end = c.c.CurrentNode().EndByte()
	}
	return c.sourceCode[start:end], more
}
