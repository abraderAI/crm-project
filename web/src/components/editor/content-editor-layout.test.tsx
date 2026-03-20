import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { ContentEditorLayout } from "./content-editor-layout";

describe("ContentEditorLayout", () => {
  it("renders header and body", () => {
    render(
      <ContentEditorLayout header={<span>Header</span>}>
        <p>Body content</p>
      </ContentEditorLayout>,
    );
    expect(screen.getByTestId("editor-header")).toHaveTextContent("Header");
    expect(screen.getByTestId("editor-body")).toHaveTextContent("Body content");
  });

  it("renders sidebar when provided", () => {
    render(
      <ContentEditorLayout header={<span>H</span>} sidebar={<div>Sidebar</div>}>
        <p>Body</p>
      </ContentEditorLayout>,
    );
    expect(screen.getByTestId("editor-sidebar")).toHaveTextContent("Sidebar");
  });

  it("omits sidebar when not provided", () => {
    render(
      <ContentEditorLayout header={<span>H</span>}>
        <p>Body</p>
      </ContentEditorLayout>,
    );
    expect(screen.queryByTestId("editor-sidebar")).toBeNull();
  });

  it("renders composer when provided", () => {
    render(
      <ContentEditorLayout header={<span>H</span>} composer={<div>Composer</div>}>
        <p>Body</p>
      </ContentEditorLayout>,
    );
    expect(screen.getByTestId("editor-composer")).toHaveTextContent("Composer");
  });

  it("omits composer when not provided", () => {
    render(
      <ContentEditorLayout header={<span>H</span>}>
        <p>Body</p>
      </ContentEditorLayout>,
    );
    expect(screen.queryByTestId("editor-composer")).toBeNull();
  });
});
