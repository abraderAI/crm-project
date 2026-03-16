import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { SystemHealthWidget } from "./system-health-widget";

const mockFetchSystemHealth = vi.fn();

vi.mock("@/lib/widget-api", () => ({
  fetchSystemHealth: (...args: unknown[]) => mockFetchSystemHealth(...args),
}));

const healthyData = {
  api_status: "healthy",
  db_status: "healthy",
  channel_health: { email: "healthy", chat: "healthy", voice: "healthy" },
  uptime: "99.9%",
};

describe("SystemHealthWidget", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state initially", () => {
    mockFetchSystemHealth.mockReturnValue(new Promise(() => {}));
    render(<SystemHealthWidget token="tok" />);
    expect(screen.getByTestId("system-health-loading")).toBeInTheDocument();
  });

  it("renders health status after loading", async () => {
    mockFetchSystemHealth.mockResolvedValue(healthyData);

    render(<SystemHealthWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("system-health-content")).toBeInTheDocument();
    });

    expect(screen.getByTestId("health-api")).toBeInTheDocument();
    expect(screen.getByTestId("health-db")).toBeInTheDocument();
    expect(screen.getByTestId("health-email")).toBeInTheDocument();
    expect(screen.getByTestId("health-chat")).toBeInTheDocument();
    expect(screen.getByTestId("health-voice")).toBeInTheDocument();
    expect(screen.getByTestId("health-uptime")).toHaveTextContent("99.9%");
  });

  it("displays status text for each service", async () => {
    mockFetchSystemHealth.mockResolvedValue({
      ...healthyData,
      db_status: "degraded",
    });

    render(<SystemHealthWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("health-db")).toHaveTextContent("degraded");
    });
  });

  it("shows error state on failure", async () => {
    mockFetchSystemHealth.mockRejectedValue(new Error("fail"));

    render(<SystemHealthWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("system-health-error")).toBeInTheDocument();
    });
  });

  it("passes token to API", async () => {
    mockFetchSystemHealth.mockResolvedValue(healthyData);

    render(<SystemHealthWidget token="my-token" />);

    await waitFor(() => {
      expect(mockFetchSystemHealth).toHaveBeenCalledWith("my-token");
    });
  });

  it("renders channel health entries dynamically", async () => {
    mockFetchSystemHealth.mockResolvedValue({
      ...healthyData,
      channel_health: { email: "healthy", chat: "down" },
    });

    render(<SystemHealthWidget token="tok" />);

    await waitFor(() => {
      expect(screen.getByTestId("health-email")).toBeInTheDocument();
      expect(screen.getByTestId("health-chat")).toHaveTextContent("down");
    });
  });
});
