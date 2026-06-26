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

vi.mock("@/react/components/RouterLink", () => ({
  RouterLink: ({
    children,
    className,
  }: {
    children: React.ReactNode;
    className?: string;
  }) => <a className={className}>{children}</a>,
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useEnvironment: mocks.useEnvironment,
  usePlanFeature: mocks.usePlanFeature,
}));

vi.mock("@/react/hooks/useProjectByName", () => ({
  useProjectByName: mocks.useProjectByName,
}));

vi.mock("@/react/stores/app", () => ({
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
    tags: { protected: "protected" },
  });
  mocks.usePlanFeature.mockReturnValue(true);
  mocks.useProjectByName.mockReturnValue({ title: "Sample project" });
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
    expect(links[1].className).toContain("normal-link");
    expect(links[1].textContent).toContain("Sample project");

    unmount();
  });
});
