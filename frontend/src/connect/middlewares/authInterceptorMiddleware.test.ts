import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Permission } from "@/types";
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
      requiredPermissions: [] as Permission[],
    },
  },
}));

vi.mock("@/react/i18n", () => ({
  default: {
    t: (key: string) => key,
  },
}));

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/utils/app-store-bridge", () => ({
  appStoreUtilBridge: () => ({
    isLoggedIn: () => mocks.authStore.isLoggedIn,
    setUnauthenticatedOccurred: (v: boolean) => {
      mocks.authStore.unauthenticatedOccurred = v;
    },
  }),
}));

vi.mock("@/react/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/react/router")>()),
  router: {
    currentRoute: mocks.currentRoute,
    push: mocks.routerPush,
  },
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
    service: { name: "TestService", typeName: "bytebase.v1.TestService" },
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
      requiredPermissions: [],
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
    expect(mocks.pushNotification).toHaveBeenCalledTimes(1);
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
      name: "error.403",
      query: {
        from: "/instances",
        api: "/bytebase.v1.TestService/Chat",
        permissions: "",
        resources: "",
      },
    });
  });

  test("builds a permission denied route query from request and route metadata", async () => {
    mocks.currentRoute.value = {
      name: "workspace.database",
      fullPath: "/databases?q=project:unassigned",
      requiredPermissions: ["bb.databases.list"],
    };
    const error = new ConnectError(
      'user does not have permission "bb.databases.list"',
      Code.PermissionDenied
    );
    const next = vi.fn().mockRejectedValue(error);

    await expect(authInterceptor(next)(createRequest())).rejects.toMatchObject({
      code: Code.PermissionDenied,
    });

    expect(mocks.routerPush).toHaveBeenCalledWith({
      name: "error.403",
      query: {
        from: "/databases?q=project:unassigned",
        api: "/bytebase.v1.TestService/Chat",
        permissions: "bb.databases.list",
        resources: "",
      },
    });
  });
});
