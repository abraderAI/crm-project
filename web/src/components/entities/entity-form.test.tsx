import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { EntityForm } from "./entity-form";

describe("EntityForm", () => {
  const defaultProps = {
    mode: "create" as const,
    entityKind: "org" as const,
    onSubmit: vi.fn(),
  };

  it("renders form with title for create mode", () => {
    render(<EntityForm {...defaultProps} />);
    expect(screen.getByText("Create org")).toBeInTheDocument();
  });

  it("renders form with title for edit mode", () => {
    render(<EntityForm {...defaultProps} mode="edit" />);
    expect(screen.getByText("Edit org")).toBeInTheDocument();
  });

  it("renders name, description, and metadata fields", () => {
    render(<EntityForm {...defaultProps} />);
    expect(screen.getByTestId("entity-name-input")).toBeInTheDocument();
    expect(screen.getByTestId("entity-description-input")).toBeInTheDocument();
    expect(screen.getByTestId("entity-metadata-input")).toBeInTheDocument();
  });

  it("pre-fills initial values", () => {
    render(
      <EntityForm
        {...defaultProps}
        mode="edit"
        initialValues={{ name: "Test", description: "Desc", metadata: '{"k":"v"}' }}
      />,
    );
    expect(screen.getByTestId("entity-name-input")).toHaveValue("Test");
    expect(screen.getByTestId("entity-description-input")).toHaveValue("Desc");
    expect(screen.getByTestId("entity-metadata-input")).toHaveValue('{"k":"v"}');
  });

  it("shows type selector for space entities", () => {
    render(<EntityForm {...defaultProps} entityKind="space" />);
    expect(screen.getByTestId("entity-type-select")).toBeInTheDocument();
  });

  it("does not show type selector for org entities", () => {
    render(<EntityForm {...defaultProps} entityKind="org" />);
    expect(screen.queryByTestId("entity-type-select")).not.toBeInTheDocument();
  });

  it("does not show type selector for board entities", () => {
    render(<EntityForm {...defaultProps} entityKind="board" />);
    expect(screen.queryByTestId("entity-type-select")).not.toBeInTheDocument();
  });

  it("submits form with values", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<EntityForm {...defaultProps} onSubmit={onSubmit} />);

    await user.type(screen.getByTestId("entity-name-input"), "My Org");
    await user.type(screen.getByTestId("entity-description-input"), "A description");
    await user.click(screen.getByTestId("entity-submit-btn"));

    expect(onSubmit).toHaveBeenCalledWith({
      name: "My Org",
      description: "A description",
      metadata: "{}",
    });
  });

  it("submits space form with type", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<EntityForm {...defaultProps} entityKind="space" onSubmit={onSubmit} />);

    await user.type(screen.getByTestId("entity-name-input"), "Sales Space");
    await user.selectOptions(screen.getByTestId("entity-type-select"), "crm");
    await user.click(screen.getByTestId("entity-submit-btn"));

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ name: "Sales Space", type: "crm" }),
    );
  });

  it("validates required name field", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<EntityForm {...defaultProps} onSubmit={onSubmit} />);

    await user.click(screen.getByTestId("entity-submit-btn"));

    expect(screen.getByTestId("name-error")).toHaveTextContent("Name is required");
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("validates metadata is valid JSON", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<EntityForm {...defaultProps} onSubmit={onSubmit} />);

    await user.type(screen.getByTestId("entity-name-input"), "Test");
    await user.type(screen.getByTestId("entity-metadata-input"), "invalid json");
    await user.click(screen.getByTestId("entity-submit-btn"));

    expect(screen.getByTestId("metadata-error")).toHaveTextContent("Metadata must be valid JSON");
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("accepts valid JSON metadata", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<EntityForm {...defaultProps} onSubmit={onSubmit} />);

    await user.type(screen.getByTestId("entity-name-input"), "Test");
    // Use paste instead of type because userEvent treats { and } as special modifier keys.
    const metaInput = screen.getByTestId("entity-metadata-input");
    await user.click(metaInput);
    await user.paste('{"tier":"pro"}');
    await user.click(screen.getByTestId("entity-submit-btn"));

    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ metadata: '{"tier":"pro"}' }));
  });

  it("shows Create button in create mode", () => {
    render(<EntityForm {...defaultProps} />);
    expect(screen.getByTestId("entity-submit-btn")).toHaveTextContent("Create");
  });

  it("shows Save button in edit mode", () => {
    render(<EntityForm {...defaultProps} mode="edit" />);
    expect(screen.getByTestId("entity-submit-btn")).toHaveTextContent("Save");
  });

  it("shows Saving... when loading", () => {
    render(<EntityForm {...defaultProps} loading={true} />);
    expect(screen.getByTestId("entity-submit-btn")).toHaveTextContent("Saving...");
    expect(screen.getByTestId("entity-submit-btn")).toBeDisabled();
  });

  it("renders cancel button when onCancel provided", () => {
    render(<EntityForm {...defaultProps} onCancel={vi.fn()} />);
    expect(screen.getByTestId("entity-cancel-btn")).toBeInTheDocument();
  });

  it("does not render cancel button when onCancel not provided", () => {
    render(<EntityForm {...defaultProps} />);
    expect(screen.queryByTestId("entity-cancel-btn")).not.toBeInTheDocument();
  });

  it("calls onCancel when cancel clicked", async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    render(<EntityForm {...defaultProps} onCancel={onCancel} />);

    await user.click(screen.getByTestId("entity-cancel-btn"));
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it("trims whitespace from name on submit", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<EntityForm {...defaultProps} onSubmit={onSubmit} />);

    await user.type(screen.getByTestId("entity-name-input"), "  Test  ");
    await user.click(screen.getByTestId("entity-submit-btn"));

    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ name: "Test" }));
  });

  it("defaults metadata to {} when empty", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(<EntityForm {...defaultProps} onSubmit={onSubmit} />);

    await user.type(screen.getByTestId("entity-name-input"), "Test");
    await user.click(screen.getByTestId("entity-submit-btn"));

    expect(onSubmit).toHaveBeenCalledWith(expect.objectContaining({ metadata: "{}" }));
  });

  it("renders all space type options", () => {
    render(<EntityForm {...defaultProps} entityKind="space" />);
    const select = screen.getByTestId("entity-type-select");
    expect(select.querySelectorAll("option")).toHaveLength(5);
  });
});
