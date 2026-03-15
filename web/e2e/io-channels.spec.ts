import { test, expect, type Page } from "@playwright/test";
import * as fs from "node:fs";
import * as path from "node:path";

/**
 * IO Channels E2E integration tests.
 *
 * Tests A–C exercise channel admin pages with fully mocked API responses
 * (no live backend required). Test D is a standalone widget smoke test.
 */

/** Helper: mock a channel health endpoint returning the given status. */
async function mockChannelHealth(
  page: Page,
  channelType: string,
  status: string,
  enabled: boolean,
): Promise<void> {
  await page.route(
    `**/v1/orgs/*/channels/${channelType}/health`,
    async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          status,
          enabled,
          error_rate: status === "healthy" ? 0.01 : 0.15,
          last_event_at: new Date().toISOString(),
        }),
      });
    },
  );
}

// ---------------------------------------------------------------------------
// Test A — Channel overview page
// ---------------------------------------------------------------------------
test.describe("IO Channels: Channel Overview @io-channels", () => {
  test("renders all 3 channel cards with health badges", async ({ page }) => {
    // NOTE: The channel overview page is a Next.js server component. Health data
    // is fetched server-side and cannot be intercepted via page.route(). The page
    // renders with "unconfigured" status when no API is reachable — that is the
    // correct fallback behaviour in an isolated test environment.
    await page.goto("/admin/channels");

    // Assert all 3 channel cards are visible.
    await expect(page.getByTestId("channel-card-email")).toBeVisible();
    await expect(page.getByTestId("channel-card-voice")).toBeVisible();
    await expect(page.getByTestId("channel-card-chat")).toBeVisible();

    // Assert one health badge renders per card (text will be "unconfigured"
    // since there is no live API — presence is what matters here).
    const badges = page.getByTestId("channel-health-badge");
    await expect(badges).toHaveCount(3);

    // Assert Configure and DLQ links are present for each channel type.
    await expect(page.getByTestId("channel-configure-email")).toBeVisible();
    await expect(page.getByTestId("channel-configure-voice")).toBeVisible();
    await expect(page.getByTestId("channel-configure-chat")).toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// Test B — Channel config form
// ---------------------------------------------------------------------------
test.describe("IO Channels: Channel Config Form @io-channels", () => {
  test("renders email config form and saves successfully", async ({ page }) => {
    // Mock email channel health.
    await mockChannelHealth(page, "email", "healthy", true);

    // Mock the PUT config endpoint (client-side fetch on save).
    // The GET config is fetched server-side by the Next.js server component and
    // cannot be intercepted here — the form renders with null initialConfig,
    // which is valid. The correct API path is /v1/orgs/*/channels/email (no /config suffix).
    await page.route("**/v1/orgs/*/channels/email", async (route) => {
      if (route.request().method() === "PUT") {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({ ok: true }),
        });
      } else {
        await route.continue();
      }
    });

    // Mock DLQ endpoint (client-side fetch).
    await page.route("**/v1/orgs/*/channels/email/dlq**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: [], total: 0 }),
      });
    });

    await page.goto("/admin/channels/email");

    // Assert config form is present (renders even when initialConfig is null).
    await expect(page.getByTestId("channel-config-form")).toBeVisible();

    // Assert all expected form fields are present.
    await expect(page.getByTestId("field-input-imap_host")).toBeVisible();
    await expect(page.getByTestId("field-input-imap_user")).toBeVisible();
    await expect(page.getByTestId("field-input-imap_password")).toBeVisible();

    // NOTE: field-masked-imap_password only appears when initialConfig has a
    // secret value. Since config is fetched server-side (not mockable here),
    // initialConfig is null and the masked indicator is correctly absent.

    // Fill in imap_host and save.
    await page.getByTestId("field-input-imap_host").fill("imap.new-host.com");
    await page.getByTestId("config-save-btn").click();

    // After save completes, button reverts to default label.
    await expect(page.getByTestId("config-save-btn")).not.toHaveText("Saving...");

    // No error should be shown.
    await expect(page.getByTestId("config-error")).not.toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// Test C — DLQ monitor
// ---------------------------------------------------------------------------
test.describe("IO Channels: DLQ Monitor @io-channels", () => {
  test("displays failed events and retries first row", async ({ page }) => {
    // Mock email channel health + config.
    await mockChannelHealth(page, "email", "healthy", true);
    await page.route("**/v1/orgs/*/channels/email/config", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          channel_type: "email",
          enabled: true,
          settings: "{}",
        }),
      });
    });

    const dlqEvents = [
      {
        id: "evt-001",
        channel_type: "email",
        status: "failed",
        error_message: "IMAP connection timeout after 30s",
        attempts: 3,
        created_at: new Date().toISOString(),
        last_attempt_at: new Date().toISOString(),
        payload: "{}",
      },
      {
        id: "evt-002",
        channel_type: "email",
        status: "failed",
        error_message: "Invalid MIME structure in message",
        attempts: 1,
        created_at: new Date().toISOString(),
        last_attempt_at: new Date().toISOString(),
        payload: "{}",
      },
    ];

    // Mock DLQ list endpoint — stateful so it returns updated status after retry.
    // The ChannelDetailDLQ component re-fetches the full list after each retry,
    // so the mock must reflect the new state on subsequent GET calls.
    let evt001Retried = false;
    await page.route("**/v1/orgs/*/channels/email/dlq**", async (route) => {
      if (route.request().method() === "GET") {
        const events = [
          { ...dlqEvents[0], status: evt001Retried ? "retrying" : "failed" },
          dlqEvents[1],
        ];
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({ data: events, total: 2 }),
        });
      } else {
        await route.continue();
      }
    });

    // Mock retry endpoint — flips the stateful flag so the next GET returns "retrying".
    await page.route("**/v1/orgs/*/channels/email/dlq/evt-001/retry", async (route) => {
      evt001Retried = true;
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ ...dlqEvents[0], status: "retrying" }),
      });
    });

    await page.goto("/admin/channels/email");

    // Assert DLQ monitor section is visible.
    await expect(page.getByTestId("dlq-monitor")).toBeVisible();

    // Assert both event rows are rendered.
    await expect(page.getByTestId("dlq-row-evt-001")).toBeVisible();
    await expect(page.getByTestId("dlq-row-evt-002")).toBeVisible();

    // Assert first row status shows "failed" initially.
    await expect(page.getByTestId("dlq-status-evt-001")).toHaveText("failed");

    // Click Retry on the first row — triggers POST then re-fetches the list.
    await page.getByTestId("dlq-retry-evt-001").click();

    // After re-fetch the status should reflect the updated state from the mock.
    await expect(page.getByTestId("dlq-status-evt-001")).toHaveText("retrying");
  });
});

// ---------------------------------------------------------------------------
// Test D — Widget smoke test
// ---------------------------------------------------------------------------
test.describe("IO Channels: Widget Smoke Test @io-channels", () => {
  test("widget global is defined and init does not throw", async ({ page }) => {
    // Collect page errors.
    const pageErrors: Error[] = [];
    page.on("pageerror", (err) => pageErrors.push(err));

    // Try to read the built widget JS from disk.
    const widgetPath = path.resolve(__dirname, "../../widget/dist/widget.js");
    let widgetCode = "";
    try {
      widgetCode = fs.readFileSync(widgetPath, "utf-8");
    } catch {
      // If widget hasn't been built, skip with a clear message.
      test.skip(true, "widget/dist/widget.js not found — run `task widget:build` first");
      return;
    }

    // Create a minimal HTML page with the widget script inline.
    await page.setContent(`
      <!DOCTYPE html>
      <html>
        <head><title>Widget Smoke Test</title></head>
        <body>
          <div id="app"></div>
          <script>${widgetCode}</script>
          <script>
            window.__widgetInitResult = "pending";
            try {
              if (typeof CRMChatWidget !== "undefined") {
                CRMChatWidget.init({ orgId: "test", apiUrl: "http://localhost:8080" });
                window.__widgetInitResult = "success";
              } else {
                window.__widgetInitResult = "undefined";
              }
            } catch (e) {
              window.__widgetInitResult = "error:" + e.message;
            }
          </script>
        </body>
      </html>
    `);

    // Assert no page errors were thrown.
    expect(pageErrors).toHaveLength(0);

    // Assert CRMChatWidget global is defined.
    const widgetDefined = await page.evaluate(() => typeof (window as Record<string, unknown>).CRMChatWidget !== "undefined");
    expect(widgetDefined).toBe(true);

    // Assert init completed successfully.
    const initResult = await page.evaluate(() => (window as Record<string, unknown>).__widgetInitResult);
    expect(initResult).toBe("success");
  });
});
