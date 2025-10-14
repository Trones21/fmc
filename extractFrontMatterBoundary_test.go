package main

import (
	"testing"
)

// TestExtractFrontMatterBoundary tests the extractFrontMatterBoundary function.
func TestExtractFrontMatterBoundary(t *testing.T) {
	tests := []struct {
		content   string
		expected  string
		expectErr bool
	}{
		{
			content: `---
title: Test
date: 2024-12-21
---   
<the contents of the article>`,

			expected: `---
title: Test
date: 2024-12-21
---`,
			expectErr: false,
		},
		// {
		// 	content: `---
		// 	title: Missing End`,
		// 	expected:  "",
		// 	expectErr: true,
		// },
		// {
		// 	content:   `No Delimiters Here`,
		// 	expected:  "",
		// 	expectErr: true,
		// },
		// {
		// 	content: `---
		// 	---
		// 	content here`,
		// 	expected:  "",
		// 	expectErr: false,
		// },
	}

	for _, test := range tests {
		result, err := extractFrontMatterBoundary(test.content)
		if test.expectErr {
			if err == nil {
				t.Errorf("expected error but got none for content: %s", test.content)
				continue
			}
		} else {
			if err != nil {
				t.Errorf("did not expect error but got: %v for content: %s", err, test.content)
				continue
			}
			if result != test.expected {
				t.Errorf("test_expected: %s \n test_output: %s \n  test_input: %s \n", test.expected, result, test.content)
				continue
			}
		}
	}
}
