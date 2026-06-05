import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
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

  test("renders project ID and title as links when a row href is provided", () => {
    act(() => {
      root.render(
        <ProjectTable
          projectList={[project]}
          getRowHref={() => "/projects/sample/issues"}
        />
      );
    });

    const links = [...container.querySelectorAll("a")];
    expect(links).toHaveLength(2);
    expect(links.map((link) => link.getAttribute("href"))).toEqual([
      "/projects/sample/issues",
      "/projects/sample/issues",
    ]);
    expect(links.map((link) => link.textContent)).toEqual([
      "sample",
      "Sample Project",
    ]);
  });

  test("keeps plain clicks on row links native without a row click handler", () => {
    act(() => {
      root.render(
        <ProjectTable projectList={[project]} getRowHref={() => "#sample"} />
      );
    });

    const links = [...container.querySelectorAll("a")];
    expect(links).toHaveLength(2);
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

  test("routes plain clicks on row links through the row click handler", () => {
    const onRowClick = vi.fn();

    act(() => {
      root.render(
        <ProjectTable
          projectList={[project]}
          getRowHref={() => "/projects/sample/issues"}
          onRowClick={onRowClick}
        />
      );
    });

    const links = [...container.querySelectorAll("a")];
    expect(links).toHaveLength(2);
    for (const link of links) {
      const event = new MouseEvent("click", {
        bubbles: true,
        cancelable: true,
      });
      const notPrevented = link.dispatchEvent(event);
      expect(notPrevented).toBe(false);
    }

    expect(onRowClick).toHaveBeenCalledTimes(2);
    expect(onRowClick.mock.calls.map((call) => call[0])).toEqual([
      project,
      project,
    ]);
  });

  test("keeps modified clicks on row links native", () => {
    const onRowClick = vi.fn();

    act(() => {
      root.render(
        <ProjectTable
          projectList={[project]}
          getRowHref={() => "#sample"}
          onRowClick={onRowClick}
        />
      );
    });

    const links = [...container.querySelectorAll("a")];
    expect(links).toHaveLength(2);
    for (const link of links) {
      const modifiedNotPrevented = link.dispatchEvent(
        new MouseEvent("click", {
          bubbles: true,
          cancelable: true,
          metaKey: true,
        })
      );
      expect(modifiedNotPrevented).toBe(true);
    }

    expect(onRowClick).not.toHaveBeenCalled();
  });
});
