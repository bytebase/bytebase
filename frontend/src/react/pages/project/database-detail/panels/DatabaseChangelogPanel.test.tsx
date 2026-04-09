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
  localStorage: {
    clear: vi.fn(),
    getItem: vi.fn(() => null),
    removeItem: vi.fn(),
    setItem: vi.fn(),
  },
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

vi.stubGlobal("localStorage", mocks.localStorage);

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
    push: vi.fn(),
    install: vi.fn(),
  },
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: (props: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button {...props} />
  ),
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

vi.mock("@/utils/v1/changelog", () => ({
  changelogLink: (changelog: { name: string }) =>
    `/changelogs/${changelog.name}`,
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
  mocks.localStorage.clear.mockReset();
  mocks.localStorage.getItem.mockReset();
  mocks.localStorage.getItem.mockReturnValue(null);
  mocks.localStorage.removeItem.mockReset();
  mocks.localStorage.setItem.mockReset();
  mocks.fetchChangelogList.mockReset();
  mocks.fetchChangelogList.mockResolvedValue({
    nextPageToken: "",
    changelogs: [
      {
        name: "instances/inst1/databases/db1/changelogs/1",
        planTitle: "Release 1",
      } as Changelog,
    ],
  });
  mocks.createApp.mockReset();
  mocks.createApp.mockImplementation(() => ({
    use() {
      return this;
    },
    mount() {},
    unmount: vi.fn(),
  }));

  vi.resetModules();
  ({ DatabaseChangelogPanel } = await import("./DatabaseChangelogPanel"));
});

describe("DatabaseChangelogPanel", () => {
  test("renders changelog rows in a React table", async () => {
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
    expect(container.querySelector("table")).not.toBeNull();
    expect(container.textContent).toContain(
      "instances/inst1/databases/db1/changelogs/1"
    );
    expect(container.textContent).toContain("Release 1");

    unmount();
  });
});
