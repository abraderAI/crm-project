import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { HomeLayoutEditor } from "./home-layout-editor";
import type { WidgetRegistry } from "./home-layout";
import type { WidgetConfig } from "@/lib/tier-types";

const testRegistry: WidgetRegistry = {
  a: { title: "Widget A", render: () => <span>A</span> },
  b: { title: "Widget B", render: () => <span>B</span> },
  c: { title: "Widget C", render: () => <span>C</span> },
};

function makeLayout(): WidgetConfig[] {
  return [
    { widget_id: "a", visible: true },
    { widget_id: "b", visible: true },
    { widget_id: "c", visible: true },
  ];
}

describe("HomeLayoutEditor", () => {
  const mockSave = vi.fn();
  const mockReset = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    mockSave.mockResolvedValue(undefined);
    mockReset.mockResolvedValue(undefined);
  });

  it("renders all widgets in the editor list", () => {
    render(
      <HomeLayoutEditor
        layout={makeLayout()}
        registry={testRegistry}
        onSave={mockSave}
        onReset={mockReset}
        isCustomized={false}
      />,
    );

    expect(screen.getByTestId("home-layout-editor")).toBeInTheDocument();
    expect(screen.getByTestId("editor-item-a")).toBeInTheDocument();
    expect(screen.getByTestId("editor-item-b")).toBeInTheDocument();
    expect(screen.getByTestId("editor-item-c")).toBeInTheDocument();
  });

  it("toggles widget visibility on click", async () => {
    const user = userEvent.setup();
    render(
      <HomeLayoutEditor
        layout={makeLayout()}
        registry={testRegistry}
        onSave={mockSave}
        onReset={mockReset}
        isCustomized={false}
      />,
    );

    const toggleA = screen.getByTestId("toggle-a");
    expect(toggleA).toHaveAttribute("aria-label", "Hide Widget A");

    await user.click(toggleA);
    expect(toggleA).toHaveAttribute("aria-label", "Show Widget A");

    // Toggle back.
    await user.click(toggleA);
    expect(toggleA).toHaveAttribute("aria-label", "Hide Widget A");
  });

  it("moves a widget up", async () => {
    const user = userEvent.setup();
    render(
      <HomeLayoutEditor
        layout={makeLayout()}
        registry={testRegistry}
        onSave={mockSave}
        onReset={mockReset}
        isCustomized={false}
      />,
    );

    await user.click(screen.getByTestId("move-up-b"));

    // Verify order after move: b, a, c.
    const items = screen.getByTestId("editor-widget-list").querySelectorAll("li");
    expect(items[0]).toHaveAttribute("data-testid", "editor-item-b");
    expect(items[1]).toHaveAttribute("data-testid", "editor-item-a");
    expect(items[2]).toHaveAttribute("data-testid", "editor-item-c");
  });

  it("moves a widget down", async () => {
    const user = userEvent.setup();
    render(
      <HomeLayoutEditor
        layout={makeLayout()}
        registry={testRegistry}
        onSave={mockSave}
        onReset={mockReset}
        isCustomized={false}
      />,
    );

    await user.click(screen.getByTestId("move-down-a"));

    // Verify order after move: b, a, c.
    const items = screen.getByTestId("editor-widget-list").querySelectorAll("li");
    expect(items[0]).toHaveAttribute("data-testid", "editor-item-b");
    expect(items[1]).toHaveAttribute("data-testid", "editor-item-a");
  });

  it("disables move-up for first item", () => {
    render(
      <HomeLayoutEditor
        layout={makeLayout()}
        registry={testRegistry}
        onSave={mockSave}
        onReset={mockReset}
        isCustomized={false}
      />,
    );

    expect(screen.getByTestId("move-up-a")).toBeDisabled();
  });

  it("disables move-down for last item", () => {
    render(
      <HomeLayoutEditor
        layout={makeLayout()}
        registry={testRegistry}
        onSave={mockSave}
        onReset={mockReset}
        isCustomized={false}
      />,
    );

    expect(screen.getByTestId("move-down-c")).toBeDisabled();
  });

  it("calls onSave with current layout on save click", async () => {
    const user = userEvent.setup();
    render(
      <HomeLayoutEditor
        layout={makeLayout()}
        registry={testRegistry}
        onSave={mockSave}
        onReset={mockReset}
        isCustomized={false}
      />,
    );

    await user.click(screen.getByTestId("editor-save"));

    await waitFor(() => {
      expect(mockSave).toHaveBeenCalledTimes(1);
    });
    expect(mockSave).toHaveBeenCalledWith(makeLayout());
  });

  it("calls onSave with modified layout after toggle and reorder", async () => {
    const user = userEvent.setup();
    render(
      <HomeLayoutEditor
        layout={makeLayout()}
        registry={testRegistry}
        onSave={mockSave}
        onReset={mockReset}
        isCustomized={false}
      />,
    );

    // Toggle b visibility off, then move c up.
    await user.click(screen.getByTestId("toggle-b"));
    await user.click(screen.getByTestId("move-up-c"));
    await user.click(screen.getByTestId("editor-save"));

    await waitFor(() => {
      expect(mockSave).toHaveBeenCalledTimes(1);
    });

    const savedLayout = mockSave.mock.calls[0]?.[0] as WidgetConfig[];
    expect(savedLayout).toHaveLength(3);
    // b should be hidden, c should have moved before b.
    const bItem = savedLayout.find((w) => w.widget_id === "b");
    expect(bItem?.visible).toBe(false);
  });

  it("shows reset button only when customized", () => {
    const { rerender } = render(
      <HomeLayoutEditor
        layout={makeLayout()}
        registry={testRegistry}
        onSave={mockSave}
        onReset={mockReset}
        isCustomized={false}
      />,
    );

    expect(screen.queryByTestId("editor-reset")).not.toBeInTheDocument();

    rerender(
      <HomeLayoutEditor
        layout={makeLayout()}
        registry={testRegistry}
        onSave={mockSave}
        onReset={mockReset}
        isCustomized={true}
      />,
    );

    expect(screen.getByTestId("editor-reset")).toBeInTheDocument();
  });

  it("calls onReset on reset click", async () => {
    const user = userEvent.setup();
    render(
      <HomeLayoutEditor
        layout={makeLayout()}
        registry={testRegistry}
        onSave={mockSave}
        onReset={mockReset}
        isCustomized={true}
      />,
    );

    await user.click(screen.getByTestId("editor-reset"));

    await waitFor(() => {
      expect(mockReset).toHaveBeenCalledTimes(1);
    });
  });

  it("has accessible aria-labels for toggle buttons", () => {
    render(
      <HomeLayoutEditor
        layout={[
          { widget_id: "a", visible: true },
          { widget_id: "b", visible: false },
        ]}
        registry={testRegistry}
        onSave={mockSave}
        onReset={mockReset}
        isCustomized={false}
      />,
    );

    expect(screen.getByTestId("toggle-a")).toHaveAttribute("aria-label", "Hide Widget A");
    expect(screen.getByTestId("toggle-b")).toHaveAttribute("aria-label", "Show Widget B");
  });

  it("has accessible aria-labels for move buttons", () => {
    render(
      <HomeLayoutEditor
        layout={makeLayout()}
        registry={testRegistry}
        onSave={mockSave}
        onReset={mockReset}
        isCustomized={false}
      />,
    );

    expect(screen.getByTestId("move-up-b")).toHaveAttribute("aria-label", "Move Widget B up");
    expect(screen.getByTestId("move-down-b")).toHaveAttribute("aria-label", "Move Widget B down");
  });
});
