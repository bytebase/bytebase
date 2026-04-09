import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type {
  Changelog,
  Database,
} from "@/types/proto-es/v1/database_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({
    t: (key: string) => key,
  })),
  fetchChangelogList: vi.fn(),
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

let DatabaseChangelogPanel: typeof import("./DatabaseChangelogPanel").DatabaseChangelogPanel;

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

vi.mock("@/components/Changelog", () => ({
  ChangelogDataTable: Symbol("ChangelogDataTable"),
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
        props?: {
          changelogs?: Changelog[];
        };
      };
      element.innerHTML = (vnode.props?.changelogs ?? [])
        .map((changelog) => `<div>${changelog.name}</div>`)
        .join("");
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
  useChangelogStore: () => ({
    fetchChangelogList: mocks.fetchChangelogList,
  }),
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
  mocks.fetchChangelogList.mockReset();
  mocks.fetchChangelogList.mockResolvedValue({
    nextPageToken: "",
    changelogs: [
      {
        name: "instances/inst1/databases/db1/changelogs/1",
      } as Changelog,
    ],
  });

  vi.resetModules();
  ({ DatabaseChangelogPanel } = await import("./DatabaseChangelogPanel"));
});

describe("DatabaseChangelogPanel", () => {
  test("loads changelog rows through the paged fetcher", async () => {
    const { container, render, unmount } = renderIntoContainer(
      createElement(DatabaseChangelogPanel, {
        database: makeDatabase(),
      })
    );

    render();
    await flush();

    expect(mocks.fetchChangelogList).toHaveBeenCalledWith({
      parent: "instances/inst1/databases/db1",
      pageSize: expect.any(Number),
      pageToken: "",
    });
    expect(container.textContent).toContain(
      "instances/inst1/databases/db1/changelogs/1"
    );

    unmount();
  });
});
