import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { PlatformStats } from "./platform-stats";
import type { PlatformStats as PlatformStatsType } from "@/lib/api-types";

const fixture: PlatformStatsType = {
  orgs: { total: 150, last_7d: 12, last_30d: 45 },
  users: { total: 3200, last_7d: 88, last_30d: 340 },
  threads: { total: 9500, last_7d: 210, last_30d: 870 },
  messages: { total: 45000, last_7d: 1200, last_30d: 5600 },
  db_size_bytes: 536870912, // 512 MB
  api_uptime_pct: 99.98,
  failed_webhooks_24h: 3,
  pending_notifications: 17,
};

describe("PlatformStats", () => {
  it("renders the component container", () => {
    render(<PlatformStats stats={fixture} />);
    expect(screen.getByTestId("platform-stats")).toBeInTheDocument();
  });

  it("renders Total Orgs metric card", () => {
    render(<PlatformStats stats={fixture} />);
    expect(screen.getByText("Total Orgs")).toBeInTheDocument();
    expect(screen.getByTestId("platform-stats-orgs")).toHaveTextContent("150");
  });

  it("renders Total Users metric card", () => {
    render(<PlatformStats stats={fixture} />);
    expect(screen.getByText("Total Users")).toBeInTheDocument();
    expect(screen.getByTestId("platform-stats-users")).toHaveTextContent("3,200");
  });

  it("renders Total Threads metric card", () => {
    render(<PlatformStats stats={fixture} />);
    expect(screen.getByText("Total Threads")).toBeInTheDocument();
    expect(screen.getByTestId("platform-stats-threads")).toHaveTextContent("9,500");
  });

  it("renders DB Size metric card", () => {
    render(<PlatformStats stats={fixture} />);
    expect(screen.getByText("DB Size")).toBeInTheDocument();
    expect(screen.getByTestId("platform-stats-db-size")).toHaveTextContent("512.0 MB");
  });

  it("renders API Uptime metric card", () => {
    render(<PlatformStats stats={fixture} />);
    expect(screen.getByText("API Uptime")).toBeInTheDocument();
    expect(screen.getByTestId("platform-stats-api-uptime")).toHaveTextContent("99.98%");
  });

  it("renders all 5 metric cards", () => {
    render(<PlatformStats stats={fixture} />);
    const cards = screen.getAllByTestId("metric-card");
    expect(cards).toHaveLength(5);
  });

  it("renders loading skeletons when loading is true", () => {
    render(<PlatformStats stats={fixture} loading={true} />);
    const skeletons = screen.getAllByTestId("metric-card-skeleton");
    expect(skeletons).toHaveLength(5);
  });

  it("does not render skeletons when loading is false", () => {
    render(<PlatformStats stats={fixture} loading={false} />);
    expect(screen.queryByTestId("metric-card-skeleton")).not.toBeInTheDocument();
  });

  it("formats zero bytes correctly", () => {
    const zeroDb = { ...fixture, db_size_bytes: 0 };
    render(<PlatformStats stats={zeroDb} />);
    expect(screen.getByTestId("platform-stats-db-size")).toHaveTextContent("0 B");
  });

  it("formats bytes in KB range", () => {
    const kbDb = { ...fixture, db_size_bytes: 2048 };
    render(<PlatformStats stats={kbDb} />);
    expect(screen.getByTestId("platform-stats-db-size")).toHaveTextContent("2.0 KB");
  });

  it("formats bytes in GB range", () => {
    const gbDb = { ...fixture, db_size_bytes: 2147483648 };
    render(<PlatformStats stats={gbDb} />);
    expect(screen.getByTestId("platform-stats-db-size")).toHaveTextContent("2.0 GB");
  });

  it("formats 100% uptime", () => {
    const full = { ...fixture, api_uptime_pct: 100 };
    render(<PlatformStats stats={full} />);
    expect(screen.getByTestId("platform-stats-api-uptime")).toHaveTextContent("100%");
  });

  it("formats low uptime with decimals", () => {
    const low = { ...fixture, api_uptime_pct: 95.5 };
    render(<PlatformStats stats={low} />);
    expect(screen.getByTestId("platform-stats-api-uptime")).toHaveTextContent("95.5%");
  });

  it("renders metric cards in a grid layout", () => {
    render(<PlatformStats stats={fixture} />);
    const grid = screen.getByTestId("platform-stats");
    expect(grid.className).toContain("grid");
  });

  it("localizes large user counts", () => {
    const large = { ...fixture, users: { total: 1000000, last_7d: 0, last_30d: 0 } };
    render(<PlatformStats stats={large} />);
    expect(screen.getByTestId("platform-stats-users")).toHaveTextContent("1,000,000");
  });

  it("handles zero totals", () => {
    const zero: PlatformStatsType = {
      orgs: { total: 0, last_7d: 0, last_30d: 0 },
      users: { total: 0, last_7d: 0, last_30d: 0 },
      threads: { total: 0, last_7d: 0, last_30d: 0 },
      messages: { total: 0, last_7d: 0, last_30d: 0 },
      db_size_bytes: 0,
      api_uptime_pct: 0,
      failed_webhooks_24h: 0,
      pending_notifications: 0,
    };
    render(<PlatformStats stats={zero} />);
    expect(screen.getByTestId("platform-stats-orgs")).toHaveTextContent("0");
    expect(screen.getByTestId("platform-stats-api-uptime")).toHaveTextContent("0%");
  });
});
