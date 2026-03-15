import { test, expect } from "@playwright/test";

test.describe("Channel Admin UI @channel-admin", () => {
  test("navigates to /admin/channels and renders all 3 channel cards", async ({
    page,
  }) => {
    await page.goto("/admin/channels");

    // Verify the overview component renders.
    await expect(page.getByTestId("channel-overview")).toBeVisible();

    // Verify all 3 channel type cards are present.
    await expect(page.getByTestId("channel-card-email")).toBeVisible();
    await expect(page.getByTestId("channel-card-voice")).toBeVisible();
    await expect(page.getByTestId("channel-card-chat")).toBeVisible();
  });

  test("clicking Configure on email card navigates to config form", async ({
    page,
  }) => {
    await page.goto("/admin/channels");

    // Click "Configure" on the email card.
    await page.getByTestId("channel-configure-email").click();

    // Verify navigation to email config page.
    await expect(page).toHaveURL(/\/admin\/channels\/email/);

    // Verify the config form renders.
    await expect(page.getByTestId("channel-config-form")).toBeVisible();
  });
});
