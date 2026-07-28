import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  getOrFetchSettingByName: vi.fn(async () => undefined),
  getSettingByName: vi.fn(() => undefined),
  settingsByName: {},
  upsertSetting: vi.fn(async () => undefined),
}));

const t = vi.hoisted(
  () => (key: string) =>
    ({
      "bbkit.confirm-button.sure-to-delete": "Sure to delete?",
      "common.create": "Create",
      "common.deleted": "Deleted",
      "common.discard-changes": "Discard changes",
      "common.learn-more": "Learn more",
      "common.no-data": "No data",
      "common.save": "Save",
      "common.sensitive-placeholder": "Sensitive - write only",
      "common.updated": "Updated",
      "settings.im-integration.description": "IM integration description",
    })[key] ?? key
);

vi.mock("react-i18next", () => ({
  initReactI18next: {
    type: "3rdParty",
    init: vi.fn(),
  },
  useTranslation: () => ({
    t,
  }),
}));

vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: ReactElement }) => children,
}));

vi.mock("@/stores/app", () => {
  const useAppStore = <T,>(
    selector: (state: { settingsByName: typeof mocks.settingsByName }) => T
  ) =>
    selector({
      settingsByName: mocks.settingsByName,
    });
  useAppStore.getState = () => ({
    getOrFetchSettingByName: mocks.getOrFetchSettingByName,
    getSettingByName: mocks.getSettingByName,
    upsertSetting: mocks.upsertSetting,
  });
  return { useAppStore };
});

vi.mock("@/stores", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: () => true,
}));

vi.mock("@/assets/im/dingtalk.png", () => ({ default: "/dingtalk.png" }));
vi.mock("@/assets/im/feishu.webp", () => ({ default: "/feishu.webp" }));
vi.mock("@/assets/im/slack.png", () => ({ default: "/slack.png" }));
vi.mock("@/assets/im/teams.svg", () => ({ default: "/teams.svg" }));
vi.mock("@/assets/im/wecom.png", () => ({ default: "/wecom.png" }));

let IMPage: typeof import("./IMPage").IMPage;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  return {
    container,
    render: async () => {
      await act(async () => {
        root.render(element);
      });
    },
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.getOrFetchSettingByName.mockResolvedValue(undefined);
  mocks.getSettingByName.mockReturnValue(undefined);
  mocks.settingsByName = {};
  ({ IMPage } = await import("./IMPage"));
});

describe("IMPage", () => {
  test("uses shared Select for new IM integration type", async () => {
    const { container, render, unmount } = renderIntoContainer(<IMPage />);
    await render();

    await act(async () => {
      Array.from(container.querySelectorAll("button"))
        .find((button) => button.textContent?.includes("Create"))
        ?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    });

    expect(container.querySelector("select")).toBeNull();
    expect(
      Array.from(container.querySelectorAll("button")).some((button) =>
        button.textContent?.includes("Slack")
      )
    ).toBe(true);

    unmount();
  });
});
