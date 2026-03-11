import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ThreadFilters, type ThreadFilterValues } from "./thread-filters";

const defaultValues: ThreadFilterValues = {
  status: "all",
  priority: "all",
  sortBy: "newest",
  search: "",
};

describe("ThreadFilters", () => {
  it("renders all filter controls", () => {
    render(<ThreadFilters values={defaultValues} onChange={vi.fn()} />);
    expect(screen.getByTestId("thread-filters")).toBeInTheDocument();
    expect(screen.getByTestId("thread-search-input")).toBeInTheDocument();
    expect(screen.getByTestId("thread-status-filter")).toBeInTheDocument();
    expect(screen.getByTestId("thread-priority-filter")).toBeInTheDocument();
    expect(screen.getByTestId("thread-sort-select")).toBeInTheDocument();
  });

  it("shows current search value", () => {
    render(
      <ThreadFilters values={{ ...defaultValues, search: "test query" }} onChange={vi.fn()} />,
    );
    expect(screen.getByTestId("thread-search-input")).toHaveValue("test query");
  });

  it("calls onChange when search changes", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<ThreadFilters values={defaultValues} onChange={onChange} />);

    await user.type(screen.getByTestId("thread-search-input"), "b");
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ search: "b" }));
  });

  it("calls onChange when status changes", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<ThreadFilters values={defaultValues} onChange={onChange} />);

    await user.selectOptions(screen.getByTestId("thread-status-filter"), "open");
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ status: "open" }));
  });

  it("calls onChange when priority changes", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<ThreadFilters values={defaultValues} onChange={onChange} />);

    await user.selectOptions(screen.getByTestId("thread-priority-filter"), "high");
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ priority: "high" }));
  });

  it("calls onChange when sort changes", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<ThreadFilters values={defaultValues} onChange={onChange} />);

    await user.selectOptions(screen.getByTestId("thread-sort-select"), "most_votes");
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ sortBy: "most_votes" }));
  });

  it("preserves other values when one filter changes", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    const values: ThreadFilterValues = {
      status: "open",
      priority: "high",
      sortBy: "oldest",
      search: "test",
    };
    render(<ThreadFilters values={values} onChange={onChange} />);

    await user.selectOptions(screen.getByTestId("thread-sort-select"), "newest");
    expect(onChange).toHaveBeenCalledWith({
      status: "open",
      priority: "high",
      sortBy: "newest",
      search: "test",
    });
  });

  it("uses custom status options", () => {
    render(
      <ThreadFilters
        values={defaultValues}
        onChange={vi.fn()}
        statusOptions={["all", "draft", "published"]}
      />,
    );
    const select = screen.getByTestId("thread-status-filter");
    expect(select.querySelectorAll("option")).toHaveLength(3);
  });

  it("uses custom priority options", () => {
    render(
      <ThreadFilters
        values={defaultValues}
        onChange={vi.fn()}
        priorityOptions={["all", "p0", "p1"]}
      />,
    );
    const select = screen.getByTestId("thread-priority-filter");
    expect(select.querySelectorAll("option")).toHaveLength(3);
  });
});
