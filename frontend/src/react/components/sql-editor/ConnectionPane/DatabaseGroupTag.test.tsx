import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  fetchDBGroup: vi.fn().mockResolvedValue(undefined),
  getDBGroupByName: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/stores/app", () => {
  const state = {
    fetchDBGroup: mocks.fetchDBGroup,
    // Proxy so `dbGroupsByName[name]` resolves through the per-test
    // `getDBGroupByName` mock (preserves the existing test bodies).
    dbGroupsByName: new Proxy(
      {},
      { get: (_t, prop: string) => mocks.getDBGroupByName(prop) }
    ),
  };
  return {
    useAppStore: Object.assign(
      (selector: (s: typeof state) => unknown) => selector(state),
      { getState: () => state }
    ),
  };
});

vi.mock("@/types/proto-es/v1/database_group_service_pb", () => ({
  DatabaseGroupView: { FULL: 2 },
}));

vi.mock("@/react/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

let DatabaseGroupTag: typeof import("./DatabaseGroupTag").DatabaseGroupTag;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  ({ DatabaseGroupTag } = await import("./DatabaseGroupTag"));
});

describe("DatabaseGroupTag", () => {
  test("renders nothing while the group is not yet resolved", () => {
    mocks.getDBGroupByName.mockReturnValue({ name: "", title: "" });
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseGroupTag
        databaseGroupName="projects/p/databaseGroups/g"
        onUncheck={vi.fn()}
      />
    );
    render();
    expect(container.textContent).toBe("");
    unmount();
  });

  test("renders group title + close button once resolved", () => {
    mocks.getDBGroupByName.mockReturnValue({
      name: "projects/p/databaseGroups/g",
      title: "My Group",
    });
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseGroupTag
        databaseGroupName="projects/p/databaseGroups/g"
        onUncheck={vi.fn()}
      />
    );
    render();
    expect(container.textContent).toContain("My Group");
    expect(
      container.querySelector("button[aria-label='common.close']")
    ).not.toBeNull();
    unmount();
  });

  test("close button fires onUncheck with the group name", () => {
    mocks.getDBGroupByName.mockReturnValue({
      name: "projects/p/databaseGroups/g",
      title: "G",
    });
    const onUncheck = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseGroupTag
        databaseGroupName="projects/p/databaseGroups/g"
        onUncheck={onUncheck}
      />
    );
    render();
    const btn = container.querySelector(
      "button[aria-label='common.close']"
    ) as HTMLButtonElement;
    act(() => {
      btn.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    expect(onUncheck).toHaveBeenCalledWith("projects/p/databaseGroups/g");
    unmount();
  });

  test("close button is disabled when disabled prop is set", () => {
    mocks.getDBGroupByName.mockReturnValue({
      name: "projects/p/databaseGroups/g",
      title: "G",
    });
    const onUncheck = vi.fn();
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseGroupTag
        databaseGroupName="projects/p/databaseGroups/g"
        disabled
        onUncheck={onUncheck}
      />
    );
    render();
    const btn = container.querySelector(
      "button[aria-label='common.close']"
    ) as HTMLButtonElement;
    expect(btn.disabled).toBe(true);
    act(() => {
      btn.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });
    expect(onUncheck).not.toHaveBeenCalled();
    unmount();
  });
});
