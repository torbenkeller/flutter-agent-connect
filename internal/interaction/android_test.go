package interaction

import "testing"

func TestEscapeForADBInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "email address",
			input:    "user@example.com",
			expected: `user\@example.com`,
		},
		{
			name:     "spaces become %s",
			input:    "hello world",
			expected: "hello%sworld",
		},
		{
			name:     "password with special chars",
			input:    "P@ss#w0rd!",
			expected: `P\@ss\#w0rd\!`,
		},
		{
			name:     "url with query params",
			input:    "https://example.com?a=1&b=2",
			expected: `https://example.com\?a=1\&b=2`,
		},
		{
			name:     "shell metacharacters",
			input:    `$HOME | rm -rf *`,
			expected: `\$HOME%s\|%srm%s-rf%s\*`,
		},
		{
			name:     "quotes and backticks",
			input:    `it's "quoted" and ` + "`backticked`",
			expected: `it\'s%s\"quoted\"%sand%s` + "\\`backticked\\`",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "brackets and braces",
			input:    "{key}=[value]",
			expected: `\{key\}=\[value\]`,
		},
		{
			name:     "all dangerous chars",
			input:    `@#$%^&*(){}[]<>|~!?;`,
			expected: `\@\#\$\%\^\&\*\(\)\{\}\[\]\<\>\|\~\!\?\;`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeForADBInput(tt.input)
			if got != tt.expected {
				t.Errorf("escapeForADBInput(%q)\n  got:  %s\n  want: %s", tt.input, got, tt.expected)
			}
		})
	}
}
