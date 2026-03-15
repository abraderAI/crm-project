import { test, expect } from "@playwright/test";

/** Mock admin support metrics response. */
const MOCK_ADMIN_SUPPORT_METRICS = {
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
  org_breakdown: [
    {
      org_id: "o1",
      org_name: "Acme Corp",
      org_slug: "acme-corp",
      open_count: 10,
      overdue_count: 2,
      avg_resolution_hours: 10.0,
      avg_first_response_hours: 1.5,
      total_in_range: 30,
    },
    {
      org_id: "o2",
      org_name: "Beta Inc",
      org_slug: "beta-inc",
      open_count: 4,
      overdue_count: 2,
      avg_resolution_hours: 15.0,
      avg_first_response_hours: 3.1,
      total_in_range: 12,
    },
  ],
};

/** Mock admin sales metrics response. */
const MOCK_ADMIN_SALES_METRICS = {
  pipeline_funnel: [
    { stage: "new_lead", count: 10 },
    { stage: "contacted", count: 8 },
    { stage: "qualified", count: 5 },
    { stage: "closed_won", count: 3 },
    { stage: "closed_lost", count: 2 },
  ],
  lead_velocity: [
    { date: "2026-03-01", count: 5 },
    { date: "2026-03-02", count: 8 },
    { date: "2026-03-03", count: 3 },
  ],
  win_rate: 0.35,
  loss_rate: 0.15,
  avg_deal_value: 25000,
  leads_by_assignee: [
    { user_id: "u1", name: "Alice", count: 12 },
    { user_id: "u2", name: "Bob", count: 8 },
  ],
  score_distribution: [
    { range: "0-20", count: 5 },
    { range: "20-40", count: 10 },
    { range: "40-60", count: 15 },
    { range: "60-80", count: 8 },
    { range: "80-100", count: 3 },
  ],
  stage_conversion_rates: [
    { from_stage: "new_lead", to_stage: "contacted", rate: 0.75 },
    { from_stage: "contacted", to_stage: "qualified", rate: 0.6 },
  ],
  avg_time_in_stage: [
    { stage: "new_lead", avg_hours: 24 },
    { stage: "contacted", avg_hours: 48 },
  ],
  org_breakdown: [
    {
      org_id: "o1",
      org_name: "Acme Corp",
      org_slug: "acme-corp",
      total_leads: 35,
      win_rate: 0.4,
      avg_deal_value: 30000,
      open_pipeline_count: 8,
    },
    {
      org_id: "o2",
      org_name: "Beta Inc",
      org_slug: "beta-inc",
      total_leads: 15,
      win_rate: 0.25,
      avg_deal_value: 20000,
      open_pipeline_count: 4,
    },
  ],
};

test.describe("Admin Reporting: Support Dashboard", () => {
  test.beforeEach(async ({ page }) => {
    // Mock the admin support metrics API endpoint.
    await page.route("**/v1/admin/reports/support", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_ADMIN_SUPPORT_METRICS),
      });
    });
  });

  test("renders 'Platform Support Overview' heading", async ({ page }) => {
    await page.goto("/admin/reports/support");
    await expect(page.getByTestId("admin-support-title")).toBeVisible();
    await expect(page.getByTestId("admin-support-title")).toHaveText("Platform Support Overview");
  });

  test("org breakdown table renders with ≥1 row", async ({ page }) => {
    await page.goto("/admin/reports/support");
    await expect(page.getByTestId("org-breakdown-table")).toBeVisible();
    const rows = page.getByTestId("org-breakdown-row");
    await expect(rows.first()).toBeVisible();
    expect(await rows.count()).toBeGreaterThanOrEqual(1);
  });

  test("org name is a clickable link", async ({ page }) => {
    await page.goto("/admin/reports/support");
    const orgLink = page.getByTestId("org-link-acme-corp");
    await expect(orgLink).toBeVisible();
    await expect(orgLink).toHaveAttribute("href", "/orgs/acme-corp/reports/support");
  });

  test("click org link navigates to org-scoped report", async ({ page }) => {
    await page.goto("/admin/reports/support");
    const orgLink = page.getByTestId("org-link-acme-corp");
    await expect(orgLink).toBeVisible();
    await orgLink.click();
    await page.waitForURL("**/orgs/acme-corp/reports/support", { timeout: 5000 }).catch(() => {
      // Navigation may not fully complete in mocked environment.
    });
    expect(page.url()).toContain("/orgs/acme-corp/reports/support");
  });
});

test.describe("Admin Reporting: Sales Dashboard", () => {
  test.beforeEach(async ({ page }) => {
    // Mock the admin sales metrics API endpoint.
    await page.route("**/v1/admin/reports/sales", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_ADMIN_SALES_METRICS),
      });
    });
  });

  test("renders 'Platform Sales Overview' heading", async ({ page }) => {
    await page.goto("/admin/reports/sales");
    await expect(page.getByTestId("admin-sales-title")).toBeVisible();
    await expect(page.getByTestId("admin-sales-title")).toHaveText("Platform Sales Overview");
  });

  test("KPI cards (Total Leads, Win Rate, Avg Deal Value) are visible", async ({ page }) => {
    await page.goto("/admin/reports/sales");
    await expect(page.getByText("Total Leads")).toBeVisible();
    await expect(page.getByText("Platform Win Rate")).toBeVisible();
    await expect(page.getByText("Avg Deal Value")).toBeVisible();
  });

  test("org breakdown table renders with ≥1 row", async ({ page }) => {
    await page.goto("/admin/reports/sales");
    await expect(page.getByTestId("org-breakdown-table")).toBeVisible();
    const rows = page.getByTestId("org-breakdown-row");
    await expect(rows.first()).toBeVisible();
    expect(await rows.count()).toBeGreaterThanOrEqual(1);
  });
});
