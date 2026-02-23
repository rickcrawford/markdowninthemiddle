package tokens

import (
	"fmt"

	"github.com/pkoukk/tiktoken-go"
)

// Counter counts tokens using a specific TikToken encoding.
type Counter struct {
	enc *tiktoken.Tiktoken
}

// NewCounter creates a token counter for the given encoding name
// (e.g. "cl100k_base", "o200k_base", "p50k_base").
func NewCounter(encoding string) (*Counter, error) {
	enc, err := tiktoken.GetEncoding(encoding)
	if err != nil {
		return nil, fmt.Errorf("loading tiktoken encoding %q: %w", encoding, err)
	}
	return &Counter{enc: enc}, nil
}

// Count returns the number of tokens in the given text.
func (c *Counter) Count(text string) int {
	tokens := c.enc.Encode(text, nil, nil)
	return len(tokens)
}
