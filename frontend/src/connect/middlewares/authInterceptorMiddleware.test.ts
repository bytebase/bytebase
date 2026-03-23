import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ignoredCodesContextKey, silentContextKey } from "../context-key";

const mocks = vi.hoisted(() => ({
  authStore: {
    unauthenticatedOccurred: false,
    isLoggedIn: true,
  },
  pushNotification: vi.fn(),
  refreshTokens: vi.fn(),
  routerPush: vi.fn(),
  currentRoute: {
    value: {
      name: "workspace.home",
      fullPath: "/instances",
    },
  },
}));

vi.mock("@/plugins/i18n", () => ({
  t: (key: string) => key,
}));

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
  useAuthStore: () => mocks.authStore,
}));

vi.mock("@/router", () => ({
  router: {
    currentRoute: mocks.currentRoute,
    push: mocks.routerPush,
  },
}));

vi.mock("@/router/dashboard/workspaceRoutes", () => ({
  WORKSPACE_ROUTE_403: "workspace.403",
}));

vi.mock("../refreshToken", () => ({
  refreshTokens: mocks.refreshTokens,
}));

import { authInterceptor } from "./authInterceptorMiddleware";

const createRequest = ({
  methodName = "Chat",
  silent = false,
  ignoredCodes = [],
}: {
  methodName?: string;
  silent?: boolean;
  ignoredCodes?: Code[];
} = {}) =>
  ({
    contextValues: createContextValues()
      .set(silentContextKey, silent)
      .set(ignoredCodesContextKey, ignoredCodes),
    method: { name: methodName },
    service: { name: "bytebase.v1.TestService" },
  }) as never;

describe("authInterceptor", () => {
  beforeEach(() => {
    mocks.authStore.unauthenticatedOccurred = false;
    mocks.authStore.isLoggedIn = true;
    mocks.pushNotification.mockReset();
    mocks.refreshTokens.mockReset();
    mocks.routerPush.mockReset();
    mocks.currentRoute.value = {
      name: "workspace.home",
      fullPath: "/instances",
    };
  });

  test("refreshes and retries silent unauthenticated requests", async () => {
    const next = vi
      .fn()
      .mockRejectedValueOnce(
        new ConnectError("access token not found", Code.Unauthenticated)
      )
      .mockResolvedValueOnce({ ok: true });
    mocks.refreshTokens.mockResolvedValue(undefined);

    const result = await authInterceptor(next)(createRequest({ silent: true }));

    expect(result).toEqual({ ok: true });
    expect(mocks.refreshTokens).toHaveBeenCalledTimes(1);
    expect(next).toHaveBeenCalledTimes(2);
    expect(mocks.pushNotification).not.toHaveBeenCalled();
    expect(mocks.authStore.unauthenticatedOccurred).toBe(false);
  });

  test("keeps silent unauthenticated refresh failures notification-free", async () => {
    const next = vi
      .fn()
      .mockRejectedValue(
        new ConnectError("access token not found", Code.Unauthenticated)
      );
    mocks.refreshTokens.mockRejectedValue(new Error("refresh failed"));

    await expect(
      authInterceptor(next)(createRequest({ silent: true }))
    ).rejects.toMatchObject({ code: Code.Unauthenticated });

    expect(mocks.refreshTokens).toHaveBeenCalledTimes(1);
    expect(mocks.authStore.unauthenticatedOccurred).toBe(true);
    expect(mocks.pushNotification).not.toHaveBeenCalled();
  });

  test("notifies on non-silent unauthenticated refresh failure", async () => {
    const next = vi
      .fn()
      .mockRejectedValue(
        new ConnectError("access token not found", Code.Unauthenticated)
      );
    mocks.refreshTokens.mockRejectedValue(new Error("refresh failed"));

    await expect(authInterceptor(next)(createRequest())).rejects.toMatchObject({
      code: Code.Unauthenticated,
    });

    expect(mocks.authStore.unauthenticatedOccurred).toBe(true);
    expect(mocks.pushNotification).toHaveBeenCalledWith({
      module: "bytebase",
      style: "WARN",
      title: "auth.token-expired-title",
      description: "auth.token-expired-description",
    });
  });

  test("preserves ignored unauthenticated codes", async () => {
    const next = vi
      .fn()
      .mockRejectedValue(
        new ConnectError("access token not found", Code.Unauthenticated)
      );

    await expect(
      authInterceptor(next)(
        createRequest({ ignoredCodes: [Code.Unauthenticated] })
      )
    ).rejects.toMatchObject({ code: Code.Unauthenticated });

    expect(mocks.refreshTokens).not.toHaveBeenCalled();
    expect(mocks.authStore.unauthenticatedOccurred).toBe(false);
  });

  test("keeps permission denied redirects gated by silent mode", async () => {
    const error = new ConnectError("forbidden", Code.PermissionDenied);
    const next = vi.fn().mockRejectedValue(error);

    await expect(
      authInterceptor(next)(createRequest({ silent: true }))
    ).rejects.toMatchObject({ code: Code.PermissionDenied });
    expect(mocks.routerPush).not.toHaveBeenCalled();

    await expect(authInterceptor(next)(createRequest())).rejects.toMatchObject({
      code: Code.PermissionDenied,
    });
    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "workspace.403",
      query: undefined,
    });
  });
});
