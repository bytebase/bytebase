import { create } from "@bufbuild/protobuf";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useRef, useState } from "react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { getLayerRoot } from "@/components/ui/layer";
import { DEBOUNCE_SEARCH_DELAY } from "@/types/common";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { InstanceResourceSchema } from "@/types/proto-es/v1/instance_service_pb";

const mocks = vi.hoisted(() => ({
  searchAgentProjects: vi.fn(),
  searchAgentDatabases: vi.fn(),
  onSelectProject: vi.fn(),
  onSelectDatabase: vi.fn(),
  onOpenChange: vi.fn(),
  canListWorkspaceDatabases: false,
}));

vi.mock("../logic/resourceSearch", () => ({
  searchAgentProjects: mocks.searchAgentProjects,
  searchAgentDatabases: mocks.searchAgentDatabases,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: {
    hasWorkspacePermission: (permission: string) => boolean;
  }) => unknown) =>
    selector({
      hasWorkspacePermission: (permission) =>
        permission === "bb.databases.list" && mocks.canListWorkspaceDatabases,
    }),
}));

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({
    t: (key: string) =>
      ({
        "agent.resource-picker.add-context": "Add context",
        "agent.resource-picker.projects": "Projects",
        "agent.resource-picker.databases": "Databases",
        "agent.resource-picker.search-projects": "Search projects",
        "agent.resource-picker.choose-project-for-databases":
          "Choose a project to browse databases",
        "agent.resource-picker.search-databases": "Search databases",
        "agent.resource-picker.no-projects": "No projects found",
        "agent.resource-picker.no-databases": "No databases found",
        "agent.resource-picker.load-error": "Could not load projects",
        "agent.resource-picker.load-databases-error": "Could not load databases",
        "agent.resource-picker.retry": "Retry",
        "agent.resource-picker.reference-limit":
          "You can reference one project and one of its databases per message. To use both, select the project first. Ask about another resource in a separate message.",
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
const database = create(DatabaseSchema$, {
  name: "instances/main/databases/orders",
  project: project.name,
});
const secondDatabase = create(DatabaseSchema$, {
  name: "instances/reporting/databases/orders",
  project: project.name,
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
  projectName,
  selectedProjectName,
  hasSelectedReference = false,
}: {
  workspaceName?: string;
  projectName?: string;
  selectedProjectName?: string;
  hasSelectedReference?: boolean;
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
        projectName={projectName}
        selectedProjectName={selectedProjectName}
        hasSelectedReference={hasSelectedReference}
        onSelectProject={mocks.onSelectProject}
        onSelectDatabase={mocks.onSelectDatabase}
      />
    </>
  );
}

beforeEach(() => {
  mocks.searchAgentProjects.mockReset();
  mocks.searchAgentDatabases.mockReset();
  mocks.onSelectProject.mockReset();
  mocks.onSelectDatabase.mockReset();
  mocks.onOpenChange.mockReset();
  mocks.canListWorkspaceDatabases = false;
});

afterEach(() => {
  vi.useRealTimers();
});

describe("AgentResourcePicker", () => {
  test("explains the project and database limit when another reference is blocked", async () => {
    render(<PickerHarness hasSelectedReference />);

    expect(screen.getByRole("status")).toHaveTextContent(
      "You can reference one project and one of its databases per message. To use both, select the project first. Ask about another resource in a separate message."
    );
    expect(screen.queryByRole("button", { name: "Projects" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Databases" })).toBeNull();
    expect(mocks.searchAgentProjects).not.toHaveBeenCalled();
    expect(mocks.onSelectProject).not.toHaveBeenCalled();
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Close" })).toHaveFocus()
    );
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(mocks.onOpenChange).toHaveBeenCalledWith(false);
  });

  test("blocks a third reference after a project and database pair", async () => {
    render(<PickerHarness selectedProjectName={project.name} hasSelectedReference />);

    expect(screen.getByRole("status")).toHaveTextContent(
      "You can reference one project and one of its databases per message. To use both, select the project first. Ask about another resource in a separate message."
    );
    expect(screen.queryByRole("button", { name: "Projects" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Databases" })).toBeNull();
    expect(mocks.searchAgentDatabases).not.toHaveBeenCalled();
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
      expect.any(AbortSignal),
      false
    );
    await waitFor(() =>
      expect(screen.getByRole("textbox", { name: "Search projects" })).toHaveFocus()
    );

    fireEvent.click(screen.getByRole("button", { name: "Orders projects/orders" }));
    expect(mocks.onSelectProject).toHaveBeenCalledExactlyOnceWith(project);
    expect(mocks.onOpenChange).toHaveBeenCalledWith(false);
  });

  test("browses the current project's databases without choosing a project again", async () => {
    mocks.searchAgentDatabases.mockResolvedValueOnce({
      items: [database],
      nextPageToken: "",
    });
    render(<PickerHarness projectName={project.name} />);

    expect(mocks.searchAgentDatabases).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Databases" }));

    const row = await screen.findByRole("button", {
      name: /main.*orders/,
    });
    expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
      project.name,
      "",
      "",
      expect.any(AbortSignal)
    );
    expect(row).toHaveTextContent("main / orders");
    expect(row).not.toHaveTextContent(database.name);
    fireEvent.click(row);
    expect(mocks.onSelectDatabase).toHaveBeenCalledExactlyOnceWith(database);
    expect(mocks.onOpenChange).toHaveBeenCalledWith(false);
  });

  test("uses an inline project instead of chat context for database search", async () => {
    mocks.searchAgentDatabases.mockResolvedValueOnce({
      items: [database],
      nextPageToken: "",
    });
    render(
      <PickerHarness
        projectName={secondProject.name}
        selectedProjectName={project.name}
      />
    );

    expect(screen.queryByRole("button", { name: "Projects" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Databases" }));

    expect(await screen.findByRole("button", { name: /main.*orders/ })).toBeInTheDocument();
    expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
      project.name,
      "",
      "",
      expect.any(AbortSignal)
    );
  });

  test("rejects a database returned outside the selected inline project", async () => {
    mocks.searchAgentDatabases.mockResolvedValueOnce({
      items: [database],
      nextPageToken: "",
    });
    render(<PickerHarness selectedProjectName={secondProject.name} />);

    fireEvent.click(screen.getByRole("button", { name: "Databases" }));

    expect(await screen.findByText("No databases found")).toBeInTheDocument();
    expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
      secondProject.name,
      "",
      "",
      expect.any(AbortSignal)
    );
    expect(mocks.onSelectDatabase).not.toHaveBeenCalled();
  });

  test("shows instance and database names instead of resource paths", async () => {
    mocks.searchAgentDatabases.mockResolvedValueOnce({
      items: [database, secondDatabase],
      nextPageToken: "",
    });
    render(<PickerHarness projectName={project.name} />);

    fireEvent.click(screen.getByRole("button", { name: "Databases" }));

    expect(await screen.findByRole("button", { name: /main.*orders/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /reporting.*orders/ })).toBeInTheDocument();
    expect(screen.queryByText(database.name)).toBeNull();
    expect(screen.queryByText(secondDatabase.name)).toBeNull();
  });

  test("uses the unified database display with the instance title", async () => {
    const titledDatabase = create(DatabaseSchema$, {
      name: database.name,
      project: project.name,
      instanceResource: create(InstanceResourceSchema, {
        title: "Main production",
      }),
    });
    mocks.searchAgentDatabases.mockResolvedValueOnce({
      items: [titledDatabase],
      nextPageToken: "",
    });
    render(<PickerHarness projectName={project.name} />);

    fireEvent.click(screen.getByRole("button", { name: "Databases" }));

    expect(
      await screen.findByRole("button", { name: /Main production.*orders/ })
    ).toBeInTheDocument();
  });

  test("chooses a project before searching databases in workspace context", async () => {
    mocks.searchAgentProjects.mockResolvedValue({
      items: [project],
      nextPageToken: "",
    });
    mocks.searchAgentDatabases.mockResolvedValueOnce({
      items: [database],
      nextPageToken: "",
    });
    render(<PickerHarness />);

    fireEvent.click(screen.getByRole("button", { name: "Databases" }));
    expect(screen.getByText("Choose a project to browse databases")).toBeInTheDocument();
    expect(mocks.searchAgentProjects).toHaveBeenCalledWith(
      "",
      "",
      expect.any(AbortSignal),
      true
    );
    expect(mocks.searchAgentDatabases).not.toHaveBeenCalled();
    fireEvent.click(await screen.findByRole("button", {
      name: "Orders projects/orders",
    }));
    expect(mocks.onSelectProject).not.toHaveBeenCalled();
    expect(mocks.onOpenChange).not.toHaveBeenCalled();
    expect(await screen.findByRole("button", {
      name: /main.*orders/,
    })).toBeInTheDocument();
    expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
      project.name,
      "",
      "",
      expect.any(AbortSignal)
    );
    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    expect(screen.getByRole("textbox", { name: "Search projects" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    expect(screen.getByRole("button", { name: "Databases" })).toBeInTheDocument();
  });

  test("browses and selects workspace databases directly when permitted", async () => {
    mocks.canListWorkspaceDatabases = true;
    mocks.searchAgentDatabases
      .mockResolvedValueOnce({ items: [database], nextPageToken: "next" })
      .mockResolvedValueOnce({ items: [secondDatabase], nextPageToken: "" });
    render(<PickerHarness />);

    fireEvent.click(screen.getByRole("button", { name: "Databases" }));
    expect(mocks.searchAgentProjects).not.toHaveBeenCalled();
    expect(screen.queryByText("Choose a project to browse databases")).toBeNull();
    expect(await screen.findByRole("button", {
      name: /main.*orders/,
    })).toBeInTheDocument();
    expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
      "workspaces/default",
      "",
      "",
      expect.any(AbortSignal)
    );

    fireEvent.click(screen.getByRole("button", { name: "Load more" }));
    fireEvent.click(await screen.findByRole("button", {
      name: /reporting.*orders/,
    }));
    expect(mocks.searchAgentDatabases).toHaveBeenLastCalledWith(
      "workspaces/default",
      "",
      "next",
      expect.any(AbortSignal)
    );
    expect(mocks.onSelectDatabase).toHaveBeenCalledExactlyOnceWith(secondDatabase);
    expect(mocks.onOpenChange).toHaveBeenCalledWith(false);
    await waitFor(() =>
      expect(screen.queryByRole("dialog", { name: "Add context" })).toBeNull()
    );
  });

  test("does not select a workspace database with a noncanonical owner", async () => {
    mocks.canListWorkspaceDatabases = true;
    const invalidDatabase = create(DatabaseSchema$, {
      name: "instances/main/databases/invalid-owner",
      project: "orders",
    });
    mocks.searchAgentDatabases.mockResolvedValueOnce({
      items: [invalidDatabase],
      nextPageToken: "",
    });
    render(<PickerHarness />);

    fireEvent.click(screen.getByRole("button", { name: "Databases" }));
    expect(await screen.findByText("No databases found")).toBeInTheDocument();
    expect(screen.queryByRole("button", {
      name: /main.*invalid-owner/,
    })).toBeNull();
    expect(mocks.onSelectDatabase).not.toHaveBeenCalled();
    expect(mocks.onOpenChange).not.toHaveBeenCalled();
  });

  test("does not call a filtered first page empty when later databases remain", async () => {
    mocks.canListWorkspaceDatabases = true;
    const invalidDatabase = create(DatabaseSchema$, {
      name: "instances/main/databases/invalid-owner",
      project: "orders",
    });
    mocks.searchAgentDatabases
      .mockResolvedValueOnce({
        items: [invalidDatabase],
        nextPageToken: "next",
      })
      .mockResolvedValueOnce({ items: [database], nextPageToken: "" });
    render(<PickerHarness />);

    fireEvent.click(screen.getByRole("button", { name: "Databases" }));
    expect(await screen.findByRole("button", { name: "Load more" })).toBeInTheDocument();
    expect(screen.queryByText("No databases found")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Load more" }));
    expect(await screen.findByRole("button", {
      name: /main.*orders/,
    })).toBeInTheDocument();
  });

  test("leaves workspace database search if list permission is revoked", async () => {
    mocks.canListWorkspaceDatabases = true;
    let resolveOldPage!: (page: {
      items: Database[];
      nextPageToken: string;
    }) => void;
    const oldPage = new Promise<{ items: Database[]; nextPageToken: string }>(
      (resolve) => { resolveOldPage = resolve; }
    );
    mocks.searchAgentDatabases.mockReturnValueOnce(oldPage);
    mocks.searchAgentProjects.mockResolvedValueOnce({
      items: [project],
      nextPageToken: "",
    });
    const { rerender } = render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Databases" }));
    await waitFor(() => expect(mocks.searchAgentDatabases).toHaveBeenCalledTimes(1));
    const oldSignal = mocks.searchAgentDatabases.mock.calls[0][3] as AbortSignal;

    mocks.canListWorkspaceDatabases = false;
    rerender(<PickerHarness />);
    expect(oldSignal.aborted).toBe(true);
    expect(await screen.findByText("Choose a project to browse databases")).toBeInTheDocument();
    expect(await screen.findByRole("button", {
      name: "Orders projects/orders",
    })).toBeInTheDocument();
    await act(async () => {
      resolveOldPage({ items: [database], nextPageToken: "" });
      await oldPage;
    });
    expect(screen.queryByRole("button", {
      name: /main.*orders/,
    })).toBeNull();
  });

  test("searches and paginates databases remotely without dropping earlier results", async () => {
    mocks.searchAgentDatabases
      .mockResolvedValueOnce({ items: [database], nextPageToken: "" })
      .mockResolvedValueOnce({ items: [database], nextPageToken: "next" })
      .mockResolvedValueOnce({ items: [secondDatabase], nextPageToken: "" });
    render(<PickerHarness projectName={project.name} />);
    fireEvent.click(screen.getByRole("button", { name: "Databases" }));
    const search = await screen.findByRole("textbox", {
      name: "Search databases",
    });
    await screen.findByRole("button", {
      name: /main.*orders/,
    });
    fireEvent.change(search, { target: { value: "orders" } });
    await waitFor(() =>
      expect(mocks.searchAgentDatabases).toHaveBeenCalledWith(
        project.name,
        "orders",
        "",
        expect.any(AbortSignal)
      )
    );
    fireEvent.click(await screen.findByRole("button", { name: "Load more" }));
    expect(await screen.findByRole("button", {
      name: /reporting.*orders/,
    })).toBeInTheDocument();
    expect(screen.getByRole("button", {
      name: /main.*orders/,
    })).toBeInTheDocument();
    expect(mocks.searchAgentDatabases).toHaveBeenLastCalledWith(
      project.name,
      "orders",
      "next",
      expect.any(AbortSignal)
    );
  });

  test("shows a loading indicator immediately while database search is debounced", async () => {
    mocks.searchAgentDatabases
      .mockResolvedValueOnce({ items: [database], nextPageToken: "" })
      .mockResolvedValueOnce({ items: [secondDatabase], nextPageToken: "" });
    render(<PickerHarness projectName={project.name} />);
    fireEvent.click(screen.getByRole("button", { name: "Databases" }));
    await screen.findByRole("button", { name: /main.*orders/ });

    fireEvent.change(screen.getByRole("textbox", { name: "Search databases" }), {
      target: { value: "reporting" },
    });
    const loading = screen.getByRole("status");
    expect(loading).toHaveTextContent("Loading");
    expect(loading.querySelector(".animate-spin")).not.toBeNull();
    expect(screen.queryByRole("button", { name: /main.*orders/ })).toBeNull();

    await screen.findByRole("button", { name: /reporting.*orders/ });
    expect(screen.queryByRole("status")).toBeNull();
  });

  test("retries a failed database search without calling it empty", async () => {
    mocks.searchAgentDatabases
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ items: [database], nextPageToken: "" });
    render(<PickerHarness projectName={project.name} />);
    fireEvent.click(screen.getByRole("button", { name: "Databases" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Could not load databases");
    expect(screen.queryByText("No databases found")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(await screen.findByRole("button", {
      name: /main.*orders/,
    })).toBeInTheDocument();
  });

  test("selects a database with keyboard navigation", async () => {
    mocks.searchAgentDatabases.mockResolvedValue({
      items: [database, secondDatabase],
      nextPageToken: "",
    });
    render(<PickerHarness projectName={project.name} />);
    const category = screen.getByRole("button", { name: "Projects" });
    await waitFor(() => expect(category).toHaveFocus());
    fireEvent.keyDown(category, { key: "ArrowDown" });
    const databaseCategory = screen.getByRole("button", { name: "Databases" });
    expect(databaseCategory).toHaveFocus();
    fireEvent.keyDown(databaseCategory, { key: "Enter" });
    const search = await screen.findByRole("textbox", { name: "Search databases" });
    await waitFor(() => expect(search).toHaveFocus());
    await screen.findByRole("button", {
      name: /reporting.*orders/,
    });
    fireEvent.keyDown(search, { key: "ArrowDown" });
    fireEvent.keyDown(document.activeElement as HTMLElement, { key: "ArrowDown" });
    const second = screen.getByRole("button", {
      name: /reporting.*orders/,
    });
    expect(second).toHaveFocus();
    fireEvent.keyDown(second, { key: "Enter" });
    expect(mocks.onSelectDatabase).toHaveBeenCalledExactlyOnceWith(secondDatabase);
  });

  test("drops a chosen database project when the workspace changes", async () => {
    let resolveOldPage!: (page: {
      items: Database[];
      nextPageToken: string;
    }) => void;
    const oldPage = new Promise<{ items: Database[]; nextPageToken: string }>(
      (resolve) => { resolveOldPage = resolve; }
    );
    mocks.searchAgentProjects
      .mockResolvedValueOnce({ items: [project], nextPageToken: "" })
      .mockResolvedValueOnce({ items: [secondProject], nextPageToken: "" });
    mocks.searchAgentDatabases.mockReturnValueOnce(oldPage);
    const { rerender } = render(
      <PickerHarness workspaceName="workspaces/first" />
    );
    fireEvent.click(screen.getByRole("button", { name: "Databases" }));
    fireEvent.click(await screen.findByRole("button", {
      name: "Orders projects/orders",
    }));
    await waitFor(() => expect(mocks.searchAgentDatabases).toHaveBeenCalledTimes(1));
    const oldSignal = mocks.searchAgentDatabases.mock.calls[0][3] as AbortSignal;

    rerender(<PickerHarness workspaceName="workspaces/second" />);
    expect(oldSignal.aborted).toBe(true);
    expect(await screen.findByRole("button", {
      name: "Payments projects/payments",
    })).toBeInTheDocument();
    expect(screen.getByRole("textbox", {
      name: "Search projects",
    })).toBeInTheDocument();
    await act(async () => {
      resolveOldPage({ items: [database], nextPageToken: "" });
      await oldPage;
    });
    expect(screen.queryByRole("button", {
      name: /main.*orders/,
    })).toBeNull();
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
      2, "", "next", expect.any(AbortSignal), false
    );
  });

  test("shows a loading indicator while fetching the next page", async () => {
    const nextPage = deferredPage();
    mocks.searchAgentProjects
      .mockResolvedValueOnce({ items: [project], nextPageToken: "next" })
      .mockReturnValueOnce(nextPage.promise);
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    const loadMore = await screen.findByRole("button", { name: "Load more" });

    fireEvent.click(loadMore);
    const loading = screen.getByRole("button", { name: "Loading" });
    expect(loading).toBeDisabled();
    expect(loading.querySelector(".animate-spin")).not.toBeNull();
    expect(screen.getByRole("button", {
      name: "Orders projects/orders",
    })).toBeInTheDocument();

    await act(async () => {
      nextPage.resolve({ items: [secondProject], nextPageToken: "" });
      await nextPage.promise;
    });
    expect(screen.queryByRole("button", { name: "Loading" })).toBeNull();
    expect(screen.getByRole("button", {
      name: "Payments projects/payments",
    })).toBeInTheDocument();
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

  test("shows a loading indicator immediately while project search is debounced", async () => {
    mocks.searchAgentProjects
      .mockResolvedValueOnce({ items: [project], nextPageToken: "" })
      .mockResolvedValueOnce({ items: [secondProject], nextPageToken: "" });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    await screen.findByRole("button", { name: "Orders projects/orders" });

    fireEvent.change(screen.getByRole("textbox", { name: "Search projects" }), {
      target: { value: "payments" },
    });
    const loading = screen.getByRole("status");
    expect(loading).toHaveTextContent("Loading");
    expect(loading.querySelector(".animate-spin")).not.toBeNull();
    expect(screen.queryByRole("button", { name: "Orders projects/orders" })).toBeNull();

    await screen.findByRole("button", { name: "Payments projects/payments" });
    expect(screen.queryByRole("status")).toBeNull();
  });

  test("reloads results when the pending search returns to the current query", async () => {
    mocks.searchAgentProjects.mockResolvedValue({
      items: [project],
      nextPageToken: "",
    });
    render(<PickerHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Projects" }));
    await screen.findByRole("button", { name: "Orders projects/orders" });

    const search = screen.getByRole("textbox", { name: "Search projects" });
    fireEvent.change(search, { target: { value: "pay" } });
    fireEvent.change(search, { target: { value: "" } });
    expect(screen.getByRole("status")).toHaveTextContent("Loading");
    await waitFor(() => expect(mocks.searchAgentProjects).toHaveBeenCalledTimes(2), {
      timeout: DEBOUNCE_SEARCH_DELAY + 1000,
    });
    expect(await screen.findByRole("button", {
      name: "Orders projects/orders",
    })).toBeInTheDocument();
    expect(screen.queryByRole("status")).toBeNull();
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
      3, "", "next", expect.any(AbortSignal), false
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
        "pay", "", expect.any(AbortSignal), false
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
