import { create } from "@bufbuild/protobuf";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useRef, useState } from "react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { getLayerRoot } from "@/components/ui/layer";
import { DEBOUNCE_SEARCH_DELAY } from "@/types/common";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";

const mocks = vi.hoisted(() => ({
  searchAgentProjects: vi.fn(),
  onSelectProject: vi.fn(),
  onOpenChange: vi.fn(),
}));

vi.mock("../logic/resourceSearch", () => ({
  searchAgentProjects: mocks.searchAgentProjects,
}));

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({
    t: (key: string) =>
      ({
        "agent.resource-picker.add-context": "Add context",
        "agent.resource-picker.projects": "Projects",
        "agent.resource-picker.search-projects": "Search projects",
        "agent.resource-picker.no-projects": "No projects found",
        "agent.resource-picker.load-error": "Could not load projects",
        "agent.resource-picker.retry": "Retry",
        "agent.resource-picker.one-reference-per-message":
          "Only one reference per message. Send a separate question for another resource.",
        "common.back": "Back",
        "common.close": "Close",
        "common.loading": "Loading",
        "common.load-more": "Load more",
      })[key] ?? key,
  }),
}));

import { AgentResourcePicker } from "./AgentResourcePicker";

const project = create(ProjectSchema, {
  name: "projects/orders",
  title: "Orders",
});
const secondProject = create(ProjectSchema, {
  name: "projects/payments",
  title: "Payments",
});

function deferredPage() {
  let resolve!: (page: { items: Project[]; nextPageToken: string }) => void;
  const promise = new Promise<{ items: Project[]; nextPageToken: string }>(
    (complete) => { resolve = complete; }
  );
  return { promise, resolve };
}

function PickerHarness({
  workspaceName = "workspaces/default",
  hasSelectedReference = false,
} = {}) {
  const anchorRef = useRef<HTMLTextAreaElement>(null);
  const [open, setOpen] = useState(true);
  return (
    <>
      <textarea ref={anchorRef} aria-label="Question" />
      <AgentResourcePicker
        open={open}
        onOpenChange={(nextOpen) => {
          mocks.onOpenChange(nextOpen);
          setOpen(nextOpen);
        }}
        anchorRef={anchorRef}
        workspaceName={workspaceName}
        hasSelectedReference={hasSelectedReference}
        onSelectProject={mocks.onSelectProject}
      />
    </>
  );
}

beforeEach(() => {
  mocks.searchAgentProjects.mockReset();
  mocks.onSelectProject.mockReset();
  mocks.onOpenChange.mockReset();
});

afterEach(() => {
  vi.useRealTimers();
});

describe("AgentResourcePicker", () => {
  test("stops a second reference and asks for a separate question", async () => {
    render(<PickerHarness hasSelectedReference />);

    expect(screen.getByRole("status")).toHaveTextContent(
      "Only one reference per message. Send a separate question for another resource."
    );
    expect(screen.queryByRole("button", { name: "Projects" })).toBeNull();
    expect(mocks.searchAgentProjects).not.toHaveBeenCalled();
    expect(mocks.onSelectProject).not.toHaveBeenCalled();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Close" })).toHaveFocus()
    );
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(mocks.onOpenChange).toHaveBeenCalledWith(false);
  });

  test("browses projects without a request until the category is chosen", async () => {
    mocks.searchAgentProjects.mockResolvedValueOnce({
      items: [project],
      nextPageToken: "",
    });
    render(<PickerHarness />);

    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Projects" })).toHaveFocus()
    );
    expect(mocks.searchAgentProjects).not.toHaveBeenCalled();
    expect(
      getLayerRoot("agent").querySelector("[data-agent-resource-picker]")
    ).not.toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Orders projects/orders" })).toBeInTheDocument()
    );
    expect(mocks.searchAgentProjects).toHaveBeenCalledWith(
      "",
      "",
      expect.any(AbortSignal)
    );
    await waitFor(() =>
      expect(screen.getByRole("textbox", { name: "Search projects" })).toHaveFocus()
    );

    fireEvent.click(screen.getByRole("button", { name: "Orders projects/orders" }));
    expect(mocks.onSelectProject).toHaveBeenCalledExactlyOnceWith(project);
    expect(mocks.onOpenChange).toHaveBeenCalledWith(false);
  });

  test("loads another page without dropping the first page", async () => {
    mocks.searchAgentProjects
      .mockResolvedValueOnce({ items: [project], nextPageToken: "next" })
      .mockResolvedValueOnce({ items: [secondProject], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));

    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Orders projects/orders" })).toBeInTheDocument()
    );
    fireEvent.click(screen.getByRole("button", { name: "Load more" }));

    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Payments projects/payments" })).toBeInTheDocument()
    );
    expect(screen.getByRole("button", { name: "Orders projects/orders" })).toBeInTheDocument();
    expect(mocks.searchAgentProjects).toHaveBeenNthCalledWith(
      2, "", "next", expect.any(AbortSignal)
    );
  });

  test("shows each project's canonical resource ID beneath its title", async () => {
    const idLikeTitle = create(ProjectSchema, {
      name: "projects/actual-id",
      title: "projects/looks-like-an-id",
    });
    const matchingTitle = create(ProjectSchema, {
      name: "projects/same",
      title: "projects/same",
    });
    mocks.searchAgentProjects.mockResolvedValueOnce({
      items: [project, idLikeTitle, matchingTitle],
      nextPageToken: "",
    });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));

    for (const item of [project, idLikeTitle, matchingTitle]) {
      const row = await screen.findByRole("button", {
        name: `${item.title} ${item.name}`,
      });
      const titleAndId = row.querySelector("span.flex-col");
      expect(titleAndId).toHaveClass("flex-col");
      expect(titleAndId?.children).toHaveLength(2);
      expect(titleAndId?.children[0]).toHaveTextContent(item.title);
      expect(titleAndId?.children[1]).toHaveTextContent(item.name);
    }
  });

  test("disambiguates duplicate project titles after loading another page", async () => {
    const first = create(ProjectSchema, {
      name: "projects/new-one",
      title: "New Project",
    });
    const second = create(ProjectSchema, {
      name: "projects/new-two",
      title: "New Project",
    });
    mocks.searchAgentProjects
      .mockResolvedValueOnce({
        items: [first, secondProject],
        nextPageToken: "next",
      })
      .mockResolvedValueOnce({ items: [second], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));

    const firstRow = await screen.findByRole("button", { name: "New Project projects/new-one" });
    expect(firstRow).toHaveTextContent(first.name);
    fireEvent.click(screen.getByRole("button", { name: "Load more" }));

    await waitFor(() =>
      expect(
        document.querySelector('[data-agent-resource-project="projects/new-two"]')
      ).not.toBeNull()
    );
    expect(
      document.querySelector('[data-agent-resource-project="projects/new-one"]')
    ).toHaveTextContent(first.name);
    expect(
      document.querySelector('[data-agent-resource-project="projects/new-two"]')
    ).toHaveTextContent(second.name);
    expect(screen.getByRole("button", { name: "Payments projects/payments" })).toHaveTextContent(
      secondProject.name
    );
    fireEvent.click(
      document.querySelector<HTMLButtonElement>(
        '[data-agent-resource-project="projects/new-two"]'
      )!
    );
    expect(mocks.onSelectProject).toHaveBeenCalledExactlyOnceWith(second);
  });

  test("shows a retryable error instead of an empty result", async () => {
    mocks.searchAgentProjects
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ items: [project], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent("Could not load projects")
    );
    expect(screen.queryByText("No projects found")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Orders projects/orders" })).toBeInTheDocument()
    );
  });

  test("distinguishes an empty project search from a failed search", async () => {
    mocks.searchAgentProjects.mockResolvedValueOnce({ items: [], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    await waitFor(() =>
      expect(screen.getByText("No projects found")).toBeInTheDocument()
    );
    expect(screen.queryByRole("alert")).toBeNull();
  });

  test("retries a failed continuation without replacing earlier results", async () => {
    mocks.searchAgentProjects
      .mockResolvedValueOnce({ items: [project], nextPageToken: "next" })
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ items: [secondProject], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Load more" })).toBeInTheDocument()
    );
    fireEvent.click(screen.getByRole("button", { name: "Load more" }));
    await waitFor(() => expect(screen.getByRole("alert")).toBeInTheDocument());
    expect(screen.getByRole("button", { name: "Orders projects/orders" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Payments projects/payments" })).toBeInTheDocument()
    );
    expect(mocks.searchAgentProjects).toHaveBeenNthCalledWith(
      3, "", "next", expect.any(AbortSignal)
    );
  });

  test("keeps the newest query result when an older request finishes late", async () => {
    const oldPage = deferredPage();
    const newPage = deferredPage();
    mocks.searchAgentProjects
      .mockReturnValueOnce(oldPage.promise)
      .mockReturnValueOnce(newPage.promise);
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    const search = screen.getByRole("textbox", { name: "Search projects" });
    fireEvent.change(search, { target: { value: "pay" } });
    await waitFor(() =>
      expect(mocks.searchAgentProjects).toHaveBeenCalledWith(
        "pay", "", expect.any(AbortSignal)
      ),
      { timeout: DEBOUNCE_SEARCH_DELAY + 1000 }
    );

    await act(async () => {
      newPage.resolve({ items: [secondProject], nextPageToken: "" });
      await newPage.promise;
    });
    expect(screen.getByRole("button", { name: "Payments projects/payments" })).toBeInTheDocument();

    await act(async () => {
      oldPage.resolve({ items: [project], nextPageToken: "old-next" });
      await oldPage.promise;
    });
    expect(screen.queryByRole("button", { name: "Orders projects/orders" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Load more" })).toBeNull();
  });

  test("does not show results from a previous workspace", async () => {
    const oldWorkspacePage = deferredPage();
    mocks.searchAgentProjects
      .mockReturnValueOnce(oldWorkspacePage.promise)
      .mockResolvedValueOnce({ items: [secondProject], nextPageToken: "" });
    const { rerender } = render(<PickerHarness workspaceName="workspaces/first" />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    const oldSignal = mocks.searchAgentProjects.mock.calls[0][2] as AbortSignal;

    rerender(<PickerHarness workspaceName="workspaces/second" />);
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Payments projects/payments" })).toBeInTheDocument()
    );
    expect(oldSignal.aborted).toBe(true);
    await act(async () => {
      oldWorkspacePage.resolve({ items: [project], nextPageToken: "" });
      await oldWorkspacePage.promise;
    });
    expect(screen.queryByRole("button", { name: "Orders projects/orders" })).toBeNull();
  });

  test("supports keyboard selection, back navigation, and Escape", async () => {
    mocks.searchAgentProjects.mockResolvedValue({ items: [project, secondProject], nextPageToken: "" });
    render(<PickerHarness />);
    const category = screen.getByRole("button", { name: "Projects" });
    await waitFor(() => expect(category).toHaveFocus());
    fireEvent.keyDown(category, { key: "Enter" });
    const search = await screen.findByRole("textbox", { name: "Search projects" });
    await waitFor(() => expect(search).toHaveFocus());

    fireEvent.keyDown(search, { key: "ArrowDown" });
    const firstProject = screen.getByRole("button", { name: "Orders projects/orders" });
    expect(firstProject).toHaveFocus();
    expect(firstProject).toHaveClass("focus-visible:bg-control-bg");
    expect(firstProject).toHaveClass("focus-visible:ring-inset");
    expect(firstProject).toHaveClass("focus-visible:ring-offset-0");
    fireEvent.keyDown(document.activeElement as HTMLElement, { key: "ArrowDown" });
    const secondProjectButton = screen.getByRole("button", { name: "Payments projects/payments" });
    expect(secondProjectButton).toHaveFocus();
    expect(secondProjectButton).toHaveClass("focus-visible:bg-control-bg");
    fireEvent.keyDown(document.activeElement as HTMLElement, { key: "ArrowUp" });
    expect(screen.getByRole("button", { name: "Orders projects/orders" })).toHaveFocus();

    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    expect(screen.getByRole("button", { name: "Projects" })).toBeInTheDocument();
    fireEvent.keyDown(screen.getByRole("button", { name: "Projects" }), { key: "Escape" });
    expect(mocks.onOpenChange).toHaveBeenCalledWith(false);
    expect(mocks.onSelectProject).not.toHaveBeenCalled();
  });

  test("selects the first search result with Enter", async () => {
    mocks.searchAgentProjects.mockResolvedValue({ items: [project], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Orders projects/orders" })).toBeInTheDocument()
    );
    fireEvent.keyDown(screen.getByRole("textbox", { name: "Search projects" }), { key: "Enter" });
    expect(mocks.onSelectProject).toHaveBeenCalledExactlyOnceWith(project);
  });

  test("does not select stale results before a new query is sent", async () => {
    mocks.searchAgentProjects
      .mockResolvedValueOnce({ items: [project], nextPageToken: "" })
      .mockResolvedValueOnce({ items: [secondProject], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Orders projects/orders" })).toBeInTheDocument()
    );

    const search = screen.getByRole("textbox", { name: "Search projects" });
    fireEvent.change(search, { target: { value: "pay" } });
    fireEvent.keyDown(search, { key: "Enter" });
    expect(mocks.onSelectProject).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: "Orders projects/orders" })).toBeNull();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Payments projects/payments" })).toBeInTheDocument()
    );
  });

  test("selects a focused result with Enter but not during IME composition", async () => {
    mocks.searchAgentProjects.mockResolvedValue({ items: [project], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    const result = await screen.findByRole("button", { name: "Orders projects/orders" });
    const search = screen.getByRole("textbox", { name: "Search projects" });
    fireEvent.keyDown(search, { key: "Enter", isComposing: true });
    expect(mocks.onSelectProject).not.toHaveBeenCalled();
    result.focus();
    fireEvent.keyDown(result, { key: "Enter" });
    expect(mocks.onSelectProject).toHaveBeenCalledExactlyOnceWith(project);
  });
});
