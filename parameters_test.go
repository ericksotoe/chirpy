package main

import (
	"testing"
)

func TestCleanUpBadWords(t *testing.T) {
	// 1. Define your test cases
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no bad words",
			input:    "I love coding in go",
			expected: "I love coding in go",
		},
		{
			name:     "single bad word",
			input:    "this is a kerfuffle",
			expected: "this is a ****",
		},
		{
			name:     "multiple bad words and casing",
			input:    "Fornax and Sharbert are banned",
			expected: "**** and **** are banned",
		},
		{
			name:     "empty body",
			input:    "",
			expected: "",
		},
		{
			name:     "puncuation bad word",
			input:    "Sharbert! that was an awesome Fornax@",
			expected: "Sharbert! that was an awesome Fornax@",
		},
	}

	// 2. Iterate through cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize the struct with the test input
			params := &parameters{Body: tt.input}

			// Call the function under test
			cleanUpBadWords(params)

			// 3. Assert the results
			if params.Body != tt.expected {
				t.Errorf("cleanUpBadWords() failed for case '%s'\n got:  %q\n want: %q",
					tt.name, params.Body, tt.expected)
			}
		})
	}
}
