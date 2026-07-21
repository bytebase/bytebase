import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  getWorkspaceProfile: vi.fn(() => ({
    announcement: {
      text: "Bytebase now supports custom themes!",
      link: "bytebase.com",
    },
  })),
  updateWorkspaceProfile: vi.fn(async () => undefined),
}));

const t = vi.hoisted(
  () => (key: string) =>
    ({
      "common.preview": "Preview",
      "settings.general.workspace.announcement-alert-level.field.critical":
        "Critical",
      "settings.general.workspace.announcement-alert-level.field.info":
        "Normal",
      "settings.general.workspace.announcement-alert-level.field.warning":
        "Warning",
      "settings.general.workspace.announcement-text.description":
        "To hide the announcement, leave it empty.",
      "settings.general.workspace.announcement-text.placeholder":
        "Bytebase now supports custom themes!",
      "settings.general.workspace.announcement-text.self": "Title",
      "settings.general.workspace.announcement-theme.background": "Background",
      "settings.general.workspace.announcement-theme.custom": "Custom",
      "settings.general.workspace.announcement-theme.description":
        "Choose a built-in theme or define custom colors for the announcement banner.",
      "settings.general.workspace.announcement-theme.self": "Theme",
      "settings.general.workspace.announcement-theme.text": "Text",
      "settings.general.workspace.extra-link.placeholder": "bytebase.com",
      "settings.general.workspace.extra-link.self": "Link",
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

vi.mock("@/components/FeatureBadge", () => ({
  FeatureBadge: () => null,
}));

vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: ReactElement }) => children,
  usePermissionCheck: () => [true],
}));

vi.mock("@/hooks/useAppState", () => ({
  usePlanFeature: () => true,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      getWorkspaceProfile: mocks.getWorkspaceProfile,
      updateWorkspaceProfile: mocks.updateWorkspaceProfile,
    }),
  },
}));

vi.mock("@/utils", () => ({
  colorToHex: () => "#4f46e5",
  hexToColor: (hex: string) => ({ hex }),
}));

let AnnouncementSection: typeof import("./AnnouncementSection").AnnouncementSection;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  act(() => {
    root.render(element);
  });
  return {
    container,
    unmount: () =>
      act(() => {
        root.unmount();
        container.remove();
      }),
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  ({ AnnouncementSection } = await import("./AnnouncementSection"));
});

describe("AnnouncementSection", () => {
  test("keeps announcement sections as peers for consistent field group spacing", () => {
    const { container, unmount } = renderIntoContainer(
      <AnnouncementSection title="Announcement" onDirtyChange={() => {}} />
    );

    const fieldGroup = container.querySelector(
      '[data-slot="form-field-group"]'
    );
    const sections = Array.from(fieldGroup?.children ?? []).filter(
      (element) => element.getAttribute("data-slot") === "form-field"
    );

    expect(sections.map((section) => section.textContent)).toEqual([
      expect.stringContaining("Theme"),
      expect.stringContaining("Preview"),
      expect.stringContaining("Title"),
      expect.stringContaining("Link"),
    ]);

    unmount();
  });
});
