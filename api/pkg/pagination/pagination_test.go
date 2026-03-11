package pagination

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_Defaults(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: ""}}
	p := Parse(r)
	assert.Equal(t, DefaultLimit, p.Limit)
	assert.Equal(t, "", p.Cursor)
}

func TestParse_CustomLimit(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "limit=25"}}
	p := Parse(r)
	assert.Equal(t, 25, p.Limit)
}

func TestParse_LimitClampMin(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "limit=0"}}
	p := Parse(r)
	assert.Equal(t, 1, p.Limit)
}

func TestParse_LimitClampNegative(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "limit=-5"}}
	p := Parse(r)
	assert.Equal(t, 1, p.Limit)
}

func TestParse_LimitClampMax(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "limit=999"}}
	p := Parse(r)
	assert.Equal(t, MaxLimit, p.Limit)
}

func TestParse_InvalidLimit(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "limit=abc"}}
	p := Parse(r)
	assert.Equal(t, DefaultLimit, p.Limit)
}

func TestParse_WithCursor(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "cursor=abc123"}}
	p := Parse(r)
	assert.Equal(t, "abc123", p.Cursor)
}

func TestParse_CursorAndLimit(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "cursor=xyz&limit=10"}}
	p := Parse(r)
	assert.Equal(t, "xyz", p.Cursor)
	assert.Equal(t, 10, p.Limit)
}

func TestEncodeDecode_Roundtrip(t *testing.T) {
	id := uuid.New()
	encoded := EncodeCursor(id)
	decoded, err := DecodeCursor(encoded)
	require.NoError(t, err)
	assert.Equal(t, id, decoded)
}

func TestDecodeCursor_Empty(t *testing.T) {
	id, err := DecodeCursor("")
	assert.NoError(t, err)
	assert.Equal(t, uuid.Nil, id)
}

func TestDecodeCursor_InvalidBase64(t *testing.T) {
	_, err := DecodeCursor("not!valid!base64!!!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cursor encoding")
}

func TestDecodeCursor_MissingPrefix(t *testing.T) {
	_, err := DecodeCursor("bm9wcmVmaXg=") // base64("noprefix")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cursor format")
}

func TestDecodeCursor_InvalidUUID(t *testing.T) {
	// base64("cursor:not-a-uuid")
	_, err := DecodeCursor("Y3Vyc29yOm5vdC1hLXV1aWQ=")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cursor UUID")
}

func TestDecodeCursor_ShortInput(t *testing.T) {
	// base64("cur") - shorter than prefix
	_, err := DecodeCursor("Y3Vy")
	assert.Error(t, err)
}

func TestEncodeCursor_DifferentUUIDs(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	assert.NotEqual(t, EncodeCursor(id1), EncodeCursor(id2))
}

func TestPageInfo_JSON(t *testing.T) {
	pi := PageInfo{NextCursor: "abc", HasMore: true}
	assert.Equal(t, "abc", pi.NextCursor)
	assert.True(t, pi.HasMore)
}

func TestParse_ExactMaxLimit(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "limit=100"}}
	p := Parse(r)
	assert.Equal(t, 100, p.Limit)
}

func TestParse_ExactMinLimit(t *testing.T) {
	r := &http.Request{URL: &url.URL{RawQuery: "limit=1"}}
	p := Parse(r)
	assert.Equal(t, 1, p.Limit)
}

// FuzzDecodeCursor exercises cursor decoding with random inputs.
func FuzzDecodeCursor(f *testing.F) {
	seeds := []string{
		"", "abc", "Y3Vyc29yOm5vdC1hLXV1aWQ=",
		"bm9wcmVmaXg=", "not!valid!base64!!!",
		"Y3Vy", "====", "a]]]", "AAAA",
		EncodeCursor(uuid.New()),
		EncodeCursor(uuid.Nil),
		"Y3Vyc29yOjAxOTU3ZDQ4LTdmZDgtNzAxZi1iOTUyLWQ1NjNiNjgxZWYwMQ==",
		"dGVzdA==", "Zm9v", "YmFy", "cXV4",
		"!!!", "+++", "///", "___",
		"\x00\x01\x02\x03",
		"a]]]===", "////", "++==",
		"cursor:invalid", "cursor:", "cursor::",
		"Y3Vyc29yOg==", // cursor:
		"Y3Vyc29yOjo=", // cursor::
		"QUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQQ==",
		"verylongstringthatisnotbase64butmightcauseissues",
		"c3VwZXJsb25nc3RyaW5nd2l0aG5vdXVpZA==",
		"Y3Vyc29yOmFiYy1kZWYtZ2hpLWprbC1tbm8=",
		"Y3Vyc29yOjAwMDAwMDAwLTAwMDAtMDAwMC0wMDAwLTAwMDAwMDAwMDAwMA==",
		" ", "\t", "\n", "\r\n",
		"a", "ab", "abc", "abcd",
		"MTIzNDU2Nzg5MA==",
		"🎉", "日本語", "العربية",
		"cursor:00000000-0000-0000-0000-000000000000",
		"Y3Vyc29yOjAxMjM0NTY3LTg5YWItY2RlZi0wMTIzLTQ1Njc4OWFiY2RlZg==",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		id, err := DecodeCursor(input)
		// Should never panic, errors are fine.
		if err == nil && input != "" {
			// If it decoded successfully, verify it round-trips.
			reEncoded := EncodeCursor(id)
			reDecoded, err2 := DecodeCursor(reEncoded)
			assert.NoError(t, err2)
			assert.Equal(t, id, reDecoded)
		}
	})
}
