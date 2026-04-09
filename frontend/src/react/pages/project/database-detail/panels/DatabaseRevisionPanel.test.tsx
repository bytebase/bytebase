import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  type Revision,
  Revision_Type,
} from "@/types/proto-es/v1/revision_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  confirm: vi.fn(() => true),
  localStorage: {
    clear: vi.fn(),
    getItem: vi.fn(() => null),
    removeItem: vi.fn(),
    setItem: vi.fn(),
  },
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
  })),
  CreateRevisionDrawer: Symbol("CreateRevisionDrawer"),
  listRevisions: vi.fn(),
  deleteRevision: vi.fn(),
  currentUser: {
    value: {
      email: "test@example.com",
    },
  },
  appRenderers: [] as Array<
    () => {
      component: unknown;
      props: Record<string, unknown>;
    }
  >,
  createApp: vi.fn(),
  h: vi.fn((component: unknown, props: Record<string, unknown>) => ({
    component,
    props,
  })),
}));

let DatabaseRevisionPanel: typeof import("./DatabaseRevisionPanel").DatabaseRevisionPanel;

vi.stubGlobal("localStorage", mocks.localStorage);
vi.stubGlobal("confirm", mocks.confirm);

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
  default: mocks.CreateRevisionDrawer,
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

vi.mock("@/react/components/ui/button", () => ({
  Button: (props: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button {...props} />
  ),
}));

vi.mock("@/router", () => ({
  router: {
    install: vi.fn(),
    push: vi.fn(),
  },
}));

vi.mock("@/react/components/ui/dialog", () => ({
  Dialog: ({ open, children }: { open: boolean; children: React.ReactNode }) =>
    open ? <div data-testid="dialog-root">{children}</div> : null,
  DialogContent: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  DialogTitle: ({ children }: { children: React.ReactNode }) => (
    <h1>{children}</h1>
  ),
}));

vi.mock("@/connect", () => ({
  revisionServiceClientConnect: {
    listRevisions: mocks.listRevisions,
  },
}));

vi.mock("@/store", () => ({
  pinia: {
    install: vi.fn(),
  },
  useCurrentUserV1: () => mocks.currentUser,
  useRevisionStore: () => ({
    deleteRevision: mocks.deleteRevision,
  }),
}));

vi.mock("@/utils/v1/revision", () => ({
  getRevisionType: (type: number) => `type-${type}`,
  revisionLink: (revision: { name: string }) => `/revisions/${revision.name}`,
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
  mocks.confirm.mockReset();
  mocks.confirm.mockReturnValue(true);
  mocks.localStorage.clear.mockReset();
  mocks.localStorage.getItem.mockReset();
  mocks.localStorage.getItem.mockReturnValue(null);
  mocks.localStorage.removeItem.mockReset();
  mocks.localStorage.setItem.mockReset();
  mocks.listRevisions.mockReset();
  mocks.listRevisions.mockResolvedValue({
    nextPageToken: "",
    revisions: [
      {
        name: "instances/inst1/databases/db1/revisions/1",
        version: "1.0.0",
        type: Revision_Type.VERSIONED,
      } as Revision,
    ],
  });
  mocks.deleteRevision.mockReset();
  mocks.deleteRevision.mockResolvedValue(undefined);
  mocks.appRenderers.length = 0;
  mocks.createApp.mockReset();
  mocks.createApp.mockImplementation(
    (appDefinition: { render: () => unknown }) => {
      return {
        use() {
          return this;
        },
        mount() {
          mocks.appRenderers.push(
            appDefinition.render as () => {
              component: unknown;
              props: Record<string, unknown>;
            }
          );
        },
        unmount: vi.fn(),
      };
    }
  );

  vi.resetModules();
  ({ DatabaseRevisionPanel } = await import("./DatabaseRevisionPanel"));
});

describe("DatabaseRevisionPanel", () => {
  test("opens the legacy revision drawer and refreshes after create", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseRevisionPanel, {
        database: makeDatabase(),
      })
    );

    render();
    await flush();

    // Vue mount is lazy — not created until the drawer first opens
    expect(mocks.createApp).toHaveBeenCalledTimes(0);

    const importButton = Array.from(container.querySelectorAll("button")).find(
      (button) => button.textContent === "common.import"
    ) as HTMLButtonElement | undefined;
    expect(importButton).toBeDefined();

    click(importButton as HTMLButtonElement);
    await flush();

    // After clicking Import, the Vue drawer mount is created
    expect(mocks.createApp).toHaveBeenCalledTimes(1);

    const latestDrawerVNode = mocks.appRenderers.at(-1)?.();
    expect(latestDrawerVNode).toMatchObject({
      component: mocks.CreateRevisionDrawer,
      props: expect.objectContaining({
        database: "instances/inst1/databases/db1",
        show: true,
      }),
    });

    act(() => {
      (latestDrawerVNode?.props["onCreated"] as (() => void) | undefined)?.();
    });
    await flush();

    expect(mocks.listRevisions).toHaveBeenCalledTimes(2);
    // Vue mount stays alive after closing (lazy-mounted, never unmounted)
    expect(mocks.createApp).toHaveBeenCalledTimes(1);
    expect(mocks.appRenderers.at(-1)?.()).toMatchObject({
      component: mocks.CreateRevisionDrawer,
      props: expect.objectContaining({
        show: false,
      }),
    });

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

    expect(container.querySelector("table")).not.toBeNull();
    expect(container.textContent).toContain("1.0.0");

    const deleteButton = Array.from(container.querySelectorAll("button")).find(
      (button) =>
        button.dataset.name === "instances/inst1/databases/db1/revisions/1"
    ) as HTMLButtonElement | undefined;
    expect(deleteButton).toBeDefined();

    click(deleteButton as HTMLButtonElement);
    await flush();

    expect(mocks.confirm).toHaveBeenCalledWith(
      "database.revision.delete-confirm-dialog"
    );
    expect(mocks.deleteRevision).toHaveBeenCalledWith(
      "instances/inst1/databases/db1/revisions/1"
    );
    expect(mocks.listRevisions).toHaveBeenCalledTimes(2);

    unmount();
  });
});
