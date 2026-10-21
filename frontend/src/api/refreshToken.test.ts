import { afterEach, beforeEach, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({ refresh: vi.fn(), workerRefresh: vi.fn() }));
vi.mock("@/api", () => ({
  authServiceClientConnect: { refresh: mocks.refresh },
}));
vi.mock("./refreshTokenSharedWorker", () => ({
  refreshTokensInWorker: mocks.workerRefresh,
}));

import { refreshTokens } from "./refreshToken";

beforeEach(() => {
  mocks.refresh.mockReset().mockResolvedValue({});
  mocks.workerRefresh.mockReset().mockResolvedValue(undefined);
});
afterEach(() => vi.unstubAllGlobals());

test("uses Web Locks when available without starting a SharedWorker", async () => {
  const request = vi.fn(
    (
      _name: string,
      _options: LockOptions,
      callback: (lock: Lock) => Promise<boolean>
    ) => callback({ name: "bb_token_refresh", mode: "exclusive" })
  );
  vi.stubGlobal("navigator", { locks: { request } });
  await refreshTokens();
  expect(request).toHaveBeenCalledTimes(1);
  expect(mocks.refresh).toHaveBeenCalledTimes(1);
  expect(mocks.workerRefresh).not.toHaveBeenCalled();
});

test("shares the SharedWorker fallback between callers without Web Locks", async () => {
  vi.stubGlobal("navigator", {});
  await Promise.all([refreshTokens(), refreshTokens()]);
  expect(mocks.workerRefresh).toHaveBeenCalledTimes(1);
  expect(mocks.refresh).not.toHaveBeenCalled();
});

test("propagates fallback errors without an uncoordinated retry", async () => {
  vi.stubGlobal("navigator", {});
  mocks.workerRefresh.mockRejectedValue(new Error("worker failed"));
  await expect(refreshTokens()).rejects.toThrow("worker failed");
  expect(mocks.refresh).not.toHaveBeenCalled();
});
