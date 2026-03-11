import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { MetadataEditor, entriesToRecord, recordToEntries } from "./metadata-editor";

describe("recordToEntries", () => {
  it("converts record to entries", () => {
    expect(recordToEntries({ name: "test", count: 5 })).toEqual([
      { key: "name", value: "test" },
      { key: "count", value: "5" },
    ]);
  });

  it("handles empty record", () => {
    expect(recordToEntries({})).toEqual([]);
  });

  it("stringifies non-string values", () => {
    const entries = recordToEntries({ nested: { a: 1 } });
    expect(entries[0]?.value).toBe('{"a":1}');
  });
});

describe("entriesToRecord", () => {
  it("converts entries to record", () => {
    expect(
      entriesToRecord([
        { key: "name", value: "test" },
        { key: "count", value: "5" },
      ]),
    ).toEqual({ name: "test", count: "5" });
  });

  it("trims keys", () => {
    expect(entriesToRecord([{ key: "  name  ", value: "test" }])).toEqual({ name: "test" });
  });

  it("skips empty keys", () => {
    expect(
      entriesToRecord([
        { key: "", value: "orphan" },
        { key: "valid", value: "ok" },
      ]),
    ).toEqual({
      valid: "ok",
    });
  });

  it("skips whitespace-only keys", () => {
    expect(entriesToRecord([{ key: "   ", value: "orphan" }])).toEqual({});
  });
});

describe("MetadataEditor", () => {
  it("renders empty state", () => {
    render(<MetadataEditor entries={[]} onChange={vi.fn()} />);
    expect(screen.getByTestId("metadata-editor")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-empty")).toBeInTheDocument();
  });

  it("renders existing entries", () => {
    render(
      <MetadataEditor
        entries={[
          { key: "tier", value: "pro" },
          { key: "status", value: "active" },
        ]}
        onChange={vi.fn()}
      />,
    );
    expect(screen.getByTestId("metadata-row-0")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-row-1")).toBeInTheDocument();
    expect(screen.getByTestId("metadata-key-0")).toHaveValue("tier");
    expect(screen.getByTestId("metadata-value-0")).toHaveValue("pro");
  });

  it("adds a new entry when add button clicked", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<MetadataEditor entries={[]} onChange={onChange} />);

    await user.click(screen.getByTestId("metadata-add-btn"));
    expect(screen.getByTestId("metadata-row-0")).toBeInTheDocument();
    expect(onChange).toHaveBeenCalledWith([{ key: "", value: "" }]);
  });

  it("removes an entry when remove button clicked", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<MetadataEditor entries={[{ key: "tier", value: "pro" }]} onChange={onChange} />);

    await user.click(screen.getByTestId("metadata-remove-0"));
    expect(screen.queryByTestId("metadata-row-0")).not.toBeInTheDocument();
    expect(onChange).toHaveBeenCalledWith([]);
  });

  it("updates key value", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<MetadataEditor entries={[{ key: "tier", value: "pro" }]} onChange={onChange} />);

    const keyInput = screen.getByTestId("metadata-key-0");
    await user.clear(keyInput);
    await user.type(keyInput, "plan");
    const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1];
    expect(lastCall?.[0]).toEqual([{ key: "plan", value: "pro" }]);
  });

  it("updates field value", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<MetadataEditor entries={[{ key: "tier", value: "pro" }]} onChange={onChange} />);

    const valueInput = screen.getByTestId("metadata-value-0");
    await user.clear(valueInput);
    await user.type(valueInput, "enterprise");
    const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1];
    expect(lastCall?.[0]).toEqual([{ key: "tier", value: "enterprise" }]);
  });

  it("disables inputs when disabled", () => {
    render(
      <MetadataEditor
        entries={[{ key: "tier", value: "pro" }]}
        onChange={vi.fn()}
        disabled={true}
      />,
    );
    expect(screen.getByTestId("metadata-key-0")).toBeDisabled();
    expect(screen.getByTestId("metadata-value-0")).toBeDisabled();
    expect(screen.getByTestId("metadata-remove-0")).toBeDisabled();
    expect(screen.getByTestId("metadata-add-btn")).toBeDisabled();
  });
});
