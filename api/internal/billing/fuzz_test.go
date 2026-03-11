package billing

import (
	"context"
	"testing"
)

// FuzzWebhookPayload fuzzes the FlexPoint webhook handler with random payloads.
func FuzzWebhookPayload(f *testing.F) {
	// Seed corpus — ≥50 entries per spec requirement.
	seeds := []struct {
		eventType  string
		orgID      string
		customerID string
		data       string
	}{
		{EventPaymentSucceeded, "org-1", "cust-1", `{}`},
		{EventPaymentFailed, "org-2", "cust-2", `{}`},
		{EventSubscriptionCreated, "org-3", "cust-3", `{"billing_tier":"pro"}`},
		{EventSubscriptionCanceled, "org-4", "cust-4", `{}`},
		{EventInvoicePaid, "org-5", "cust-5", `{}`},
		{EventInvoiceOverdue, "org-6", "cust-6", `{}`},
		{EventCustomerCreated, "org-7", "cust-7", `{}`},
		{"unknown.event", "org-8", "cust-8", `{}`},
		{"", "org-9", "", `{}`},
		{"payment.succeeded", "", "", `{}`},
		{"payment.succeeded", "org-10", "cust-10", `{"tier":"enterprise"}`},
		{"subscription.created", "org-11", "", `{"billing_tier":"starter"}`},
		{"subscription.created", "org-12", "", `{"tier":"pro"}`},
		{"subscription.created", "org-13", "", `{"other":"value"}`},
		{"subscription.created", "org-14", "", ``},
		{"invoice.paid", "org-15", "cust-15", `{"amount":100}`},
		{"invoice.overdue", "org-16", "cust-16", `{"days":30}`},
		{"customer.created", "org-17", "cust-new", `{}`},
		{"payment.succeeded", "org-18", "", `not-json`},
		{"payment.failed", "org-19", "", `[1,2,3]`},
		{"test.event.one", "org-20", "cust-20", `{"key":"val"}`},
		{"test.event.two", "org-21", "", `{}`},
		{"test.event.three", "", "cust-22", `{}`},
		{"", "", "", ``},
		{"a", "b", "c", "d"},
		{"payment.succeeded", "aaaa-bbbb-cccc-dddd", "cust-uuid", `{"nested":{"deep":"value"}}`},
		{"subscription.created", "org-x", "", `{"billing_tier":""}`},
		{"subscription.created", "org-y", "", `{"billing_tier":123}`},
		{"payment.refunded", "org-26", "cust-26", `{}`},
		{"account.updated", "org-27", "cust-27", `{"plan":"enterprise"}`},
		{"plan.changed", "org-28", "cust-28", `{"from":"free","to":"pro"}`},
		{"charge.succeeded", "org-29", "", `{"amount":9999}`},
		{"charge.failed", "org-30", "", `{"reason":"insufficient_funds"}`},
		{"payout.created", "org-31", "cust-31", `{}`},
		{"payout.paid", "org-32", "", `{}`},
		{"transfer.created", "org-33", "", `{}`},
		{"dispute.created", "org-34", "cust-34", `{}`},
		{"dispute.closed", "org-35", "cust-35", `{"status":"won"}`},
		{"mandate.updated", "org-36", "cust-36", `{}`},
		{"tax.rate.updated", "org-37", "", `{"rate":0.2}`},
		{"coupon.created", "org-38", "", `{"code":"SAVE50"}`},
		{"coupon.deleted", "org-39", "", `{}`},
		{"balance.available", "org-40", "", `{"amount":5000}`},
		{"reporting.report_run.succeeded", "org-41", "", `{}`},
		{"identity.verification_session.created", "org-42", "", `{}`},
		{"checkout.session.completed", "org-43", "cust-43", `{"mode":"subscription"}`},
		{"payment_intent.created", "org-44", "cust-44", `{}`},
		{"setup_intent.created", "org-45", "cust-45", `{}`},
		{"price.updated", "org-46", "", `{"unit_amount":1999}`},
		{"product.created", "org-47", "", `{"name":"Pro Plan"}`},
	}

	for _, s := range seeds {
		f.Add(s.eventType, s.orgID, s.customerID, s.data)
	}

	fp := NewFlexPointProvider("")
	f.Fuzz(func(t *testing.T, eventType, orgID, customerID, data string) {
		payload := WebhookPayload{
			EventType:  eventType,
			OrgID:      orgID,
			CustomerID: customerID,
			Data:       data,
		}

		// Should never panic regardless of input.
		result, err := fp.HandleWebhook(context.Background(), payload)
		if eventType == "" {
			if err == nil {
				t.Error("expected error for empty event_type")
			}
			return
		}
		if err != nil {
			return // Non-empty event type errors are acceptable for malformed signatures.
		}
		if result == nil {
			t.Error("result should not be nil when no error")
		}
	})
}

// FuzzWebhookSignature fuzzes HMAC-SHA256 signature verification.
func FuzzWebhookSignature(f *testing.F) {
	seeds := []struct {
		body   string
		secret string
	}{
		{`{"event":"test"}`, "secret1"},
		{`{}`, ""},
		{`{"nested":{"deep":"value"}}`, "long-secret-key-123"},
		{`null`, "s"},
		{`[]`, "abc"},
		{`{"a":"b","c":"d"}`, "key"},
		{`simple text`, "another-secret"},
		{`{"amount":9999}`, "billing-secret"},
		{`{"type":"payment.succeeded"}`, "webhook-key"},
		{`{"customer_id":"cust_123"}`, "test"},
		{string(make([]byte, 0)), "empty-body"},
		{string(make([]byte, 1024)), "large-body"},
		{`{"special":"chars!@#$%^&*()"}`, "special"},
		{`{"unicode":"日本語テスト"}`, "unicode-secret"},
		{`{"emoji":"🎉🎊"}`, "emoji-key"},
		{`<xml>test</xml>`, "xml-secret"},
		{`key=value&other=data`, "form-secret"},
		{`line1\nline2\nline3`, "newline-secret"},
		{`\t\ttabbed`, "tab-secret"},
		{`"quoted string"`, "quote-secret"},
		{`true`, "bool-secret"},
		{`12345`, "num-secret"},
		{`-1.5e10`, "float-secret"},
		{`{"a":null}`, "null-secret"},
		{`{"arr":[1,2,3]}`, "arr-secret"},
		{`{"deep":{"nested":{"value":true}}}`, "deep-secret"},
		{`repeat repeat repeat`, "repeat-secret"},
		{`{"key":"` + string(make([]byte, 500)) + `"}`, "big-val-secret"},
		{``, "empty-body-secret"},
		{` `, "space-body-secret"},
		{`{`, "incomplete-json"},
		{`{"unterminated": "value`, "bad-json"},
		{`{"key": }`, "malformed-json"},
		{`[1, 2,`, "incomplete-arr"},
		{`{"billing_tier":"free"}`, "tier-secret"},
		{`{"billing_tier":"pro","amount":100}`, "combo-secret"},
		{`{"payment_status":"active"}`, "status-secret"},
		{`{"customer_id":"fp_cust_abc123"}`, "cust-secret"},
		{`{"org_id":"01234567-89ab-cdef-0123-456789abcdef"}`, "org-secret"},
		{`{"event_type":"payment.succeeded","timestamp":1234567890}`, "ts-secret"},
		{`{"retry_count":3,"max_retries":5}`, "retry-secret"},
		{`{"webhook_id":"wh_abc123"}`, "wh-secret"},
		{`{"version":"2024-01-01"}`, "ver-secret"},
		{`{"source":"flexpoint","env":"production"}`, "src-secret"},
		{`{"metadata":{"key":"value"}}`, "meta-secret"},
		{`{"items":[{"id":"1"},{"id":"2"}]}`, "items-secret"},
		{`{"currency":"USD","amount":1000}`, "curr-secret"},
		{`{"interval":"month","count":1}`, "interval-secret"},
		{`{"tax":{"rate":0.08,"amount":80}}`, "tax-secret"},
		{`{"discount":{"percent":20}}`, "disc-secret"},
	}

	for _, s := range seeds {
		f.Add(s.body, s.secret)
	}

	f.Fuzz(func(t *testing.T, body, secret string) {
		bodyBytes := []byte(body)

		// Compute signature.
		sig := ComputeWebhookSignature(bodyBytes, secret)
		if sig == "" {
			t.Error("signature should never be empty")
		}

		// Verify with correct secret should always pass.
		if !VerifyWebhookSignature(bodyBytes, sig, secret) {
			t.Error("verification should pass with correct secret and body")
		}

		// Verify with wrong secret should always fail (unless secret is same).
		wrongSig := ComputeWebhookSignature(bodyBytes, secret+"x")
		if VerifyWebhookSignature(bodyBytes, wrongSig, secret) {
			t.Error("verification should fail with wrong secret")
		}
	})
}

// FuzzCreateCustomerInput fuzzes customer creation input validation.
func FuzzCreateCustomerInput(f *testing.F) {
	seeds := []struct {
		name  string
		orgID string
		email string
	}{
		{"Test Co", "org-1", "test@example.com"},
		{"", "org-2", "test@example.com"},
		{"Valid Name", "", "test@example.com"},
		{"", "", ""},
		{"A", "B", "C"},
		{"Very Long Company Name That Goes On And On", "org-id-123", "long@email.com"},
		{"Unicode 日本語", "org-jp", "jp@test.com"},
		{"Special !@#$", "org-spec", "spec@test.com"},
		{"  spaces  ", "org-sp", "sp@test.com"},
		{"Name", "org-10", ""},
	}

	for _, s := range seeds {
		f.Add(s.name, s.orgID, s.email)
	}

	fp := NewFlexPointProvider("")
	f.Fuzz(func(t *testing.T, name, orgID, email string) {
		input := CreateCustomerInput{Name: name, OrgID: orgID, Email: email}
		customer, err := fp.CreateCustomer(context.Background(), input)
		if name == "" || orgID == "" {
			if err == nil {
				t.Error("expected error for empty name or orgID")
			}
			return
		}
		if err != nil {
			t.Errorf("unexpected error for valid input: %v", err)
		}
		if customer == nil {
			t.Error("customer should not be nil for valid input")
		}
	})
}

// FuzzCreateInvoiceInput fuzzes invoice creation input validation.
func FuzzCreateInvoiceInput(f *testing.F) {
	seeds := []struct {
		customerID  string
		amount      float64
		currency    string
		description string
	}{
		{"cust-1", 99.99, "USD", "Monthly plan"},
		{"", 99.99, "USD", "Missing customer"},
		{"cust-2", 0, "USD", "Zero amount"},
		{"cust-3", -10, "USD", "Negative amount"},
		{"cust-4", 100, "", "Missing currency"},
		{"cust-5", 0.01, "EUR", "Minimum amount"},
		{"cust-6", 999999.99, "GBP", "Large amount"},
		{"cust-7", 50, "JPY", "Yen invoice"},
		{"cust-8", 1, "CHF", "Small amount"},
		{"cust-9", 10.50, "CAD", "Decimal amount"},
	}

	for _, s := range seeds {
		f.Add(s.customerID, s.amount, s.currency, s.description)
	}

	fp := NewFlexPointProvider("")
	f.Fuzz(func(t *testing.T, customerID string, amount float64, currency, description string) {
		input := CreateInvoiceInput{
			CustomerID: customerID, Amount: amount, Currency: currency, Description: description,
		}
		invoice, err := fp.CreateInvoice(context.Background(), input)
		if customerID == "" || amount <= 0 || currency == "" {
			if err == nil {
				t.Error("expected error for invalid input")
			}
			return
		}
		if err != nil {
			t.Errorf("unexpected error for valid input: %v", err)
		}
		if invoice == nil {
			t.Error("invoice should not be nil for valid input")
		}
	})
}

// FuzzMapEventToMetadata fuzzes the event-to-metadata mapping.
func FuzzMapEventToMetadata(f *testing.F) {
	seeds := []struct {
		eventType  string
		customerID string
		data       string
	}{
		{EventPaymentSucceeded, "cust-1", `{}`},
		{EventPaymentFailed, "cust-2", `{}`},
		{EventSubscriptionCreated, "cust-3", `{"billing_tier":"pro"}`},
		{EventSubscriptionCanceled, "cust-4", `{}`},
		{EventInvoicePaid, "cust-5", `{}`},
		{EventInvoiceOverdue, "cust-6", `{}`},
		{EventCustomerCreated, "cust-7", `{}`},
		{"unknown", "", `{}`},
		{"", "", ``},
		{"custom.event", "cust-8", `{"key":"val"}`},
	}

	for _, s := range seeds {
		f.Add(s.eventType, s.customerID, s.data)
	}

	f.Fuzz(func(t *testing.T, eventType, customerID, data string) {
		result := &WebhookResult{Processed: eventType != "", EventType: eventType}
		payload := WebhookPayload{EventType: eventType, CustomerID: customerID, Data: data}

		// Should never panic.
		meta := MapEventToMetadata(result, payload)
		if meta == nil {
			t.Error("meta should never be nil")
		}
	})
}

// FuzzExtractTierFromData fuzzes tier extraction from webhook data.
func FuzzExtractTierFromData(f *testing.F) {
	seeds := []string{
		`{"billing_tier":"pro"}`,
		`{"tier":"starter"}`,
		`{}`,
		``,
		`not-json`,
		`{"billing_tier":""}`,
		`{"billing_tier":123}`,
		`{"tier":null}`,
		`{"other":"field"}`,
		`[1,2,3]`,
		`{"billing_tier":"free","tier":"pro"}`,
		`{"nested":{"billing_tier":"enterprise"}}`,
		`null`,
		`true`,
		`"string"`,
		`42`,
	}

	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data string) {
		// Should never panic.
		tier := extractTierFromData(data)
		_ = tier
	})
}
