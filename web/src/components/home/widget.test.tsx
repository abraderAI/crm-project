import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Widget } from "./widget";

describe("Widget", () => {
  it("renders with title and content when visible", () => {
    render(
      <Widget id="test" title="Test Widget">
        <p>Widget content</p>
      </Widget>,
    );

    expect(screen.getByTestId("widget-test")).toBeInTheDocument();
    expect(screen.getByTestId("widget-title-test")).toHaveTextContent("Test Widget");
    expect(screen.getByTestId("widget-content-test")).toHaveTextContent("Widget content");
  });

  it("renders nothing when visible is false", () => {
    render(
      <Widget id="hidden" title="Hidden" visible={false}>
        <p>Should not appear</p>
      </Widget>,
    );

    expect(screen.queryByTestId("widget-hidden")).not.toBeInTheDocument();
  });

  it("defaults to visible when visible prop is omitted", () => {
    render(
      <Widget id="default" title="Default">
        Content
      </Widget>,
    );

    expect(screen.getByTestId("widget-default")).toBeInTheDocument();
  });

  it("sets data-widget-id attribute", () => {
    render(
      <Widget id="my-widget" title="My Widget">
        Content
      </Widget>,
    );

    expect(screen.getByTestId("widget-my-widget")).toHaveAttribute("data-widget-id", "my-widget");
  });

  it("applies additional className", () => {
    render(
      <Widget id="styled" title="Styled" className="extra-class">
        Content
      </Widget>,
    );

    expect(screen.getByTestId("widget-styled")).toHaveClass("extra-class");
  });

  it("renders different children content", () => {
    render(
      <Widget id="multi" title="Multi">
        <ul>
          <li>Item 1</li>
          <li>Item 2</li>
        </ul>
      </Widget>,
    );

    expect(screen.getByText("Item 1")).toBeInTheDocument();
    expect(screen.getByText("Item 2")).toBeInTheDocument();
  });
});
