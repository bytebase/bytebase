import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  refresh: vi.fn(),
}));

vi.mock("@/api", () => ({
  authServiceClientConnect: {
    refresh: mocks.refresh,
  },
}));

import { refreshTokens } from "./refreshToken";

describe("refreshTokens", () => {
  beforeEach(() => {
    mocks.refresh.mockReset();
    mocks.refresh.mockResolvedValue({});
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  test("refreshes under the cross-tab lock when Web Locks is available", async () => {
    const request = vi.fn(
      (
        _name: string,
        _options: LockOptions,
        callback: (lock: Lock | null) => Promise<boolean>
      ) => callback({ name: "bb_token_refresh", mode: "exclusive" })
    );
    vi.stubGlobal("navigator", { locks: { request } });

    await refreshTokens();

    expect(request).toHaveBeenCalledTimes(1);
    expect(mocks.refresh).toHaveBeenCalledTimes(1);
  });

  test("refreshes without Web Locks in a non-secure context", async () => {
    vi.stubGlobal("navigator", {});

    await expect(refreshTokens()).resolves.toBeUndefined();

    expect(mocks.refresh).toHaveBeenCalledTimes(1);
  });

  test("shares one refresh between concurrent callers without Web Locks", async () => {
    vi.stubGlobal("navigator", {});

    await Promise.all([refreshTokens(), refreshTokens()]);

    expect(mocks.refresh).toHaveBeenCalledTimes(1);
  });
});
