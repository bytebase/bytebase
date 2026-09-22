// @vitest-environment node
import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { ignoredCodesContextKey, silentContextKey } from "../context-key";

const mocks = vi.hoisted(() => ({
  pushNotification: vi.fn(),
  currentRoute: {
    value: {
      name: "workspace.database",
    },
  },
}));

vi.mock("@/lib/i18n", () => ({
  default: {
    t: (key: string) => key,
  },
}));

vi.mock("@/app/router", () => ({
  router: {
    currentRoute: mocks.currentRoute,
  },
}));

vi.mock("@/stores", () => ({
  pushNotification: mocks.pushNotification,
}));

import { errorNotificationInterceptor } from "./errorNotificationMiddleware";

const createRequest = ({
  silent = false,
  ignoredCodes = [],
  methodName = "ListDatabases",
}: {
  silent?: boolean;
  ignoredCodes?: Code[];
  methodName?: string;
} = {}) =>
  ({
    contextValues: createContextValues()
      .set(silentContextKey, silent)
      .set(ignoredCodesContextKey, ignoredCodes),
    method: { name: methodName },
    service: { name: "DatabaseService" },
  }) as never;

describe("errorNotificationInterceptor", () => {
  beforeEach(() => {
    mocks.pushNotification.mockReset();
    mocks.currentRoute.value = {
      name: "workspace.database",
    };
  });

  test("does not toast permission errors that navigate to the guard page", async () => {
    const error = new ConnectError(
      'user does not have permission "bb.databases.list"',
      Code.PermissionDenied
    );
    const next = vi.fn().mockRejectedValue(error);

    await expect(
      errorNotificationInterceptor(next)(createRequest())
    ).rejects.toMatchObject({ code: Code.PermissionDenied });

    expect(mocks.pushNotification).not.toHaveBeenCalled();
  });

  test("shows a rejected web login as a sign-in error", async () => {
    const error = new ConnectError(
      "only users can use web login",
      Code.PermissionDenied
    );
    const next = vi.fn().mockRejectedValue(error);

    await expect(
      errorNotificationInterceptor(next)(createRequest({ methodName: "Login" }))
    ).rejects.toBe(error);

    expect(mocks.pushNotification).toHaveBeenCalledWith({
      module: "bytebase",
      style: "CRITICAL",
      title: "auth.sign-in.failed",
      description: "only users can use web login",
    });
  });

  test("still toasts non-permission request failures", async () => {
    const error = new ConnectError("failed", Code.Internal);
    const next = vi.fn().mockRejectedValue(error);

    await expect(
      errorNotificationInterceptor(next)(createRequest())
    ).rejects.toMatchObject({ code: Code.Internal });

    expect(mocks.pushNotification).toHaveBeenCalledWith({
      module: "bytebase",
      style: "CRITICAL",
      title: "Code 13: Internal",
      description: "[internal] failed",
    });
  });
});
