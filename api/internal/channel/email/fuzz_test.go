package email

import (
	"fmt"
	"net/mail"
	"strings"
	"testing"
)

// FuzzEmailBody tests email body parsing with various inputs.
func FuzzEmailBody(f *testing.F) {
	// 50+ seed corpus entries for email body parsing.
	seeds := []string{
		"Hello, this is a plain text email.",
		"<html><body><p>Hello</p></body></html>",
		"<p>Simple HTML paragraph</p>",
		"",
		" ",
		"\n\n\n",
		"\r\n\r\n",
		"Line1\r\nLine2\r\nLine3",
		"<script>alert('xss')</script>Safe",
		"<style>body{color:red}</style>Visible",
		"Hello &amp; goodbye &lt;tag&gt;",
		"<div><p>Nested <b>bold <i>italic</i></b></p></div>",
		"<br><br><br>",
		"Line1<br>Line2<br/>Line3<br />Line4",
		"<a href=\"https://example.com\">Link text</a>",
		"<img src=\"https://example.com/img.png\" alt=\"image\"/>",
		"<table><tr><td>Cell1</td><td>Cell2</td></tr></table>",
		"<ul><li>Item1</li><li>Item2</li></ul>",
		"<ol><li>First</li><li>Second</li></ol>",
		"<h1>Heading</h1><p>Paragraph</p>",
		"<blockquote>Quoted text</blockquote>",
		"<pre>Preformatted\n  text</pre>",
		"<code>code snippet</code>",
		strings.Repeat("A", 1000),
		strings.Repeat("Hello World! ", 100),
		"Special chars: àéîõü ñ ß ÿ",
		"Chinese: 你好世界",
		"Japanese: こんにちは",
		"Arabic: مرحبا",
		"Emoji: 😀🎉🔥💯",
		"Mixed: Hello 你好 こんにちは",
		"Null bytes: \x00\x00\x00",
		"Control chars: \x01\x02\x03\x04\x05",
		"Tab\there\tand\tthere",
		"Backslash: \\n \\t \\r",
		"Quotes: \"double\" and 'single'",
		"Angle brackets: < > << >>",
		"Ampersand: & && &amp; &amp;amp;",
		"URL: https://example.com/path?key=value&other=123#anchor",
		"Email: user@example.com and other@test.org",
		"<p style=\"color: red; font-size: 14px;\">Styled text</p>",
		"<div class=\"container\" id=\"main\">Content</div>",
		"<!-- HTML comment -->Visible text",
		"<![CDATA[Some CDATA content]]>",
		"&nbsp;&emsp;&ensp;&thinsp;",
		"&#169; &#x00A9; &#8364;",
		"Line\x00with\x00nulls",
		"Mixed\r\nline\rendings\nhere",
		strings.Repeat("<b>", 50) + "deep" + strings.Repeat("</b>", 50),
		"<p>Unclosed tag",
		"<p>Mismatched</div>",
		"<>Empty tags</>",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, body string) {
		msg, err := mail.ReadMessage(strings.NewReader("From: test@example.com\r\n\r\n" + body))
		if err != nil {
			return
		}
		parsed, err := ParseEmail(msg)
		if err != nil {
			return
		}
		// Body should never be nil if parsing succeeded.
		_ = parsed.Body
	})
}

// FuzzMIMEParsing tests multipart MIME message parsing.
func FuzzMIMEParsing(f *testing.F) {
	// 50+ seed corpus entries for MIME parsing.
	seeds := []string{
		"Content-Type: text/plain\r\n\r\nPlain text body",
		"Content-Type: text/html\r\n\r\n<p>HTML body</p>",
		"Content-Type: text/plain; charset=utf-8\r\n\r\nUTF-8 body",
		"Content-Type: text/plain; charset=iso-8859-1\r\n\r\nLatin body",
		"Content-Type: text/plain\r\nContent-Transfer-Encoding: base64\r\n\r\nSGVsbG8gV29ybGQ=",
		"Content-Type: text/plain\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\nHello =3D World",
		"Content-Type: text/plain\r\nContent-Transfer-Encoding: 7bit\r\n\r\n7bit body",
		"Content-Type: text/plain\r\nContent-Transfer-Encoding: 8bit\r\n\r\n8bit body",
		"Content-Type: application/pdf\r\nContent-Disposition: attachment; filename=\"test.pdf\"\r\n\r\nPDF",
		"Content-Type: application/octet-stream\r\nContent-Disposition: attachment; filename=\"data.bin\"\r\n\r\nBINARY",
		"Content-Type: image/png\r\nContent-Disposition: inline\r\nContent-ID: <img1>\r\n\r\nPNG",
		"Content-Type: image/jpeg\r\nContent-Disposition: inline\r\nContent-ID: <img2>\r\n\r\nJPEG",
		"Content-Type: image/gif\r\nContent-Disposition: inline; filename=\"anim.gif\"\r\n\r\nGIF",
		"Content-Type: text/csv\r\nContent-Disposition: attachment; filename=\"data.csv\"\r\n\r\na,b,c",
		"Content-Type: application/json\r\nContent-Disposition: attachment; filename=\"config.json\"\r\n\r\n{}",
		"Content-Type: application/xml\r\nContent-Disposition: attachment\r\n\r\n<root/>",
		"Content-Type: text/plain\r\n\r\n" + strings.Repeat("Long line ", 200),
		"Content-Type: text/html\r\n\r\n" + strings.Repeat("<p>Paragraph</p>", 100),
		"Content-Type: text/plain\r\n\r\nSpecial: àéîõü",
		"Content-Type: text/plain\r\n\r\n你好世界",
		"Content-Type: text/plain\r\n\r\nLine1\r\nLine2\r\nLine3",
		"Content-Type: text/plain\r\n\r\n",
		"Content-Type: text/plain\r\n\r\n ",
		"Content-Type: text/html\r\n\r\n<html><head><style>*{margin:0}</style></head><body>Text</body></html>",
		"Content-Type: text/html\r\n\r\n<html><head><script>alert(1)</script></head><body>Safe</body></html>",
		"Content-Type: text/plain\r\nContent-Disposition: inline\r\n\r\nInline text",
		"Content-Type: application/zip\r\nContent-Disposition: attachment; filename=\"archive.zip\"\r\n\r\nZIP",
		"Content-Type: audio/mpeg\r\nContent-Disposition: attachment; filename=\"song.mp3\"\r\n\r\nMP3",
		"Content-Type: video/mp4\r\nContent-Disposition: attachment; filename=\"video.mp4\"\r\n\r\nMP4",
		"Content-Type: application/vnd.ms-excel\r\nContent-Disposition: attachment; filename=\"sheet.xls\"\r\n\r\nXLS",
		"Content-Type: message/rfc822\r\n\r\nFrom: nested@example.com\r\n\r\nNested message",
		"Content-Type: text/plain; charset=windows-1252\r\n\r\nWindows text",
		"Content-Type: text/richtext\r\n\r\nRich text content",
		"Content-Type: text/plain\r\nContent-Transfer-Encoding: base64\r\n\r\nSW52YWxpZA==",
		"Content-Type: text/plain\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n=C3=A9",
		"\r\n\r\nNo content type",
		"Content-Type: text/plain\r\n\r\n\x00\x01\x02",
		"Content-Type: text/plain\r\n\r\nNulls\x00in\x00body",
		"Content-Type: multipart/alternative; boundary=\"inner\"\r\n\r\n--inner\r\nContent-Type: text/plain\r\n\r\nPlain\r\n--inner\r\nContent-Type: text/html\r\n\r\n<p>HTML</p>\r\n--inner--",
		"Content-Type: application/pdf\r\nContent-Disposition: attachment; filename=\"file with spaces.pdf\"\r\n\r\nPDF",
		"Content-Type: application/pdf\r\nContent-Disposition: attachment; filename=\"résumé.pdf\"\r\n\r\nPDF",
		"Content-Type: image/svg+xml\r\nContent-Disposition: attachment; filename=\"icon.svg\"\r\n\r\n<svg/>",
		"Content-Type: application/gzip\r\nContent-Disposition: attachment; filename=\"archive.tar.gz\"\r\n\r\nGZIP",
		"Content-Type: text/calendar\r\nContent-Disposition: attachment; filename=\"event.ics\"\r\n\r\nICS",
		"Content-Type: text/plain\r\nContent-Transfer-Encoding: binary\r\n\r\nBinary data",
		"Content-Type: image/webp\r\nContent-Disposition: inline\r\n\r\nWEBP",
		"Content-Type: font/woff2\r\nContent-Disposition: attachment; filename=\"font.woff2\"\r\n\r\nFONT",
		"Content-Type: application/wasm\r\nContent-Disposition: attachment; filename=\"module.wasm\"\r\n\r\nWASM",
		"Content-Type: text/plain\r\n\r\nEmoji body 😀🎉🔥",
		"Content-Type: text/html\r\n\r\n<marquee>Old HTML</marquee>",
		"Content-Type: text/plain\r\n\r\n" + strings.Repeat("\n", 100),
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, part string) {
		boundary := "fuzzboundary"
		raw := fmt.Sprintf("From: fuzz@example.com\r\nContent-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n--%s\r\n%s\r\n--%s--\r\n",
			boundary, boundary, part, boundary)

		msg, err := mail.ReadMessage(strings.NewReader(raw))
		if err != nil {
			return
		}
		parsed, err := ParseEmail(msg)
		if err != nil {
			return
		}
		_ = parsed.Attachments
		_ = parsed.Body
	})
}

// FuzzEmailHeaders tests email header parsing robustness.
func FuzzEmailHeaders(f *testing.F) {
	// 50+ seed corpus entries for header parsing.
	seeds := []string{
		"From: alice@example.com",
		"From: Alice <alice@example.com>",
		"From: \"Alice Smith\" <alice@example.com>",
		"To: bob@example.com",
		"To: bob@example.com, carol@example.com",
		"To: \"Bob\" <bob@example.com>, Carol <carol@example.com>",
		"Cc: cc1@example.com, cc2@example.com",
		"Bcc: bcc@example.com",
		"Subject: Hello World",
		"Subject: Re: Hello World",
		"Subject: Fwd: Hello World",
		"Subject: Re: Re: Re: Deep thread",
		"Subject: =?UTF-8?B?SGVsbG8gV29ybGQ=?=",
		"Subject: =?UTF-8?Q?Hello_World?=",
		"Subject: =?ISO-8859-1?Q?caf=E9?=",
		"Message-ID: <msg001@example.com>",
		"Message-ID: <unique.123.456@mail.example.com>",
		"In-Reply-To: <parent@example.com>",
		"References: <ref1@example.com> <ref2@example.com> <ref3@example.com>",
		"References: <single@example.com>",
		"From: sender@example.com\r\nTo: recipient@example.com",
		"From: sender@example.com\r\nTo: r1@example.com, r2@example.com\r\nCc: c1@example.com",
		"From: user@gmail.com\r\nMessage-ID: <CABx+XJ3@mail.gmail.com>",
		"From: noreply@company.com\r\nReply-To: support@company.com",
		"From: mailer-daemon@example.com",
		"From: postmaster@example.com",
		"Subject: ",
		"Subject: " + strings.Repeat("Very long subject ", 50),
		"From: ",
		"Message-ID: <>",
		"Message-ID: <@>",
		"In-Reply-To: <>",
		"References: ",
		"From: \"user@example.com\" <user@example.com>",
		"From: user+tag@example.com",
		"From: user.name@sub.domain.example.com",
		"To: undisclosed-recipients:;",
		"Subject: 🎉 Emoji Subject 🔥",
		"Subject: 你好世界",
		"From: münchen@example.de",
		"Subject: <script>alert('xss')</script>",
		"Subject: DROP TABLE users;--",
		"Message-ID: <" + strings.Repeat("a", 200) + "@example.com>",
		"From: a@b",
		"From: @invalid",
		"From: no-at-sign",
		"To: ,,,",
		"Cc: invalid-email",
		"From: null@\x00example.com",
		"Subject: null\x00subject",
		"References: not-an-angle-bracket-id",
		"In-Reply-To: malformed",
		"From: mixed <user@example.com> extra",
		"Subject: Line1\r\n\tLine2 folded",
		"From: test@example.com\r\nX-Custom-Header: custom-value",
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, headerLine string) {
		raw := headerLine + "\r\n\r\nBody"
		msg, err := mail.ReadMessage(strings.NewReader(raw))
		if err != nil {
			return
		}
		parsed, err := ParseEmail(msg)
		if err != nil {
			return
		}
		// Should never panic.
		_ = parsed.MessageID
		_ = parsed.From
		_ = parsed.To
		_ = parsed.CC
		_ = parsed.Subject
		_ = parsed.InReplyTo
		_ = parsed.References
	})
}
