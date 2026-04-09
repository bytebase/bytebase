import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Revision } from "@/types/proto-es/v1/revision_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
  })),
  listRevisions: vi.fn(),
  currentUser: {
    value: {
      email: "test@example.com",
    },
  },
  createApp: vi.fn(),
  h: vi.fn((component: unknown, props: Record<string, unknown>) => ({
    component,
    props,
  })),
}));

let DatabaseRevisionPanel: typeof import("./DatabaseRevisionPanel").DatabaseRevisionPanel;

vi.mock("react-i18next", () => ({
  useTranslation: mocks.useTranslation,
}));

vi.mock("vue", async (importOriginal) => {
  const actual = await importOriginal<typeof import("vue")>();
  return {
    ...actual,
    createApp: mocks.createApp,
    h: mocks.h,
  };
});

vi.mock("@/components/Revision", () => ({
  RevisionDataTable: Symbol("RevisionDataTable"),
}));

vi.mock("@/components/Revision/CreateRevisionDrawer.vue", () => ({
  default: Symbol("CreateRevisionDrawer"),
}));

vi.mock("@/plugins/i18n", () => ({
  default: {
    install: vi.fn(),
  },
}));

vi.mock("@/plugins/naive-ui", () => ({
  default: {
    install: vi.fn(),
  },
}));

vi.mock("@/router", () => ({
  router: {
    install: vi.fn(),
    push: vi.fn(),
  },
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: (props: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button {...props} />
  ),
}));

vi.mock("@/react/legacy/mountLegacyVueApp", () => ({
  createLegacyVueApp: ({ render }: { render: () => unknown }) => ({
    mount(element: Element) {
      const vnode = render() as {
        props?: Record<string, unknown>;
      };
      if (Array.isArray(vnode.props?.revisions)) {
        const deleteButton = document.createElement("button");
        deleteButton.dataset.testid = "revision-delete";
        deleteButton.addEventListener("click", () => {
          (vnode.props?.onDelete as (() => void) | undefined)?.();
        });
        element.replaceChildren(deleteButton);
        return;
      }
      if (vnode.props?.show) {
        const drawer = document.createElement("div");
        drawer.dataset.testid = "create-revision-drawer";
        const createButton = document.createElement("button");
        createButton.dataset.testid = "create-revision-created";
        createButton.addEventListener("click", () => {
          (
            vnode.props?.onCreated as
              | ((revisions: Revision[]) => void)
              | undefined
          )?.([
            { name: "instances/inst1/databases/db1/revisions/2" } as Revision,
          ]);
        });
        drawer.appendChild(createButton);
        element.replaceChildren(drawer);
        return;
      }
      element.replaceChildren();
    },
    unmount: vi.fn(),
  }),
}));

vi.mock("@/react/hooks/useVueState", () => ({
  useVueState: <T,>(getter: () => T) => getter(),
}));

vi.mock("@/store", () => ({
  pinia: {
    install: vi.fn(),
  },
  useCurrentUserV1: () => mocks.currentUser,
}));

vi.mock("@/connect", () => ({
  revisionServiceClientConnect: {
    listRevisions: mocks.listRevisions,
  },
}));

const renderIntoContainer = (element: ReturnType<typeof createElement>) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: (nextElement = element) => {
      act(() => {
        root.render(nextElement);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

const flush = async () => {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
};

const click = (element: HTMLElement) => {
  act(() => {
    element.dispatchEvent(
      new MouseEvent("click", { bubbles: true, cancelable: true })
    );
  });
};

const makeDatabase = (): Database =>
  ({
    name: "instances/inst1/databases/db1",
    project: "projects/proj1",
  }) as Database;

beforeEach(async () => {
  mocks.useTranslation.mockReset();
  mocks.useTranslation.mockReturnValue({
    t: (key: string) => key,
  });
  mocks.listRevisions.mockReset();
  mocks.listRevisions.mockResolvedValue({
    nextPageToken: "",
    revisions: [
      {
        name: "instances/inst1/databases/db1/revisions/1",
      } as Revision,
    ],
  });

  vi.resetModules();
  ({ DatabaseRevisionPanel } = await import("./DatabaseRevisionPanel"));
});

describe("DatabaseRevisionPanel", () => {
  test("opens the revision import flow and refreshes after create", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseRevisionPanel, {
        database: makeDatabase(),
      })
    );

    render();
    await flush();

    const importButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.import"
    ) as HTMLButtonElement | undefined;
    expect(importButton).toBeDefined();

    click(importButton as HTMLButtonElement);
    await flush();

    const createButton = container.querySelector(
      '[data-testid="create-revision-created"]'
    ) as HTMLButtonElement | null;
    expect(createButton).not.toBeNull();

    click(createButton as HTMLButtonElement);
    await flush();

    expect(mocks.listRevisions).toHaveBeenCalledTimes(2);

    unmount();
  });

  test("refreshes the revision list after delete", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseRevisionPanel, {
        database: makeDatabase(),
      })
    );

    render();
    await flush();

    const deleteButton = container.querySelector(
      '[data-testid="revision-delete"]'
    ) as HTMLButtonElement | null;
    expect(deleteButton).not.toBeNull();

    click(deleteButton as HTMLButtonElement);
    await flush();

    expect(mocks.listRevisions).toHaveBeenCalledTimes(2);

    unmount();
  });
});
