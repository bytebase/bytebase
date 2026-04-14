import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";

const bridgeMocks = vi.hoisted(() => {
  const app = {
    mount: vi.fn(),
    unmount: vi.fn(),
    use: vi.fn(),
  };
  app.use.mockReturnValue(app);

  return {
    app,
    createApp: vi.fn(() => app),
    h: vi.fn(),
  };
});

vi.mock("vue", () => ({
  createApp: bridgeMocks.createApp,
  h: bridgeMocks.h,
}));

vi.mock("@/plugins/i18n", () => ({
  default: {},
}));

vi.mock("@/plugins/naive-ui", () => ({
  default: {},
}));

vi.mock("@/router", () => ({
  router: {},
}));

vi.mock("@/store", () => ({
  pinia: {},
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
  });

  test("suppresses the signin footer slot", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root = createRoot(container);

    act(() => {
      root.render(<SigninBridge currentPath="/instances" />);
    });

    const render = bridgeMocks.createApp.mock.calls[0]?.[0]?.render;
    expect(render).toBeTypeOf("function");

    render();

    expect(bridgeMocks.h).toHaveBeenCalledTimes(1);
    expect(bridgeMocks.h.mock.calls[0]?.[1]).toEqual({
      redirect: false,
      redirectUrl: "/instances",
      allowSignup: false,
    });

    const slots = bridgeMocks.h.mock.calls[0]?.[2] as
      | { footer?: () => unknown }
      | undefined;
    expect(slots?.footer).toBeTypeOf("function");
    expect(slots?.footer?.()).toBeNull();

    act(() => {
      root.unmount();
    });
  });
});
