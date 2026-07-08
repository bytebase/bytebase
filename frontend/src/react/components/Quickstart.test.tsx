import type { ReactElement, ReactNode } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  getIntroStateByKey: vi.fn(
    (key: string) => key !== "hidden" && key !== "member.visit"
  ),
  getOrFetchProjectByName: vi.fn(async () => ({ name: "" })),
  loadProjectIamPolicy: vi.fn(async () => undefined),
  saveIntroStateByKey: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/react/components/RouterLink", () => ({
  RouterLink: ({
    children,
    to,
    ...props
  }: {
    children: ReactNode;
    to: { name?: string };
  }) => (
    <a data-route-name={to.name} {...props}>
      {children}
    </a>
  ),
}));

vi.mock("@/react/stores/app", () => {
  const state = {
    quickStartEnabled: () => true,
    getIntroStateByKey: mocks.getIntroStateByKey,
    loadProjectIamPolicy: mocks.loadProjectIamPolicy,
  };
  const useAppStore = (selector?: (s: typeof state) => unknown) =>
    selector ? selector(state) : state;
  useAppStore.getState = () => ({
    ...state,
    getOrFetchProjectByName: mocks.getOrFetchProjectByName,
    fetchIssueByName: vi.fn(),
    getOrFetchWorksheetByName: vi.fn(),
    saveIntroStateByKey: mocks.saveIntroStateByKey,
  });
  return { useAppStore };
});

vi.mock("@/store", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/utils", () => ({
  extractProjectResourceName: (name: string) => name,
  hasProjectPermissionV2: () => true,
  hasWorkspacePermissionV2: () => true,
}));

vi.mock("@/types", () => ({
  isValidProjectName: () => false,
  UNKNOWN_PROJECT_NAME: "projects/-",
}));

import { Quickstart } from "./Quickstart";

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  vi.clearAllMocks();
  mocks.getIntroStateByKey.mockImplementation(
    (key: string) => key !== "hidden" && key !== "member.visit"
  );
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => {
    root.unmount();
  });
  document.body.removeChild(container);
});

const render = async (element: ReactElement) => {
  await act(async () => {
    root.render(element);
    await Promise.resolve();
  });
};

describe("Quickstart", () => {
  it("links the member visit item to the Members page", async () => {
    await render(<Quickstart />);

    const memberLink = Array.from(container.querySelectorAll("a")).find((a) =>
      a.textContent?.includes("quick-start.visit-member")
    );

    expect(memberLink?.getAttribute("data-route-name")).toBe(
      "workspace.members"
    );
  });
});
