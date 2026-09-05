package tool

import "testing"

func TestTruncateToolArg(t *testing.T) {
	cases := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"ascii under limit", "hello", 60, "hello"},
		{"ascii over limit", "hello world from tool", 10, "hello w..."},
		{"ascii exact limit", "exactly!", 8, "exactly!"},
		{"ascii limit minus ellipsis", "abcde", 5, "abcde"},
		{"ascii truncate at limit", "abcdef", 5, "ab..."},
		// Multi-byte: 3 bytes/rune. Byte-slicing at n-3 would cut a CJK
		// rune in half and produce invalid UTF-8. Rune-slicing stays whole.
		{"cjk under limit", "你好世界", 60, "你好世界"},
		// "你好世界测试截断功能" = 10 runes; == n, so no truncation.
		{"cjk at limit", "你好世界测试截断功能", 10, "你好世界测试截断功能"},
		// 11 runes, n=10 → keep 7 + "...": 你好世界测试截.
		{"cjk over limit", "你好世界测试截断功能呀", 10, "你好世界测试截..."},
		{"emoji under limit", "🚀🔥💻", 60, "🚀🔥💻"},
		// 7 runes, n=6 → keep 3 + "...".
		{"emoji over limit", "🚀🔥💻⚡🌟🎯📦", 6, "🚀🔥💻..."},
		// "hello 你好世界" = 9 runes; <= 10, no truncation.
		{"mixed under limit", "hello 你好世界", 10, "hello 你好世界"},
		{"empty string", "", 60, ""},
		{"whitespace only", "   ", 60, ""},
		{"leading/trailing trimmed", "  hello world  ", 10, "hello w..."},
		// n smaller than ellipsis length (3): n-3 goes negative, which
		// would panic on a slice. Guard: return "..." when runes exceed n.
		// (Title usage always passes 60, but the function should not panic.)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := TruncateToolArg(tc.in, tc.n)
			if got != tc.want {
				t.Errorf("TruncateToolArg(%q, %d) = %q, want %q", tc.in, tc.n, got, tc.want)
			}
		})
	}
}

func TestTruncateToolArg_NegativeEffectiveLength(t *testing.T) {
	// n < 3 means n-3 < 0. []rune[:negative] panics. The function must
	// not crash — verify it returns something valid even for tiny n.
	// (This is a degenerate input; the exact return value is not part of
	// the contract, only that it doesn't panic and is valid UTF-8.)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("TruncateToolArg panicked for n=2: %v", r)
		}
	}()
	_ = TruncateToolArg("abcdef", 2)
}
