import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ThreadFilters, type ThreadFilterValues } from "./thread-filters";

const defaultValues: ThreadFilterValues = {
  sortBy: "created_at",
  sortDir: "desc",
};

describe("ThreadFilters", () => {
  it("renders the filter bar", () => {
    render(<ThreadFilters values={defaultValues} onChange={vi.fn()} />);
    expect(screen.getByTestId("thread-filters")).toBeInTheDocument();
  });

  it("renders status, priority, and assignee filters", () => {
    render(<ThreadFilters values={defaultValues} onChange={vi.fn()} />);
    expect(screen.getByTestId("filter-status")).toBeInTheDocument();
    expect(screen.getByTestId("filter-priority")).toBeInTheDocument();
    expect(screen.getByTestId("filter-assigned")).toBeInTheDocument();
  });

  it("renders sort field and direction controls", () => {
    render(<ThreadFilters values={defaultValues} onChange={vi.fn()} />);
    expect(screen.getByTestId("sort-field")).toBeInTheDocument();
    expect(screen.getByTestId("sort-direction")).toBeInTheDocument();
  });

  it("calls onChange when status filter changed", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<ThreadFilters values={defaultValues} onChange={onChange} />);

    await user.selectOptions(screen.getByTestId("filter-status"), "open");
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ status: "open" }));
  });

  it("calls onChange with undefined when status cleared", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<ThreadFilters values={{ ...defaultValues, status: "open" }} onChange={onChange} />);

    await user.selectOptions(screen.getByTestId("filter-status"), "");
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ status: undefined }));
  });

  it("calls onChange when priority filter changed", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<ThreadFilters values={defaultValues} onChange={onChange} />);

    await user.selectOptions(screen.getByTestId("filter-priority"), "high");
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ priority: "high" }));
  });

  it("calls onChange when assignee typed", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<ThreadFilters values={defaultValues} onChange={onChange} />);

    await user.type(screen.getByTestId("filter-assigned"), "a");
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ assignedTo: "a" }));
  });

  it("calls onChange when sort field changed", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<ThreadFilters values={defaultValues} onChange={onChange} />);

    await user.selectOptions(screen.getByTestId("sort-field"), "vote_score");
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ sortBy: "vote_score" }));
  });

  it("toggles sort direction on click", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<ThreadFilters values={{ ...defaultValues, sortDir: "desc" }} onChange={onChange} />);

    await user.click(screen.getByTestId("sort-direction"));
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ sortDir: "asc" }));
  });

  it("toggles sort direction from asc to desc", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<ThreadFilters values={{ ...defaultValues, sortDir: "asc" }} onChange={onChange} />);

    await user.click(screen.getByTestId("sort-direction"));
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ sortDir: "desc" }));
  });

  it("shows correct aria-label for asc direction", () => {
    render(<ThreadFilters values={{ ...defaultValues, sortDir: "asc" }} onChange={vi.fn()} />);
    expect(screen.getByLabelText("Sort ascending")).toBeInTheDocument();
  });

  it("shows correct aria-label for desc direction", () => {
    render(<ThreadFilters values={{ ...defaultValues, sortDir: "desc" }} onChange={vi.fn()} />);
    expect(screen.getByLabelText("Sort descending")).toBeInTheDocument();
  });

  it("renders custom status options", () => {
    render(
      <ThreadFilters values={defaultValues} onChange={vi.fn()} statusOptions={["new", "active"]} />,
    );
    const select = screen.getByTestId("filter-status");
    // "All statuses" + 2 custom options
    expect(select.querySelectorAll("option")).toHaveLength(3);
  });

  it("renders custom priority options", () => {
    render(
      <ThreadFilters
        values={defaultValues}
        onChange={vi.fn()}
        priorityOptions={["p0", "p1", "p2"]}
      />,
    );
    const select = screen.getByTestId("filter-priority");
    // "All priorities" + 3 custom options
    expect(select.querySelectorAll("option")).toHaveLength(4);
  });
});
