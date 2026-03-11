package upload

import (
	"testing"
)

// FuzzValidateContentType tests that ValidateContentType never panics.
func FuzzValidateContentType(f *testing.F) {
	seeds := []string{
		"image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml",
		"application/pdf", "text/plain", "text/csv", "text/markdown",
		"application/json", "application/xml", "application/zip",
		"application/octet-stream",
		"", "invalid", "text/html", "application/javascript",
		"image/jpeg; charset=utf-8", "text/plain; charset=ISO-8859-1",
		"multipart/form-data", "application/x-www-form-urlencoded",
		"audio/mp3", "video/mp4", "font/woff2",
		"application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"  image/jpeg  ", "IMAGE/JPEG",
		string(make([]byte, 5000)),
		"text/plain; boundary=something",
		"\x00\x01\x02", "image/\x00png",
		"/", "//", "type/subtype/extra",
		"application/pdf;", "text/plain;charset=utf-8",
		"image/jpeg; param=\"value\"",
		"text/html; charset=\"utf-8\"",
		"application/x-tar", "application/gzip",
		"image/tiff", "image/bmp", "image/x-icon",
		"application/wasm", "application/graphql",
		"text/css", "text/javascript", "text/xml",
		"chemical/x-pdb", "model/vrml",
		"application/x-shockwave-flash",
		"application/x-bittorrent",
		"application/x-rar-compressed",
		"application/x-7z-compressed",
		"application/vnd.api+json",
		"application/hal+json",
		"application/ld+json",
		"application/schema+json",
		"application/geo+json",
		"application/problem+json",
		"text/event-stream",
		"message/http",
		"multipart/mixed",
		"multipart/alternative",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, contentType string) {
		// Must not panic.
		result := ValidateContentType(contentType)
		_ = result
	})
}
