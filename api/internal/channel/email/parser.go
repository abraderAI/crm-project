package email

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"regexp"
	"strings"
)

// ParsedEmail holds the normalized fields extracted from a raw email message.
type ParsedEmail struct {
	MessageID  string
	InReplyTo  string
	References []string
	From       string
	To         []string
	CC         []string
	Subject    string
	PlainBody  string
	HTMLBody   string
	// Body is the preferred body text (plain text if available, else stripped HTML).
	Body        string
	Attachments []ParsedAttachment
}

// ParsedAttachment represents a MIME attachment extracted from an email.
type ParsedAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
	IsInline    bool
	ContentID   string
}

// ParseEmail parses a *mail.Message into a normalized ParsedEmail.
func ParseEmail(msg *mail.Message) (*ParsedEmail, error) {
	if msg == nil {
		return nil, fmt.Errorf("message is nil")
	}

	parsed := &ParsedEmail{
		MessageID:  cleanHeaderValue(msg.Header.Get("Message-ID")),
		InReplyTo:  cleanHeaderValue(msg.Header.Get("In-Reply-To")),
		References: parseReferences(msg.Header.Get("References")),
		From:       extractEmailAddress(msg.Header.Get("From")),
		To:         parseAddressList(msg.Header.Get("To")),
		CC:         parseAddressList(msg.Header.Get("Cc")),
		Subject:    msg.Header.Get("Subject"),
	}

	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		// Fallback: treat as plain text.
		body, readErr := io.ReadAll(msg.Body)
		if readErr != nil {
			return nil, fmt.Errorf("reading message body: %w", readErr)
		}
		parsed.PlainBody = string(body)
		parsed.Body = parsed.PlainBody
		return parsed, nil
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		if err := parseMultipart(parsed, msg.Body, params["boundary"]); err != nil {
			return nil, fmt.Errorf("parsing multipart: %w", err)
		}
	} else {
		body, readErr := io.ReadAll(msg.Body)
		if readErr != nil {
			return nil, fmt.Errorf("reading message body: %w", readErr)
		}
		decoded := decodeBody(body, msg.Header.Get("Content-Transfer-Encoding"))
		if strings.HasPrefix(mediaType, "text/html") {
			parsed.HTMLBody = string(decoded)
		} else {
			parsed.PlainBody = string(decoded)
		}
	}

	// Set preferred body: plain text first, then stripped HTML.
	if parsed.PlainBody != "" {
		parsed.Body = parsed.PlainBody
	} else if parsed.HTMLBody != "" {
		parsed.Body = stripHTML(parsed.HTMLBody)
	}

	return parsed, nil
}

// parseMultipart recursively parses multipart MIME bodies.
func parseMultipart(parsed *ParsedEmail, body io.Reader, boundary string) error {
	if boundary == "" {
		return fmt.Errorf("empty boundary")
	}

	reader := multipart.NewReader(body, boundary)
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading next part: %w", err)
		}

		partContentType := part.Header.Get("Content-Type")
		if partContentType == "" {
			partContentType = "text/plain"
		}

		partMedia, partParams, parseErr := mime.ParseMediaType(partContentType)
		if parseErr != nil {
			// Skip parts with unparsable content type.
			_ = part.Close()
			continue
		}

		// Recurse into nested multipart.
		if strings.HasPrefix(partMedia, "multipart/") {
			if err := parseMultipart(parsed, part, partParams["boundary"]); err != nil {
				_ = part.Close()
				continue
			}
			_ = part.Close()
			continue
		}

		data, readErr := io.ReadAll(part)
		_ = part.Close()
		if readErr != nil {
			continue
		}

		decoded := decodeBody(data, part.Header.Get("Content-Transfer-Encoding"))

		disposition := part.Header.Get("Content-Disposition")
		contentID := cleanHeaderValue(part.Header.Get("Content-ID"))
		isAttachment := strings.HasPrefix(disposition, "attachment") ||
			(part.FileName() != "" && !strings.HasPrefix(disposition, "inline"))
		isInline := strings.HasPrefix(disposition, "inline") && contentID != ""

		if isAttachment || (isInline && len(decoded) > 0) {
			filename := part.FileName()
			if filename == "" {
				filename = "unnamed"
			}
			parsed.Attachments = append(parsed.Attachments, ParsedAttachment{
				Filename:    filename,
				ContentType: partMedia,
				Data:        decoded,
				IsInline:    isInline,
				ContentID:   contentID,
			})
			continue
		}

		// Text content parts.
		switch {
		case strings.HasPrefix(partMedia, "text/plain"):
			if parsed.PlainBody == "" {
				parsed.PlainBody = string(decoded)
			}
		case strings.HasPrefix(partMedia, "text/html"):
			if parsed.HTMLBody == "" {
				parsed.HTMLBody = string(decoded)
			}
		default:
			// Non-text, non-attachment parts are treated as attachments.
			filename := part.FileName()
			if filename == "" {
				filename = "unnamed"
			}
			parsed.Attachments = append(parsed.Attachments, ParsedAttachment{
				Filename:    filename,
				ContentType: partMedia,
				Data:        decoded,
				IsInline:    false,
			})
		}
	}
	return nil
}

// decodeBody handles Content-Transfer-Encoding: base64 and quoted-printable.
func decodeBody(data []byte, encoding string) []byte {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(
			strings.ReplaceAll(string(data), "\n", ""),
		)
		if err != nil {
			// Fallback: return raw data.
			return data
		}
		return decoded
	case "quoted-printable":
		return decodeQuotedPrintable(data)
	default:
		return data
	}
}

// decodeQuotedPrintable decodes quoted-printable encoded data.
func decodeQuotedPrintable(data []byte) []byte {
	var buf bytes.Buffer
	lines := bytes.Split(data, []byte("\n"))
	for i, line := range lines {
		line = bytes.TrimRight(line, "\r")
		if bytes.HasSuffix(line, []byte("=")) {
			// Soft line break — continuation.
			line = line[:len(line)-1]
			buf.Write(decodeQPLine(line))
		} else {
			buf.Write(decodeQPLine(line))
			if i < len(lines)-1 {
				buf.WriteByte('\n')
			}
		}
	}
	return buf.Bytes()
}

// decodeQPLine decodes a single line of quoted-printable text.
func decodeQPLine(line []byte) []byte {
	var buf bytes.Buffer
	for i := 0; i < len(line); i++ {
		if line[i] == '=' && i+2 < len(line) {
			hi := unhex(line[i+1])
			lo := unhex(line[i+2])
			if hi >= 0 && lo >= 0 {
				buf.WriteByte(byte(hi<<4 | lo))
				i += 2
				continue
			}
		}
		buf.WriteByte(line[i])
	}
	return buf.Bytes()
}

// unhex returns the numeric value of a hex digit, or -1.
func unhex(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	default:
		return -1
	}
}

// htmlTagRegex matches HTML tags for stripping.
var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// htmlEntityRegex matches HTML entities.
var htmlEntityRegex = regexp.MustCompile(`&[a-zA-Z0-9#]+;`)

// stripHTML removes HTML tags and decodes common entities, producing plain text.
func stripHTML(html string) string {
	// Remove style and script blocks.
	html = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`).ReplaceAllString(html, "")
	html = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(html, "")

	// Replace <br>, <p>, <div> with newlines.
	html = regexp.MustCompile(`(?i)<br\s*/?>|</p>|</div>|</li>`).ReplaceAllString(html, "\n")

	// Strip remaining tags.
	html = htmlTagRegex.ReplaceAllString(html, "")

	// Decode common entities.
	html = htmlEntityRegex.ReplaceAllStringFunc(html, decodeHTMLEntity)

	// Collapse multiple newlines.
	html = regexp.MustCompile(`\n{3,}`).ReplaceAllString(html, "\n\n")

	return strings.TrimSpace(html)
}

// decodeHTMLEntity decodes common HTML entities to their character equivalents.
func decodeHTMLEntity(entity string) string {
	entities := map[string]string{
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": `"`,
		"&apos;": "'",
		"&#39;":  "'",
		"&nbsp;": " ",
	}
	if ch, ok := entities[entity]; ok {
		return ch
	}
	return entity
}

// cleanHeaderValue removes surrounding angle brackets and whitespace from header values.
func cleanHeaderValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "<")
	value = strings.TrimSuffix(value, ">")
	return value
}

// parseReferences splits the References header into individual message IDs.
func parseReferences(refs string) []string {
	if refs == "" {
		return nil
	}
	// References header contains space-separated message IDs in angle brackets.
	parts := strings.Fields(refs)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		cleaned := cleanHeaderValue(p)
		if cleaned != "" {
			result = append(result, cleaned)
		}
	}
	return result
}

// extractEmailAddress extracts the email address from a From header value.
// Handles formats like "Name <email@example.com>" and bare "email@example.com".
func extractEmailAddress(from string) string {
	if from == "" {
		return ""
	}
	addr, err := mail.ParseAddress(from)
	if err != nil {
		// Fallback: return trimmed raw value.
		return strings.TrimSpace(from)
	}
	return addr.Address
}

// parseAddressList parses a comma-separated list of email addresses.
func parseAddressList(header string) []string {
	if header == "" {
		return nil
	}
	addrs, err := mail.ParseAddressList(header)
	if err != nil {
		// Fallback: split by comma and clean.
		parts := strings.Split(header, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, extractEmailAddress(p))
			}
		}
		return result
	}
	result := make([]string, 0, len(addrs))
	for _, a := range addrs {
		result = append(result, a.Address)
	}
	return result
}
