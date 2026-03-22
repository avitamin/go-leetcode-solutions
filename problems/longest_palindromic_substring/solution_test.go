package longest_palindromic_substring

import (
	"testing"
)

func TestLongestPalindrome_CanonicalExamples(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		allowed map[string]struct{}
	}{
		{
			name:  "example_1_babad",
			input: "babad",
			allowed: map[string]struct{}{
				"bab": {},
				"aba": {},
			},
		},
		{
			name:  "example_2_cbbd",
			input: "cbbd",
			allowed: map[string]struct{}{
				"bb": {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetLogs()
			got := longestPalindrome(tt.input)
			if _, ok := tt.allowed[got]; !ok {
				Flushlogs()
				t.Fatalf("longestPalindrome(%q) = %q, allowed: %v", tt.input, got, keys(tt.allowed))
			}
		})
	}
}

func TestLongestPalindrome_ConstraintBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "min_length_single_char",
			input:    "a",
			expected: "a",
		}, {
			name:     "min_length_2_char",
			input:    "aa",
			expected: "aa",
		},
		{
			name:     "min_length_3_char",
			input:    "aaa",
			expected: "aaa",
		},
		{
			name:  "min_length_4_char",
			input: "aaaa",
			//      0123
			expected: "aaaa",
		},
		{
			name:  "min_length_4_char_2",
			input: "aaca",
			//      0123
			expected: "aca",
		},
		// {
		// 	name:     "max_length_same_char_1000",
		// 	input:    strings.Repeat("a", 1000),
		// 	expected: strings.Repeat("a", 1000),
		// },
		{
			name:     "digits_only",
			input:    "12344321",
			expected: "12344321",
		},
		{
			name:     "letters_and_digits",
			input:    "a1b2b1a",
			expected: "a1b2b1a",
		}, {
			name:     "mix",
			input:    "aacabdkacaa",
			expected: "aca",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetLogs()
			got := longestPalindrome(tt.input)
			if got != tt.expected {
				Flushlogs()
				t.Fatalf("longestPalindrome(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLongestPalindrome_SpecialCaseBehavior(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		allowed map[string]struct{}
	}{
		{
			name:  "no_palindrome_longer_than_one",
			input: "abcd",
			allowed: map[string]struct{}{
				"a": {},
				"b": {},
				"c": {},
				"d": {},
			},
		},
		{
			name:  "entire_string_is_palindrome",
			input: "racecar",
			allowed: map[string]struct{}{
				"racecar": {},
			},
		},
		{
			name:  "duplicate_letters_center",
			input: "abbaxyz",
			allowed: map[string]struct{}{
				"abba": {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetLogs()
			got := longestPalindrome(tt.input)
			if _, ok := tt.allowed[got]; !ok {
				Flushlogs()
				t.Fatalf("longestPalindrome(%q) = %q, allowed: %v", tt.input, got, keys(tt.allowed))
			}
		})
	}
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
