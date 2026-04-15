import { flushPromises, mount } from "@vue/test-utils";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { nextTick } from "vue";

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
  reactI18nLanguage: { value: "en-US" },
  changeLanguage: vi.fn(async (locale: string) => {
    mountMocks.reactI18nLanguage.value = locale;
  }),
  mountReactPage: vi.fn(async () => ({ unmount: vi.fn() })),
  routePath: null as unknown,
  updateReactPage: vi.fn(async () => {}),
  locale: null as { value: string } | null,
  reactI18n: {
    get language() {
      return mountMocks.reactI18nLanguage.value;
    },
    changeLanguage: vi.fn(async (locale: string) => {
      mountMocks.reactI18nLanguage.value = locale;
    }),
  },
}));

mountMocks.reactI18n.changeLanguage = mountMocks.changeLanguage;

vi.mock("@/react/i18n", () => ({
  default: mountMocks.reactI18n,
}));

vi.mock("@/react/mount", () => ({
  mountReactPage: mountMocks.mountReactPage,
  updateReactPage: mountMocks.updateReactPage,
}));

vi.mock("vue-i18n", async () => {
  const { ref } = await import("vue");
  mountMocks.locale ||= ref("zh-CN");

  return {
    useI18n: () => ({
      locale: mountMocks.locale,
    }),
  };
});

vi.mock("vue-router", async () => {
  const { ref } = await import("vue");
  mountMocks.routePath ||= ref("/instances");

  return {
    useRoute: () => ({
      get fullPath() {
        return (
          mountMocks.routePath as {
            value: string;
          }
        ).value;
      },
    }),
  };
});

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
    mountMocks.locale!.value = "zh-CN";
    mountMocks.reactI18nLanguage.value = "en-US";
    (
      mountMocks.routePath as {
        value: string;
      }
    ).value = "/instances";
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

  test("reconciles the latest route after async mount resolves", async () => {
    let resolveMount: ((root: unknown) => void) | undefined;
    const mountedRoot = { unmount: vi.fn(() => {}) };
    mountMocks.mountReactPage.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveMount = (root) => {
            resolve(root as { unmount: typeof mountedRoot.unmount });
          };
        })
    );

    const wrapper = mount(SessionExpiredSurfaceMount);
    await flushPromises();

    expect(mountMocks.mountReactPage).toHaveBeenCalledWith(
      expect.any(HTMLDivElement),
      "SessionExpiredSurface",
      { currentPath: "/instances" }
    );

    (
      mountMocks.routePath as {
        value: string;
      }
    ).value = "/projects/demo";
    await nextTick();
    await flushPromises();

    expect(mountMocks.updateReactPage).not.toHaveBeenCalled();

    resolveMount?.(mountedRoot);
    await flushPromises();

    await vi.waitFor(() => {
      expect(mountMocks.updateReactPage).toHaveBeenCalledWith(
        mountedRoot,
        "SessionExpiredSurface",
        { currentPath: "/projects/demo" }
      );
    });

    wrapper.unmount();
  });

  test("unmounts late-mounted roots when the Vue bridge is already gone", async () => {
    let resolveMount: ((root: unknown) => void) | undefined;
    const mountedRoot = { unmount: vi.fn(() => {}) };
    mountMocks.mountReactPage.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveMount = (root) => {
            resolve(root as { unmount: typeof mountedRoot.unmount });
          };
        })
    );

    const wrapper = mount(SessionExpiredSurfaceMount);
    await flushPromises();

    wrapper.unmount();
    resolveMount?.(mountedRoot);
    await flushPromises();

    expect(mountedRoot.unmount).toHaveBeenCalledTimes(1);
  });

  test("keeps the newest route when async syncs finish out of order", async () => {
    const mountedRoot = { unmount: vi.fn(() => {}) };
    const pendingLanguageChanges: Array<() => void> = [];
    let changeLanguageCall = 0;

    mountMocks.changeLanguage.mockImplementation(async () => {
      changeLanguageCall++;
      if (changeLanguageCall === 1) {
        return;
      }
      await new Promise<void>((resolve) => {
        pendingLanguageChanges.push(resolve);
      });
    });
    mountMocks.mountReactPage.mockResolvedValue(mountedRoot);

    const wrapper = mount(SessionExpiredSurfaceMount);
    await flushPromises();

    (
      mountMocks.routePath as {
        value: string;
      }
    ).value = "/projects/first";
    await nextTick();
    await flushPromises();

    (
      mountMocks.routePath as {
        value: string;
      }
    ).value = "/projects/second";
    await nextTick();
    await flushPromises();

    expect(pendingLanguageChanges).toHaveLength(1);

    await flushPromises();
    pendingLanguageChanges[0]?.();
    await flushPromises();

    expect(pendingLanguageChanges).toHaveLength(2);

    pendingLanguageChanges[1]?.();
    await flushPromises();

    expect(mountMocks.updateReactPage).toHaveBeenLastCalledWith(
      mountedRoot,
      "SessionExpiredSurface",
      { currentPath: "/projects/second" }
    );

    wrapper.unmount();
  });

  test("keeps the newest locale when async locale syncs finish out of order", async () => {
    const mountedRoot = { unmount: vi.fn(() => {}) };
    const pendingLanguageChanges = new Map<string, () => void>();

    mountMocks.mountReactPage.mockResolvedValue(mountedRoot);
    mountMocks.changeLanguage.mockImplementation(async (locale: string) => {
      if (locale === "zh-CN") {
        mountMocks.reactI18nLanguage.value = locale;
        return;
      }
      await new Promise<void>((resolve) => {
        pendingLanguageChanges.set(locale, () => {
          mountMocks.reactI18nLanguage.value = locale;
          resolve();
        });
      });
    });

    const wrapper = mount(SessionExpiredSurfaceMount);
    await flushPromises();

    mountMocks.locale!.value = "fr-FR";
    await nextTick();
    await flushPromises();

    mountMocks.locale!.value = "de-DE";
    await nextTick();
    await flushPromises();

    expect(pendingLanguageChanges.size).toBe(1);

    pendingLanguageChanges.get("fr-FR")?.();
    await flushPromises();

    expect(pendingLanguageChanges.size).toBe(2);

    pendingLanguageChanges.get("de-DE")?.();
    await flushPromises();

    expect(mountMocks.reactI18n.language).toBe("de-DE");

    wrapper.unmount();
  });
});
