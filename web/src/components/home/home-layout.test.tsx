import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { HomeLayout, type WidgetRegistry } from "./home-layout";
import type { WidgetConfig } from "@/lib/tier-types";

/** Test registry with widgets a through i (9 widgets). */
const testRegistry: WidgetRegistry = {
  a: { title: "Widget A", render: () => <span>Content A</span> },
  b: { title: "Widget B", render: () => <span>Content B</span> },
  c: { title: "Widget C", render: () => <span>Content C</span> },
  d: { title: "Widget D", render: () => <span>Content D</span> },
  e: { title: "Widget E", render: () => <span>Content E</span> },
  f: { title: "Widget F", render: () => <span>Content F</span> },
  g: { title: "Widget G", render: () => <span>Content G</span> },
  h: { title: "Widget H", render: () => <span>Content H</span> },
  i: { title: "Widget I", render: () => <span>Content I</span> },
};

function makeLayout(ids: string[], allVisible = true): WidgetConfig[] {
  return ids.map((id) => ({ widget_id: id, visible: allVisible }));
}

describe("HomeLayout", () => {
  it("renders a single widget", () => {
    render(<HomeLayout layout={makeLayout(["a"])} registry={testRegistry} />);
    expect(screen.getByTestId("home-layout")).toBeInTheDocument();
    expect(screen.getByTestId("widget-a")).toBeInTheDocument();
    expect(screen.getByText("Content A")).toBeInTheDocument();
  });

  it("renders multiple widgets in order", () => {
    render(<HomeLayout layout={makeLayout(["c", "a", "b"])} registry={testRegistry} />);

    const grid = screen.getByTestId("home-layout");
    const widgets = grid.querySelectorAll("[data-widget-id]");
    expect(widgets).toHaveLength(3);
    expect(widgets[0]).toHaveAttribute("data-widget-id", "c");
    expect(widgets[1]).toHaveAttribute("data-widget-id", "a");
    expect(widgets[2]).toHaveAttribute("data-widget-id", "b");
  });

  it("renders 9 widgets correctly", () => {
    render(
      <HomeLayout
        layout={makeLayout(["a", "b", "c", "d", "e", "f", "g", "h", "i"])}
        registry={testRegistry}
      />,
    );

    const grid = screen.getByTestId("home-layout");
    const widgets = grid.querySelectorAll("[data-widget-id]");
    expect(widgets).toHaveLength(9);
  });

  it("hides widgets with visible=false", () => {
    const layout: WidgetConfig[] = [
      { widget_id: "a", visible: true },
      { widget_id: "b", visible: false },
      { widget_id: "c", visible: true },
    ];
    render(<HomeLayout layout={layout} registry={testRegistry} />);

    expect(screen.getByTestId("widget-a")).toBeInTheDocument();
    expect(screen.queryByTestId("widget-b")).not.toBeInTheDocument();
    expect(screen.getByTestId("widget-c")).toBeInTheDocument();
  });

  it("shows empty state when all widgets are hidden", () => {
    render(<HomeLayout layout={makeLayout(["a", "b"], false)} registry={testRegistry} />);
    expect(screen.getByTestId("home-layout-empty")).toBeInTheDocument();
    expect(screen.getByText("No widgets to display.")).toBeInTheDocument();
  });

  it("shows empty state for empty layout", () => {
    render(<HomeLayout layout={[]} registry={testRegistry} />);
    expect(screen.getByTestId("home-layout-empty")).toBeInTheDocument();
  });

  it("skips widgets not in registry", () => {
    const layout: WidgetConfig[] = [
      { widget_id: "a", visible: true },
      { widget_id: "unknown", visible: true },
      { widget_id: "b", visible: true },
    ];
    render(<HomeLayout layout={layout} registry={testRegistry} />);

    expect(screen.getByTestId("widget-a")).toBeInTheDocument();
    expect(screen.queryByTestId("widget-unknown")).not.toBeInTheDocument();
    expect(screen.getByTestId("widget-b")).toBeInTheDocument();
  });

  it("applies responsive grid classes", () => {
    render(<HomeLayout layout={makeLayout(["a"])} registry={testRegistry} />);
    const grid = screen.getByTestId("home-layout");
    expect(grid).toHaveClass("grid");
    expect(grid).toHaveClass("gap-4");
  });

  it("applies additional className", () => {
    render(
      <HomeLayout layout={makeLayout(["a"])} registry={testRegistry} className="custom-class" />,
    );
    expect(screen.getByTestId("home-layout")).toHaveClass("custom-class");
  });
});
