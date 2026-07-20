import type { MouseEventHandler, ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { ProjectTable } from "./ProjectTable";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/components/ui/ellipsis-text", () => ({
  EllipsisText: ({ children }: { children: React.ReactNode }) => (
    <span>{children}</span>
  ),
}));

vi.mock("@/react/components/ProjectLabel", () => ({
  ProjectLabel: ({
    children,
    className,
    link,
    onClick,
    projectName,
  }: {
    children: ReactNode;
    className?: string;
    link?: boolean;
    onClick?: MouseEventHandler<HTMLAnchorElement>;
    projectName: string;
  }) =>
    link ? (
      <a
        className={className}
        data-project-name={projectName}
        data-project-id={projectName.split("/").at(-1)}
        onClick={(e) => {
          e.stopPropagation();
          onClick?.(e);
        }}
      >
        {children}
      </a>
    ) : (
      <span className={className}>{children}</span>
    ),
}));

const project = {
  name: "projects/sample",
  title: "Sample Project",
  labels: {},
} as Project;

describe("ProjectTable", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.append(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => {
      root.unmount();
    });
    container.remove();
    vi.clearAllMocks();
  });

  test("renders project ID as text and title as a settings table link", () => {
    act(() => {
      root.render(<ProjectTable projectList={[project]} showActions />);
    });

    const links = [...container.querySelectorAll("a")];
    expect(links).toHaveLength(1);
    expect(links[0].dataset.projectName).toBe("projects/sample");
    expect(links[0].dataset.projectId).toBe("sample");
    expect(links[0].textContent).toBe("Sample Project");
    expect(container.textContent).toContain("sample");
  });

  test("keeps plain clicks on settings table links native without a row click handler", () => {
    act(() => {
      root.render(<ProjectTable projectList={[project]} showActions />);
    });

    const links = [...container.querySelectorAll("a")];
    expect(links).toHaveLength(1);
    for (const link of links) {
      const notPrevented = link.dispatchEvent(
        new MouseEvent("click", {
          bubbles: true,
          cancelable: true,
        })
      );
      expect(notPrevented).toBe(true);
    }
  });

  test("routes plain row clicks through the row click handler", () => {
    const onRowClick = vi.fn();

    act(() => {
      root.render(
        <ProjectTable
          projectList={[project]}
          showActions
          onRowClick={onRowClick}
        />
      );
    });

    expect(container.querySelectorAll("a")).toHaveLength(0);

    const row = container.querySelector("tbody tr");
    expect(row).not.toBeNull();
    const event = new MouseEvent("click", {
      bubbles: true,
      cancelable: true,
    });
    const notPrevented = row?.dispatchEvent(event);
    expect(notPrevented).toBe(true);

    expect(onRowClick).toHaveBeenCalledTimes(1);
    expect(onRowClick.mock.calls[0][0]).toBe(project);
  });

  test("passes modified row clicks through the row click handler", () => {
    const onRowClick = vi.fn();

    act(() => {
      root.render(
        <ProjectTable
          projectList={[project]}
          showActions
          onRowClick={onRowClick}
        />
      );
    });

    expect(container.querySelectorAll("a")).toHaveLength(0);

    const row = container.querySelector("tbody tr");
    expect(row).not.toBeNull();
    const modifiedNotPrevented = row?.dispatchEvent(
      new MouseEvent("click", {
        bubbles: true,
        cancelable: true,
        metaKey: true,
      })
    );
    expect(modifiedNotPrevented).toBe(true);

    expect(onRowClick).toHaveBeenCalledTimes(1);
    expect(onRowClick.mock.calls[0][0]).toBe(project);
    expect(onRowClick.mock.calls[0][1].metaKey).toBe(true);
  });

  test("renders a state column when requested", () => {
    act(() => {
      root.render(
        <ProjectTable
          projectList={[
            { ...project, state: State.ACTIVE },
            {
              name: "projects/archived",
              title: "Archived Project",
              labels: {},
              state: State.DELETED,
            } as Project,
          ]}
          showState
        />
      );
    });

    const headers = [...container.querySelectorAll("thead th")].map(
      (th) => th.textContent
    );
    expect(headers).toContain("common.state");
    expect(container.textContent).toContain("common.active");
    expect(container.textContent).toContain("common.archived");
  });
});
