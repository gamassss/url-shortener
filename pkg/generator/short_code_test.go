package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateShortCode_BasicProperties(t *testing.T) {
	code, err := GenerateShortCode()

	assert.NoError(t, err)

	assert.Len(t, code, 7, "Short code should be 7 characters long")

	assert.Regexp(t, "^[a-zA-Z0-9]+$", code, "Short code should only contain alphanumeric characters")
}

func TestGenerateShortCode_Uniqueness(t *testing.T) {
	codes := make(map[string]bool, 1000)

	for i := 0; i < 1000; i++ {
		code, err := GenerateShortCode()
		assert.NoError(t, err)

		assert.False(t, codes[code], "Duplicate code generated: %s", code)
		codes[code] = true
	}

	assert.Equal(t, 1000, len(codes), "Should generate 1000 unique codes")
}

func TestGenerateShortCode_Multiple_Calls_Different_Results(t *testing.T) {
	code1, err1 := GenerateShortCode()
	code2, err2 := GenerateShortCode()

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	assert.NotEqual(t, code1, code2, "Sequential codes should be different")
}
