import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useUnsavedChangesGuard: vi.fn(),
  instanceFormContext: {
    basicInfo: { engine: 0 },
    state: { isRequesting: false },
    valueChanged: false,
  },
}));

let CreateInstancePage: typeof import("./CreateInstancePage").CreateInstancePage;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/react/hooks/useUnsavedChangesGuard", () => ({
  useUnsavedChangesGuard: mocks.useUnsavedChangesGuard,
}));

vi.mock("@/react/router", () => ({
  router: {
    push: vi.fn(),
  },
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      instanceCountLimit: () => 10,
      activatedInstanceCount: () => 0,
    }),
  },
}));

vi.mock("@/store", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/react/components/instance", () => ({
  InfoPanel: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  InfoPanelContent: () => <div />,
  InstanceFormBody: () => <div />,
  InstanceFormButtons: () => <div />,
  InstanceFormProvider: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  useInstanceFormContext: () => mocks.instanceFormContext,
}));

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.instanceFormContext.state.isRequesting = false;
  mocks.instanceFormContext.valueChanged = false;
  globalThis.ResizeObserver = class ResizeObserver {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as typeof ResizeObserver;
  ({ CreateInstancePage } = await import("./CreateInstancePage"));
});

describe("CreateInstancePage", () => {
  test("guards navigation when the create form has unsaved changes", () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(<CreateInstancePage />);
    });

    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(false);

    mocks.instanceFormContext.valueChanged = true;
    act(() => {
      root.render(<CreateInstancePage />);
    });

    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(true);

    mocks.instanceFormContext.state.isRequesting = true;
    act(() => {
      root.render(<CreateInstancePage />);
    });

    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(false);

    act(() => {
      root.unmount();
    });
  });
});
