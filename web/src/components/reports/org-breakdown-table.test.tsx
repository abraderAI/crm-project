import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import type { OrgSalesSummary, OrgSupportSummary } from "@/lib/reporting-types";
import { OrgBreakdownTable } from "./org-breakdown-table";

const SUPPORT_DATA: OrgSupportSummary[] = [
  {
    org_id: "o1",
    org_name: "Acme Corp",
    org_slug: "acme-corp",
    open_count: 14,
    overdue_count: 3,
    avg_resolution_hours: 12.5,
    avg_first_response_hours: 2.3,
    total_in_range: 42,
  },
  {
    org_id: "o2",
    org_name: "Beta Inc",
    org_slug: "beta-inc",
    open_count: 7,
    overdue_count: 0,
    avg_resolution_hours: null,
    avg_first_response_hours: null,
    total_in_range: 20,
  },
];

const SALES_DATA: OrgSalesSummary[] = [
  {
    org_id: "o1",
    org_name: "Acme Corp",
    org_slug: "acme-corp",
    total_leads: 50,
    win_rate: 0.35,
    avg_deal_value: 25000,
    open_pipeline_count: 12,
  },
  {
    org_id: "o2",
    org_name: "Beta Inc",
    org_slug: "beta-inc",
    total_leads: 30,
    win_rate: 0.2,
    avg_deal_value: null,
    open_pipeline_count: 8,
  },
];

describe("OrgBreakdownTable", () => {
  describe("support variant", () => {
    it("renders correct column headers", () => {
      render(<OrgBreakdownTable variant="support" data={SUPPORT_DATA} />);
      expect(screen.getByTestId("sort-header-org_name")).toHaveTextContent("Org");
      expect(screen.getByTestId("sort-header-open_count")).toHaveTextContent("Open Tickets");
      expect(screen.getByTestId("sort-header-overdue_count")).toHaveTextContent("Overdue");
      expect(screen.getByTestId("sort-header-avg_resolution_hours")).toHaveTextContent(
        "Avg Resolution",
      );
      expect(screen.getByTestId("sort-header-avg_first_response_hours")).toHaveTextContent(
        "Avg First Response",
      );
      expect(screen.getByTestId("sort-header-total_in_range")).toHaveTextContent(
        "Total (in range)",
      );
    });

    it("renders org name as link with correct href", () => {
      render(<OrgBreakdownTable variant="support" data={SUPPORT_DATA} />);
      const link = screen.getByTestId("org-link-acme-corp");
      expect(link).toHaveAttribute("href", "/orgs/acme-corp/reports/support");
      expect(link).toHaveTextContent("Acme Corp");
    });

    it("shows red badge on overdue count > 0", () => {
      render(<OrgBreakdownTable variant="support" data={SUPPORT_DATA} />);
      expect(screen.getByTestId("overdue-badge-acme-corp")).toBeInTheDocument();
      expect(screen.getByTestId("overdue-badge-acme-corp")).toHaveTextContent("3");
      // Beta Inc has 0 overdue — no badge
      expect(screen.queryByTestId("overdue-badge-beta-inc")).not.toBeInTheDocument();
    });

    it("formats avg_hours as 'X.X hrs' and null as '–'", () => {
      render(<OrgBreakdownTable variant="support" data={SUPPORT_DATA} />);
      expect(screen.getByText("12.5 hrs")).toBeInTheDocument();
      expect(screen.getByText("2.3 hrs")).toBeInTheDocument();
      // Beta Inc has null values — should show "–"
      const rows = screen.getAllByTestId("org-breakdown-row");
      // Find the Beta Inc row (sorted alphabetically, so Acme first, Beta second)
      const betaRow = rows[1]!;
      const cells = within(betaRow).getAllByRole("cell");
      // avg_resolution_hours is column index 3 (0-indexed)
      expect(cells[3]).toHaveTextContent("–");
      // avg_first_response_hours is column index 4
      expect(cells[4]).toHaveTextContent("–");
    });

    it("shows 'No data' row when data is empty", () => {
      render(<OrgBreakdownTable variant="support" data={[]} />);
      expect(screen.getByTestId("org-breakdown-empty")).toBeInTheDocument();
      expect(screen.getByTestId("org-breakdown-empty")).toHaveTextContent("No data");
    });

    it("renders correct number of rows", () => {
      render(<OrgBreakdownTable variant="support" data={SUPPORT_DATA} />);
      const rows = screen.getAllByTestId("org-breakdown-row");
      expect(rows).toHaveLength(2);
    });
  });

  describe("sales variant", () => {
    it("renders correct column headers", () => {
      render(<OrgBreakdownTable variant="sales" data={SALES_DATA} />);
      expect(screen.getByTestId("sort-header-org_name")).toHaveTextContent("Org");
      expect(screen.getByTestId("sort-header-total_leads")).toHaveTextContent("Total Leads");
      expect(screen.getByTestId("sort-header-win_rate")).toHaveTextContent("Win Rate");
      expect(screen.getByTestId("sort-header-avg_deal_value")).toHaveTextContent("Avg Deal Value");
      expect(screen.getByTestId("sort-header-open_pipeline_count")).toHaveTextContent(
        "Open Pipeline",
      );
    });

    it("renders org name as link with correct href", () => {
      render(<OrgBreakdownTable variant="sales" data={SALES_DATA} />);
      const link = screen.getByTestId("org-link-acme-corp");
      expect(link).toHaveAttribute("href", "/orgs/acme-corp/reports/sales");
    });

    it("formats win rate as percentage", () => {
      render(<OrgBreakdownTable variant="sales" data={SALES_DATA} />);
      expect(screen.getByText("35.0%")).toBeInTheDocument();
      expect(screen.getByText("20.0%")).toBeInTheDocument();
    });

    it("formats avg_deal_value as currency and null as '–'", () => {
      render(<OrgBreakdownTable variant="sales" data={SALES_DATA} />);
      expect(screen.getByText("$25,000")).toBeInTheDocument();
      // Beta Inc has null avg_deal_value
      const rows = screen.getAllByTestId("org-breakdown-row");
      const betaRow = rows[1]!;
      const cells = within(betaRow).getAllByRole("cell");
      // avg_deal_value is column index 3
      expect(cells[3]).toHaveTextContent("–");
    });

    it("shows 'No data' row when data is empty", () => {
      render(<OrgBreakdownTable variant="sales" data={[]} />);
      expect(screen.getByTestId("org-breakdown-empty")).toHaveTextContent("No data");
    });
  });

  describe("sorting", () => {
    it("clicking column header toggles asc/desc sort", async () => {
      const user = userEvent.setup();
      render(<OrgBreakdownTable variant="support" data={SUPPORT_DATA} />);

      // Default sort is org_name asc — Acme should be first
      let rows = screen.getAllByTestId("org-breakdown-row");
      expect(within(rows[0]!).getByText("Acme Corp")).toBeInTheDocument();
      expect(within(rows[1]!).getByText("Beta Inc")).toBeInTheDocument();

      // Sort icon should be asc on org_name
      expect(screen.getByTestId("sort-icon-asc")).toBeInTheDocument();

      // Click org_name again to toggle to desc
      await user.click(screen.getByTestId("sort-header-org_name"));
      rows = screen.getAllByTestId("org-breakdown-row");
      expect(within(rows[0]!).getByText("Beta Inc")).toBeInTheDocument();
      expect(within(rows[1]!).getByText("Acme Corp")).toBeInTheDocument();

      // Sort icon should be desc
      expect(screen.getByTestId("sort-icon-desc")).toBeInTheDocument();
    });

    it("data re-orders correctly when sorting by numeric column", async () => {
      const user = userEvent.setup();
      render(<OrgBreakdownTable variant="support" data={SUPPORT_DATA} />);

      // Click open_count header to sort asc
      await user.click(screen.getByTestId("sort-header-open_count"));
      let rows = screen.getAllByTestId("org-breakdown-row");
      // Beta (7) should come before Acme (14)
      expect(within(rows[0]!).getByText("Beta Inc")).toBeInTheDocument();
      expect(within(rows[1]!).getByText("Acme Corp")).toBeInTheDocument();

      // Click again to sort desc
      await user.click(screen.getByTestId("sort-header-open_count"));
      rows = screen.getAllByTestId("org-breakdown-row");
      expect(within(rows[0]!).getByText("Acme Corp")).toBeInTheDocument();
      expect(within(rows[1]!).getByText("Beta Inc")).toBeInTheDocument();
    });

    it("switching columns resets sort to asc", async () => {
      const user = userEvent.setup();
      render(<OrgBreakdownTable variant="support" data={SUPPORT_DATA} />);

      // Click org_name to toggle to desc
      await user.click(screen.getByTestId("sort-header-org_name"));
      expect(screen.getByTestId("sort-icon-desc")).toBeInTheDocument();

      // Click a different column — should reset to asc
      await user.click(screen.getByTestId("sort-header-open_count"));
      expect(screen.getByTestId("sort-icon-asc")).toBeInTheDocument();
    });
  });
});
