import { flushPromises, mount } from "@vue/test-utils";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";

vi.mock("./SigninBridge", () => ({
  SigninBridge: () => <button data-testid="signin-bridge">Sign in</button>,
}));

vi.mock("@/store", () => ({
  useAuthStore: () => ({
    logout: vi.fn(),
  }),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

const mountMocks = vi.hoisted(() => ({
  changeLanguage: vi.fn(async () => {}),
  mountReactPage: vi.fn(async () => ({ unmount: vi.fn() })),
  updateReactPage: vi.fn(async () => {}),
  locale: {
    __v_isRef: true,
    value: "zh-CN",
  },
}));

vi.mock("@/react/i18n", () => ({
  default: {
    language: "en-US",
    changeLanguage: mountMocks.changeLanguage,
  },
}));

vi.mock("@/react/mount", () => ({
  mountReactPage: mountMocks.mountReactPage,
  updateReactPage: mountMocks.updateReactPage,
}));

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    locale: mountMocks.locale,
  }),
}));

vi.mock("vue-router", () => ({
  useRoute: () => ({
    fullPath: "/instances",
  }),
}));

import SessionExpiredSurfaceMount from "@/components/SessionExpiredSurfaceMount.vue";
import { SessionExpiredSurface } from "./SessionExpiredSurface";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("SessionExpiredSurface", () => {
  afterEach(() => {
    document.body.innerHTML = "";
    mountMocks.changeLanguage.mockClear();
    mountMocks.mountReactPage.mockClear();
    mountMocks.updateReactPage.mockClear();
    mountMocks.locale.value = "zh-CN";
  });

  test("mounts into the critical root", async () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(<SessionExpiredSurface currentPath="/instances" />);
      await Promise.resolve();
    });

    const criticalRoot = document.getElementById("bb-react-layer-critical");
    expect(criticalRoot).toBeInstanceOf(HTMLDivElement);
    expect(
      criticalRoot?.querySelector("[data-session-expired-surface]")
    ).toBeTruthy();

    await act(async () => {
      root.unmount();
    });
  });

  test("moves focus into the critical dialog", async () => {
    const backgroundButton = document.createElement("button");
    backgroundButton.textContent = "Background";
    document.body.appendChild(backgroundButton);
    backgroundButton.focus();

    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    await act(async () => {
      root.render(<SessionExpiredSurface currentPath="/instances" />);
      await Promise.resolve();
    });

    const criticalRoot = document.getElementById("bb-react-layer-critical");
    expect(criticalRoot).toBeInstanceOf(HTMLDivElement);

    await act(async () => {
      await vi.waitFor(() => {
        expect(document.activeElement).not.toBe(backgroundButton);
        expect(
          criticalRoot?.contains(document.activeElement as Node)
        ).toBeTruthy();
      });
    });

    await act(async () => {
      root.unmount();
    });
  });

  test("syncs React i18n before the initial mount", async () => {
    const calls: string[] = [];
    mountMocks.changeLanguage.mockImplementation(async () => {
      calls.push("changeLanguage");
    });
    mountMocks.mountReactPage.mockImplementation(async () => {
      calls.push("mountReactPage");
      return { unmount: vi.fn() };
    });

    const wrapper = mount(SessionExpiredSurfaceMount);
    await flushPromises();

    await vi.waitFor(() => {
      expect(mountMocks.changeLanguage).toHaveBeenCalledWith("zh-CN");
      expect(mountMocks.mountReactPage).toHaveBeenCalledTimes(1);
      expect(calls).toEqual(["changeLanguage", "mountReactPage"]);
    });

    wrapper.unmount();
  });
});
