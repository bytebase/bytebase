import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useEnvironment: vi.fn(),
  usePlanFeature: vi.fn(),
  useProjectByName: vi.fn(),
  getOrFetchProjectByName: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/RouterLink", () => ({
  RouterLink: ({
    children,
    className,
    rel,
    target,
  }: {
    children: React.ReactNode;
    className?: string;
    rel?: string;
    target?: string;
  }) => (
    <a className={className} target={target} rel={rel}>
      {children}
    </a>
  ),
}));

vi.mock("@/hooks/useAppState", () => ({
  useEnvironment: mocks.useEnvironment,
  usePlanFeature: mocks.usePlanFeature,
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: mocks.useProjectByName,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (
      selector: (state: { projectsByName: Record<string, unknown> }) => unknown
    ) => selector({ projectsByName: {} }),
    {
      getState: () => ({
        getOrFetchProjectByName: mocks.getOrFetchProjectByName,
      }),
    }
  ),
}));

vi.mock("@/utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/utils")>()),
  hasWorkspacePermissionV2: vi.fn(() => true),
}));

let ResourceLink: typeof import("./ResourceLink").ResourceLink;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: () =>
      act(() => {
        root.render(element);
      }),
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useEnvironment.mockReturnValue({
    name: "environments/prod",
    title: "Prod",
    color: "#123456",
    tags: { protected: "protected" },
  });
  mocks.usePlanFeature.mockReturnValue(true);
  mocks.useProjectByName.mockReturnValue({
    name: "projects/sample",
    title: "Sample project",
  });
  ({ ResourceLink } = await import("./ResourceLink"));
});

describe("ResourceLink", () => {
  test("aligns environment and project link styling", () => {
    const { container, render, unmount } = renderIntoContainer(
      <div>
        <ResourceLink resource="environments/prod" />
        <ResourceLink resource="projects/sample" />
      </div>
    );

    render();

    const links = [...container.querySelectorAll("a")];
    expect(links).toHaveLength(2);
    expect(links[0].className).toContain("normal-link");
    expect(links[0].textContent).toContain("Prod");
    expect(links[0].querySelector("svg")).toBeTruthy();
    const environmentBadge = links[0].querySelector<HTMLElement>(
      "span[style]"
    );
    expect(environmentBadge?.style.backgroundColor).toBe(
      "rgba(18, 52, 86, 0.1)"
    );
    expect(environmentBadge?.style.color).toBe("rgb(18, 52, 86)");
    expect(links[1].className).toContain("normal-link");
    expect(links[1].textContent).toContain("Sample project");

    unmount();
  });

  test("can hide the resource type label for inline project links", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ResourceLink resource="projects/sample" showResourceType={false} />
    );

    render();

    const link = container.querySelector("a");
    expect(link?.textContent).toBe("Sample project");
    expect(link?.textContent).not.toContain("common.project");

    unmount();
  });

  test("keeps resource type labels outside resource anchors", () => {
    const { container, render, unmount } = renderIntoContainer(
      <div>
        <ResourceLink resource="environments/prod" />
        <ResourceLink resource="projects/sample" />
      </div>
    );

    render();

    const links = [...container.querySelectorAll("a")];
    expect(links).toHaveLength(2);
    expect(links[0].textContent).toBe("Prod");
    expect(links[1].textContent).toBe("Sample project");
    expect(container.textContent).toContain("common.environment:");
    expect(container.textContent).toContain("common.project:");

    unmount();
  });

  test("passes through link attributes for inline project links", () => {
    const { container, render, unmount } = renderIntoContainer(
      <ResourceLink
        resource="projects/sample"
        showResourceType={false}
        className="underline underline-offset-2"
        target="_blank"
        rel="noopener noreferrer"
      />
    );

    render();

    const link = container.querySelector("a");
    expect(link?.className).toContain("normal-link");
    expect(link?.className).toContain("underline underline-offset-2");
    expect(link?.getAttribute("target")).toBe("_blank");
    expect(link?.getAttribute("rel")).toBe("noopener noreferrer");

    unmount();
  });
});
