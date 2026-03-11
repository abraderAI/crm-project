import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it } from "vitest";
import { ThemeProvider } from "./theme-provider";
import { ThemeToggle } from "./theme-toggle";

function renderToggle(): ReturnType<typeof render> {
  return render(
    <ThemeProvider>
      <ThemeToggle />
    </ThemeProvider>,
  );
}

describe("ThemeToggle", () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.classList.remove("dark");
  });

  it("renders a toggle button", () => {
    renderToggle();
    const button = screen.getByTestId("theme-toggle");
    expect(button).toBeInTheDocument();
  });

  it("has accessible label with current mode", () => {
    renderToggle();
    const button = screen.getByTestId("theme-toggle");
    expect(button.getAttribute("aria-label")).toContain("system");
  });

  it("cycles from system to light to dark", async () => {
    const user = userEvent.setup();
    renderToggle();
    const button = screen.getByTestId("theme-toggle");

    // system -> dark
    // Note: the initial mode is "system" by default (index 2), so next is light (index 0)
    // Actually: MODES = [light(0), dark(1), system(2)], default is system (index 2)
    // So cycling: (2+1) % 3 = 0 -> light
    await user.click(button);
    expect(button.getAttribute("aria-label")).toContain("light");

    // light -> dark
    await user.click(button);
    expect(button.getAttribute("aria-label")).toContain("dark");

    // dark -> system
    await user.click(button);
    expect(button.getAttribute("aria-label")).toContain("system");
  });

  it("persists mode changes to localStorage", async () => {
    const user = userEvent.setup();
    renderToggle();
    const button = screen.getByTestId("theme-toggle");

    await user.click(button); // system -> light
    expect(localStorage.getItem("deft-theme")).toBe("light");

    await user.click(button); // light -> dark
    expect(localStorage.getItem("deft-theme")).toBe("dark");
  });
});
