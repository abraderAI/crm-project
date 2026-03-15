import { test, expect } from "@playwright/test";

const MOCK_SUPPORT_METRICS = {
  status_breakdown: { open: 14, in_progress: 7, resolved: 5, closed: 3 },
  volume_over_time: [
    { date: "2026-03-01", count: 5 },
    { date: "2026-03-02", count: 8 },
    { date: "2026-03-03", count: 3 },
  ],
  avg_resolution_hours: 12.5,
  tickets_by_assignee: [
    { user_id: "u1", name: "Alice", count: 10 },
    { user_id: "u2", name: "Bob", count: 5 },
  ],
  tickets_by_priority: { urgent: 3, high: 7, medium: 12, low: 5, none: 2 },
  avg_first_response_hours: 2.3,
  overdue_count: 4,
};

test.describe("Support Dashboard", () => {
  test.beforeEach(async ({ page }) => {
    // Mock the support metrics API endpoint.
    await page.route("**/v1/orgs/*/reports/support", (route) => {
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_SUPPORT_METRICS),
      });
    });

    // Mock the org members endpoint (used by AssigneeFilter).
    await page.route("**/v1/orgs/*/members", (route) => {
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          data: [{ user_id: "u1" }, { user_id: "u2" }],
          page_info: { has_more: false },
        }),
      });
    });
  });

  test("renders page title 'Support Tickets'", async ({ page }) => {
    await page.goto("/reports/support");
    await expect(page.getByTestId("support-title")).toBeVisible();
    await expect(page.getByTestId("support-title")).toHaveText("Support Tickets");
  });

  test("renders all 4 chart sections", async ({ page }) => {
    await page.goto("/reports/support");
    await expect(page.getByTestId("chart-section-status")).toBeVisible();
    await expect(page.getByTestId("chart-section-volume")).toBeVisible();
    await expect(page.getByTestId("chart-section-assignee")).toBeVisible();
    await expect(page.getByTestId("chart-section-priority")).toBeVisible();
  });

  test("renders 3 KPI metric cards", async ({ page }) => {
    await page.goto("/reports/support");
    const cards = page.getByTestId("metric-card");
    await expect(cards).toHaveCount(3);
  });

  test("changes date range and triggers new fetch", async ({ page }) => {
    let fetchCount = 0;
    await page.route("**/v1/orgs/*/reports/support", (route) => {
      fetchCount++;
      return route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_SUPPORT_METRICS),
      });
    });

    await page.goto("/reports/support");

    // Wait for initial data load.
    await expect(page.getByTestId("chart-section-status")).toBeVisible();
    const initialCount = fetchCount;

    // Open date picker and change from date.
    await page.getByTestId("date-range-trigger").click();
    await page.getByTestId("date-range-from").fill("2026-01-01");

    // Wait for re-fetch.
    await page.waitForTimeout(500);
    expect(fetchCount).toBeGreaterThan(initialCount);
  });

  test("click overdue tickets card navigates to thread list", async ({ page }) => {
    await page.goto("/reports/support");

    // Wait for data to load.
    await expect(page.getByTestId("metric-card-link")).toBeVisible();

    // Click the overdue metric card link.
    await page.getByTestId("metric-card-link").click();

    // Assert navigation happened.
    await page.waitForURL("**/crm?status=open&overdue=true");
    expect(page.url()).toContain("/crm?status=open&overdue=true");
  });
});
