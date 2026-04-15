import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";

const bridgeMocks = vi.hoisted(() => {
  type RootComponent = { render?: () => unknown };
  const app = {
    mount: vi.fn(),
    unmount: vi.fn(),
    use: vi.fn(),
  };
  app.use.mockReturnValue(app);

  return {
    app,
    createApp: vi.fn((_component: RootComponent) => app),
    h: vi.fn(),
    logout: vi.fn(),
    t: vi.fn((key: string) => key),
  };
});

vi.mock("vue", () => ({
  createApp: bridgeMocks.createApp,
  h: bridgeMocks.h,
}));

vi.mock("@/plugins/i18n", () => ({
  default: {
    global: {
      t: bridgeMocks.t,
    },
  },
}));

vi.mock("@/plugins/naive-ui", () => ({
  default: {},
}));

vi.mock("naive-ui", () => ({
  NButton: { name: "NButton" },
  NConfigProvider: { name: "NConfigProvider" },
}));

vi.mock("@/../naive-ui.config", () => ({
  dateLang: { value: "date-lang" },
  generalLang: { value: "general-lang" },
  themeOverrides: { value: { common: { primaryColor: "#4f46e5" } } },
}));

vi.mock("@/router", () => ({
  router: {},
}));

vi.mock("@/store", () => ({
  pinia: {},
  useAuthStore: () => ({
    logout: bridgeMocks.logout,
  }),
}));

vi.mock("@/views/auth/Signin.vue", () => ({
  default: { name: "Signin" },
}));

import { SigninBridge } from "./SigninBridge";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("SigninBridge", () => {
  afterEach(() => {
    document.body.innerHTML = "";
    bridgeMocks.createApp.mockClear();
    bridgeMocks.h.mockClear();
    bridgeMocks.app.mount.mockClear();
    bridgeMocks.app.unmount.mockClear();
    bridgeMocks.app.use.mockClear();
    bridgeMocks.app.use.mockReturnValue(bridgeMocks.app);
    bridgeMocks.logout.mockClear();
    bridgeMocks.t.mockClear();
  });

  test("wraps signin with Naive theme config and restores the logout footer", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(<SigninBridge currentPath="/instances" />);
    });

    const render = bridgeMocks.createApp.mock.calls[0]?.[0]?.render;
    expect(render).toBeTypeOf("function");
    if (!render) {
      throw new Error(
        "expected SigninBridge to pass a render function to createApp"
      );
    }

    render();

    const configProviderCall = bridgeMocks.h.mock.calls.find(
      ([component]) =>
        typeof component === "object" &&
        component !== null &&
        "name" in component &&
        component.name === "NConfigProvider"
    );
    expect(configProviderCall?.[1]).toEqual({
      locale: "general-lang",
      dateLocale: "date-lang",
      themeOverrides: { common: { primaryColor: "#4f46e5" } },
    });
    const providerSlots = configProviderCall?.[2] as
      | { default?: () => unknown }
      | undefined;
    expect(providerSlots?.default).toBeTypeOf("function");
    providerSlots?.default?.();

    const signinCall = bridgeMocks.h.mock.calls.find(
      ([component]) =>
        typeof component === "object" &&
        component !== null &&
        "name" in component &&
        component.name === "Signin"
    );
    expect(signinCall).toBeDefined();
    expect(signinCall?.[1]).toEqual({
      redirect: false,
      redirectUrl: "/instances",
      allowSignup: false,
    });

    const slots = signinCall?.[2] as { footer?: () => unknown } | undefined;
    expect(slots?.footer).toBeTypeOf("function");
    slots?.footer?.();

    const footerButtonCall = bridgeMocks.h.mock.calls.find(
      ([component]) =>
        typeof component === "object" &&
        component !== null &&
        "name" in component &&
        component.name === "NButton"
    );
    expect(footerButtonCall?.[1]).toMatchObject({
      quaternary: true,
      size: "small",
    });
    const buttonLabel = footerButtonCall?.[2] as (() => unknown) | undefined;
    buttonLabel?.();
    expect(bridgeMocks.t).toHaveBeenCalledWith("common.logout");

    const onClick = footerButtonCall?.[1]?.onClick as (() => void) | undefined;
    onClick?.();
    expect(bridgeMocks.logout).toHaveBeenCalledTimes(1);

    act(() => {
      root.unmount();
    });
  });
});
