import type { MouseEventHandler, ReactElement, ReactNode } from "react";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useProjectByName: vi.fn(),
  getOrFetchProjectByName: vi.fn(),
  hasWorkspacePermissionV2: vi.fn(() => true),
  isDefaultProject: vi.fn((_name: string) => false),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/RouterLink", () => ({
  RouterLink: ({
    children,
    className,
    onClick,
    to,
  }: {
    children: ReactNode;
    className?: string;
    onClick?: MouseEventHandler<HTMLAnchorElement>;
    to: { name?: string; params?: Record<string, string> };
  }) => (
    <a
      className={className}
      data-route-name={to.name}
      data-project-id={to.params?.projectId}
      onClick={onClick}
    >
      {children}
    </a>
  ),
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: mocks.useProjectByName,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      getOrFetchProjectByName: mocks.getOrFetchProjectByName,
    }),
  },
}));

vi.mock("@/utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/utils")>()),
  hasWorkspacePermissionV2: mocks.hasWorkspacePermissionV2,
}));

vi.mock("@/types/v1/project", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/types/v1/project")>()),
  isDefaultProject: mocks.isDefaultProject,
}));

import { ProjectLabel } from "./ProjectLabel";

const render = async (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);

  await act(async () => {
    root.render(element);
    await Promise.resolve();
  });

  return { container, root };
};

describe("ProjectLabel", () => {
  let root: Root | undefined;

  beforeEach(() => {
    vi.clearAllMocks();
    mocks.hasWorkspacePermissionV2.mockReturnValue(true);
    mocks.isDefaultProject.mockReturnValue(false);
    mocks.useProjectByName.mockReturnValue({
      name: "projects/sample",
      title: "Sample Project",
    });
  });

  afterEach(() => {
    act(() => root?.unmount());
    document.body.innerHTML = "";
    root = undefined;
  });

  test("renders the project title as a label", async () => {
    const rendered = await render(<ProjectLabel projectName="projects/sample" />);
    root = rendered.root;

    expect(rendered.container.textContent).toBe("Sample Project");
    expect(rendered.container.querySelector("a")).toBeNull();
    expect(mocks.getOrFetchProjectByName).toHaveBeenCalledWith(
      "projects/sample",
      true
    );
  });

  test("preserves className when rendering a label", async () => {
    const rendered = await render(
      <ProjectLabel projectName="projects/sample" className="project-label" />
    );
    root = rendered.root;

    const label = rendered.container.querySelector(".project-label");
    expect(label?.textContent).toBe("Sample Project");
    expect(rendered.container.querySelector("a")).toBeNull();
  });

  test("renders the project title as a link", async () => {
    const rendered = await render(
      <ProjectLabel projectName="projects/sample" link />
    );
    root = rendered.root;

    const link = rendered.container.querySelector("a");
    expect(link?.textContent).toBe("Sample Project");
    expect(link?.getAttribute("data-route-name")).toBe(
      "workspace.project.detail"
    );
    expect(link?.getAttribute("data-project-id")).toBe("sample");
    expect(mocks.getOrFetchProjectByName).toHaveBeenCalledWith(
      "projects/sample",
      true
    );
  });

  test("stops linked label clicks from bubbling", async () => {
    const onParentClick = vi.fn();
    const onLabelClick = vi.fn();
    const rendered = await render(
      <div onClick={onParentClick}>
        <ProjectLabel
          projectName="projects/sample"
          link
          onClick={onLabelClick}
        />
      </div>
    );
    root = rendered.root;

    const link = rendered.container.querySelector("a");
    expect(link).not.toBeNull();
    link?.dispatchEvent(
      new MouseEvent("click", { bubbles: true, cancelable: true })
    );

    expect(onLabelClick).toHaveBeenCalledTimes(1);
    expect(onParentClick).not.toHaveBeenCalled();
  });

  test("falls back to the project id when the project is not cached", async () => {
    mocks.useProjectByName.mockReturnValue({
      name: "projects/UNKNOWN",
      title: "",
    });

    const rendered = await render(<ProjectLabel projectName="projects/sample" />);
    root = rendered.root;

    expect(rendered.container.textContent).toBe("sample");
  });

  test("does not fetch the project when custom content is provided", async () => {
    const rendered = await render(
      <ProjectLabel projectName="projects/sample">Custom Project</ProjectLabel>
    );
    root = rendered.root;

    expect(rendered.container.textContent).toBe("Custom Project");
    expect(mocks.getOrFetchProjectByName).not.toHaveBeenCalled();
  });

  test("renders unknown project names as plain text even when link is requested", async () => {
    const rendered = await render(
      <ProjectLabel projectName="projects/-1" link />
    );
    root = rendered.root;

    expect(rendered.container.querySelector("a")).toBeNull();
    expect(rendered.container.textContent).toBe("projects/-1");
    expect(mocks.getOrFetchProjectByName).not.toHaveBeenCalled();
  });

  test("renders wildcard project names as plain text even when link is requested", async () => {
    const rendered = await render(<ProjectLabel projectName="projects/-" link />);
    root = rendered.root;

    expect(rendered.container.querySelector("a")).toBeNull();
    expect(rendered.container.textContent).toBe("projects/-");
    expect(mocks.getOrFetchProjectByName).not.toHaveBeenCalled();
  });

  test("renders empty project names as plain text even when link is requested", async () => {
    const rendered = await render(<ProjectLabel projectName="" link />);
    root = rendered.root;

    expect(rendered.container.querySelector("a")).toBeNull();
    expect(rendered.container.textContent).toBe("-");
    expect(mocks.getOrFetchProjectByName).not.toHaveBeenCalled();
  });

  test("renders empty project IDs as plain text even when link is requested", async () => {
    const rendered = await render(<ProjectLabel projectName="projects/" link />);
    root = rendered.root;

    expect(rendered.container.querySelector("a")).toBeNull();
    expect(rendered.container.textContent).toBe("projects/");
    expect(mocks.getOrFetchProjectByName).not.toHaveBeenCalled();
  });

  test("renders the default project as plain text even when link is requested", async () => {
    mocks.isDefaultProject.mockImplementation(
      (name) => name === "projects/default"
    );
    mocks.useProjectByName.mockReturnValue({
      name: "projects/default",
      title: "Default project",
    });

    const rendered = await render(
      <ProjectLabel projectName="projects/default" link />
    );
    root = rendered.root;

    expect(rendered.container.querySelector("a")).toBeNull();
    expect(rendered.container.textContent).toBe("Default project");
    expect(mocks.getOrFetchProjectByName).not.toHaveBeenCalled();
  });
});
