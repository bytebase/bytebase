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
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/components/RouterLink", () => ({
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

vi.mock("@/react/hooks/useProjectByName", () => ({
  useProjectByName: mocks.useProjectByName,
}));

vi.mock("@/react/stores/app", () => ({
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
});
