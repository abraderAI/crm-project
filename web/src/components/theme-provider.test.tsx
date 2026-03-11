import { render, renderHook, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ThemeProvider, useTheme } from "./theme-provider";

function TestConsumer(): React.ReactElement {
  const { mode, resolvedTheme, setMode, applyOrgTheme } = useTheme();
  return (
    <div>
      <span data-testid="mode">{mode}</span>
      <span data-testid="resolved">{resolvedTheme}</span>
      <button data-testid="set-dark" onClick={() => setMode("dark")}>
        Dark
      </button>
      <button data-testid="set-light" onClick={() => setMode("light")}>
        Light
      </button>
      <button data-testid="set-system" onClick={() => setMode("system")}>
        System
      </button>
      <button data-testid="apply-org" onClick={() => applyOrgTheme({ "--primary": "#ff0000" })}>
        Apply Org
      </button>
    </div>
  );
}

describe("ThemeProvider", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove("dark");
    document.documentElement.style.cssText = "";
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("defaults to system mode", () => {
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>,
    );
    expect(screen.getByTestId("mode").textContent).toBe("system");
  });

  it("resolves to light when system prefers light", () => {
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>,
    );
    expect(screen.getByTestId("resolved").textContent).toBe("light");
  });

  it("resolves to dark when system prefers dark", () => {
    vi.mocked(window.matchMedia).mockImplementation((query: string) => ({
      matches: query === "(prefers-color-scheme: dark)",
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }));

    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>,
    );
    expect(screen.getByTestId("resolved").textContent).toBe("dark");
  });

  it("switches to dark mode on click", async () => {
    const user = userEvent.setup();
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>,
    );

    await user.click(screen.getByTestId("set-dark"));
    expect(screen.getByTestId("mode").textContent).toBe("dark");
    expect(screen.getByTestId("resolved").textContent).toBe("dark");
    expect(document.documentElement.classList.contains("dark")).toBe(true);
  });

  it("switches to light mode on click", async () => {
    const user = userEvent.setup();
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>,
    );

    // First go dark, then light.
    await user.click(screen.getByTestId("set-dark"));
    await user.click(screen.getByTestId("set-light"));
    expect(screen.getByTestId("mode").textContent).toBe("light");
    expect(screen.getByTestId("resolved").textContent).toBe("light");
    expect(document.documentElement.classList.contains("dark")).toBe(false);
  });

  it("persists mode to localStorage", async () => {
    const user = userEvent.setup();
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>,
    );

    await user.click(screen.getByTestId("set-dark"));
    expect(localStorage.getItem("deft-theme")).toBe("dark");
  });

  it("restores mode from localStorage", () => {
    localStorage.setItem("deft-theme", "dark");
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>,
    );
    expect(screen.getByTestId("mode").textContent).toBe("dark");
    expect(screen.getByTestId("resolved").textContent).toBe("dark");
  });

  it("ignores invalid localStorage values", () => {
    localStorage.setItem("deft-theme", "invalid");
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>,
    );
    expect(screen.getByTestId("mode").textContent).toBe("system");
  });

  it("applies org theme overrides", async () => {
    const user = userEvent.setup();
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>,
    );

    await user.click(screen.getByTestId("apply-org"));
    expect(document.documentElement.style.getPropertyValue("--primary")).toBe("#ff0000");
  });

  it("switches back to system mode", async () => {
    const user = userEvent.setup();
    render(
      <ThemeProvider>
        <TestConsumer />
      </ThemeProvider>,
    );

    await user.click(screen.getByTestId("set-dark"));
    await user.click(screen.getByTestId("set-system"));
    expect(screen.getByTestId("mode").textContent).toBe("system");
    expect(localStorage.getItem("deft-theme")).toBe("system");
  });
});

describe("useTheme", () => {
  it("throws when used outside ThemeProvider", () => {
    expect(() => {
      renderHook(() => useTheme());
    }).toThrow("useTheme must be used within a ThemeProvider");
  });
});
