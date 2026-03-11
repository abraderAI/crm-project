package webhook

import (
	"testing"
)

// FuzzWebhookURL tests that isValidWebhookURL never panics.
func FuzzWebhookURL(f *testing.F) {
	seeds := []string{
		"http://example.com", "https://example.com", "ftp://bad.com",
		"", "not-a-url", "http://", "https://", "HTTP://UPPER.COM",
		"http://localhost:8080/hook", "https://user:pass@host.com",
		"http://192.168.1.1", "http://[::1]", "http://example.com/path?q=1",
		"javascript:alert(1)", "file:///etc/passwd", "data:text/html,<h1>",
		string(make([]byte, 10000)), // very long URL
		"http://a.b.c.d.e.f.g", "https://very-long-subdomain.example.com",
		"\x00\x01\x02", "http://\x00bad", "https://test\nnewline",
		"http://example.com#fragment", "http://example.com?query=1&b=2",
		"http://example.com/path/to/webhook", "https://api.stripe.com/v1/webhooks",
		"http://localhost", "http://127.0.0.1:3000/hook",
		"gopher://old.protocol.com", "ws://websocket.com",
		"wss://secure-websocket.com", "mqtt://iot.example.com",
		"http://example.com:99999", "http://example.com:-1",
		"http://example.com:abc", "http://example com/space",
		"http://example.com/パス", "http://example.com/путь",
		"http://example.com/%20encoded", "http://example.com/a%00b",
		"http://example.com/../../etc/passwd", "http://example.com/../..",
		"http://evil.com@good.com", "http://good.com\\@evil.com",
		"http://example.com/hook?callback=http://evil.com",
		"https://example.com/hook#fragment=bad",
		"http://例え.jp", "http://münchen.de",
		"http://xn--nxasmq6b.example", "http://example.com:80",
		"https://example.com:443/standard",
		"http://example.com/very/deep/nested/path/to/webhook/endpoint",
		"http://example.com/" + string(make([]byte, 5000)),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, url string) {
		// Must not panic.
		result := isValidWebhookURL(url)
		_ = result
	})
}

// FuzzMatchesFilter tests that matchesFilter never panics.
func FuzzMatchesFilter(f *testing.F) {
	seeds := []string{
		"[]", "", `["org.created"]`, `["org.created","org.updated"]`,
		"not-json", `{"object":true}`, `null`, `"string"`,
		`[1,2,3]`, `[null]`, `[true]`, `[""]`,
		string(make([]byte, 1000)),
		`["a","b","c","d","e","f","g","h","i","j"]`,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, filterJSON string) {
		// Must not panic.
		result := matchesFilter(filterJSON, "org.created")
		_ = result
	})
}
