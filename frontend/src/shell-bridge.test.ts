import { mount } from "@vue/test-utils";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { nextTick } from "vue";
import {
  emitReactLocaleChange,
  emitReactNotification,
  emitReactQuickstartReset,
} from "@/react/shell-bridge";

const mocks = vi.hoisted(() => ({
  locale: { value: "en-US" },
  route: {
    meta: {
      title: () => "Workspace",
    },
    query: {},
  },
  previousRoute: {
    query: {},
  },
  replace: vi.fn(),
  setDocumentTitle: vi.fn(),
  overrideAppProfile: vi.fn(),
  pushNotification: vi.fn(),
  tryPopNotification: vi.fn(),
  notificationCreate: vi.fn(),
  introState: {} as Record<string, boolean>,
  saveIntroStateByKey: vi.fn(
    async ({ key, newState }: { key: string; newState: boolean }) => {
      mocks.introState[key] = newState;
      return newState;
    }
  ),
}));

vi.mock("naive-ui", async () => {
  const { defineComponent, h } = await import("vue");
  const passthrough = (name: string) =>
    defineComponent({
      name,
      setup(_, { slots }) {
        return () => h("div", slots.default?.());
      },
    });
  return {
    NConfigProvider: passthrough("NConfigProvider"),
    NDialogProvider: passthrough("NDialogProvider"),
    NNotificationProvider: passthrough("NNotificationProvider"),
    useNotification: () => ({
      create: mocks.notificationCreate,
    }),
  };
});

vi.mock("../naive-ui.config", () => ({
  dateLang: {},
  generalLang: {},
  themeOverrides: {},
}));

vi.mock("@/react/ReactPageMount.vue", async () => {
  const { defineComponent, h } = await import("vue");
  return {
    default: defineComponent({
      name: "MockReactPageMount",
      props: ["page", "pageProps", "containerClass"],
      setup() {
        return () => h("div");
      },
    }),
  };
});

vi.mock("./AuthContext.vue", async () => {
  const { defineComponent, h } = await import("vue");
  return {
    default: defineComponent({
      name: "MockAuthContext",
      setup(_, { slots }) {
        return () => h("div", slots.default?.());
      },
    }),
  };
});

vi.mock("./components/misc/OverlayStackManager.vue", async () => {
  const { defineComponent, h } = await import("vue");
  return {
    default: defineComponent({
      name: "MockOverlayStackManager",
      setup(_, { slots }) {
        return () => h("div", slots.default?.());
      },
    }),
  };
});

vi.mock("./customAppProfile", () => ({
  overrideAppProfile: mocks.overrideAppProfile,
}));

vi.mock("./plugins/i18n", () => ({
  locale: mocks.locale,
  t: (key: string) => key,
}));

vi.mock("./store", () => ({
  useNotificationStore: () => ({
    pushNotification: mocks.pushNotification,
    tryPopNotification: mocks.tryPopNotification,
  }),
  useUIStateStore: () => ({
    introState: mocks.introState,
    saveIntroStateByKey: mocks.saveIntroStateByKey,
  }),
}));

vi.mock("./utils", () => ({
  isDev: () => false,
  setDocumentTitle: mocks.setDocumentTitle,
}));

vi.mock("vue-router", async () => {
  const actual =
    await vi.importActual<typeof import("vue-router")>("vue-router");
  const { reactive } = await import("vue");
  const route = reactive(mocks.route);
  return {
    ...actual,
    useRoute: () => route,
    useRouter: () => ({
      replace: mocks.replace,
    }),
  };
});

import App from "./App.vue";
import NotificationContext from "./NotificationContext.vue";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.locale.value = "en-US";
  mocks.route.meta.title = () => "Workspace";
  mocks.route.query = {};
  mocks.previousRoute.query = {};
  mocks.introState = {};
});

afterEach(() => {
  document.body.innerHTML = "";
});

describe("React shell bridge", () => {
  test("updates Vue locale and document title from locale events", async () => {
    const wrapper = mount(App, {
      attachTo: document.body,
      global: {
        stubs: {
          RouterView: { template: "<div />" },
        },
      },
    });
    await nextTick();

    emitReactLocaleChange("ja-JP");

    expect(mocks.locale.value).toBe("ja-JP");
    expect(mocks.setDocumentTitle).toHaveBeenCalledWith("Workspace");

    wrapper.unmount();
  });

  test("updates quickstart intro state from reset events", async () => {
    const wrapper = mount(App, {
      attachTo: document.body,
      global: {
        stubs: {
          RouterView: { template: "<div />" },
        },
      },
    });
    await nextTick();

    emitReactQuickstartReset({ keys: ["hidden", "data.query"] });
    await nextTick();

    expect(mocks.introState).toMatchObject({
      hidden: false,
      "data.query": false,
    });
    expect(mocks.saveIntroStateByKey).toHaveBeenCalledTimes(2);

    wrapper.unmount();
  });

  test("shows bytebase notifications and ignores other modules", async () => {
    const wrapper = mount(NotificationContext, {
      attachTo: document.body,
      slots: {
        default: "<div />",
      },
    });
    await nextTick();

    emitReactNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: "Saved",
    });

    expect(mocks.notificationCreate).toHaveBeenCalledWith(
      expect.objectContaining({
        type: "success",
      })
    );

    mocks.notificationCreate.mockClear();
    emitReactNotification({
      module: "other",
      style: "SUCCESS",
      title: "Ignored",
    });

    expect(mocks.notificationCreate).not.toHaveBeenCalled();

    wrapper.unmount();
  });
});
