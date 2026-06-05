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

vi.mock("@/react/i18n", () => ({
  default: {
    t: (key: string) => key,
  },
}));

vi.mock("@/react/router", () => ({
  router: {
    currentRoute: mocks.currentRoute,
  },
}));

vi.mock("@/store", () => ({
  pushNotification: mocks.pushNotification,
}));

import { errorNotificationInterceptor } from "./errorNotificationMiddleware";

const createRequest = ({
  silent = false,
  ignoredCodes = [],
}: {
  silent?: boolean;
  ignoredCodes?: Code[];
} = {}) =>
  ({
    contextValues: createContextValues()
      .set(silentContextKey, silent)
      .set(ignoredCodesContextKey, ignoredCodes),
    method: { name: "ListDatabases" },
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
