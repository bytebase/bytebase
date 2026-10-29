import { afterEach, beforeEach, expect, test, vi } from "vitest";

class TestWorker extends EventTarget {
  port = Object.assign(new EventTarget(), {
    start: vi.fn(),
    close: vi.fn(),
    postMessage: vi.fn(),
  });
}
let worker: TestWorker;
let construct: ReturnType<typeof vi.fn>;
let refreshTokensInWorker: typeof import("./refreshTokenSharedWorker").refreshTokensInWorker;

beforeEach(async () => {
  vi.resetModules();
  worker = new TestWorker();
  construct = vi.fn(function () {
    return worker;
  });
  vi.stubGlobal("SharedWorker", construct);
  ({ refreshTokensInWorker } = await import("./refreshTokenSharedWorker"));
});
afterEach(() => {
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

function respond(data: { ready?: boolean; error?: string } = {}) {
  worker.port.dispatchEvent(new MessageEvent("message", { data }));
}

test("shares one request within a tab without Web Locks and reuses the worker", async () => {
  vi.stubGlobal("navigator", {});
  const a = refreshTokensInWorker();
  const b = refreshTokensInWorker();
  expect(a).toBe(b);
  expect(worker.port.postMessage).not.toHaveBeenCalled();
  respond({ ready: true });
  expect(worker.port.postMessage).toHaveBeenCalledTimes(1);
  respond();
  await Promise.all([a, b]);
  const c = refreshTokensInWorker();
  respond();
  await c;
  expect(construct).toHaveBeenCalledTimes(1);
  expect(worker.port.postMessage).toHaveBeenCalledTimes(2);
});

test("propagates authentication errors and can refresh after signing in again", async () => {
  const first = expect(refreshTokensInWorker()).rejects.toThrow(
    "invalid refresh token"
  );
  respond({ ready: true });
  respond({ error: "invalid refresh token" });
  await first;
  const next = refreshTokensInWorker();
  respond();
  await next;
});

test("worker failure after readiness prevents independent retries", async () => {
  const first = expect(refreshTokensInWorker()).rejects.toThrow(
    "worker failed"
  );
  respond({ ready: true });
  worker.dispatchEvent(new Event("error"));
  await first;
  await expect(refreshTokensInWorker()).rejects.toThrow("worker failed");
  expect(construct).toHaveBeenCalledTimes(1);
  expect(worker.port.postMessage).toHaveBeenCalledTimes(1);
});

test("slow startup leaves a full watchdog interval for refresh", async () => {
  vi.useFakeTimers();
  const first = refreshTokensInWorker();
  const rejected = vi.fn();
  void first.catch(rejected);
  await vi.advanceTimersByTimeAsync(34000);
  expect(worker.port.postMessage).not.toHaveBeenCalled();
  respond({ ready: true });
  await vi.advanceTimersByTimeAsync(34000);
  expect(rejected).not.toHaveBeenCalled();
  expect(worker.port.postMessage).toHaveBeenCalledExactlyOnceWith("refresh");
  respond();
  await first;
  const later = refreshTokensInWorker();
  respond();
  await later;
  expect(construct).toHaveBeenCalledTimes(1);
});

test("a silent worker times out without starting another refresh", async () => {
  vi.useFakeTimers();
  const first = expect(refreshTokensInWorker()).rejects.toThrow(
    "worker failed"
  );
  respond({ ready: true });
  await vi.advanceTimersByTimeAsync(35000);
  await first;
  respond();
  await expect(refreshTokensInWorker()).rejects.toThrow("worker failed");
  expect(worker.port.postMessage).toHaveBeenCalledTimes(1);
});

test("does not fall back to an uncoordinated refresh when SharedWorker is unavailable", async () => {
  vi.stubGlobal("SharedWorker", undefined);
  await expect(refreshTokensInWorker()).rejects.toThrow();
});

for (const failure of ["error", "timeout"] as const) {
  test(`startup ${failure} retries within the same shared refresh promise`, async () => {
    vi.useFakeTimers();
    const first = refreshTokensInWorker();
    const rejected = vi.fn();
    void first.catch(rejected);
    expect(refreshTokensInWorker()).toBe(first);
    const failedWorker = worker;
    if (failure === "error") {
      worker.dispatchEvent(new Event("error"));
    } else {
      await vi.advanceTimersByTimeAsync(35000);
    }
    expect(failedWorker.port.postMessage).not.toHaveBeenCalled();
    expect(failedWorker.port.close).toHaveBeenCalledTimes(1);
    worker = new TestWorker();
    await vi.advanceTimersByTimeAsync(250);
    expect(rejected).not.toHaveBeenCalled();
    expect(refreshTokensInWorker()).toBe(first);
    expect(construct).toHaveBeenCalledTimes(2);
    expect(worker.port.postMessage).not.toHaveBeenCalled();
    respond({ ready: true });
    expect(worker.port.postMessage).toHaveBeenCalledExactlyOnceWith("refresh");
    failedWorker.dispatchEvent(new Event("error"));
    respond();
    await first;
    expect(rejected).not.toHaveBeenCalled();
    const later = refreshTokensInWorker();
    respond();
    await later;
    expect(worker.port.postMessage).toHaveBeenCalledTimes(2);
  });
}

test("persistent startup failures reject after three attempts without refreshing", async () => {
  vi.useFakeTimers();
  const result = expect(refreshTokensInWorker()).rejects.toThrow(
    "worker failed"
  );
  const failedWorkers = [];
  for (let attempt = 1; attempt <= 3; attempt++) {
    failedWorkers.push(worker);
    worker.dispatchEvent(new Event("error"));
    worker = new TestWorker();
    await vi.advanceTimersByTimeAsync(250 * attempt);
  }
  await result;
  expect(construct).toHaveBeenCalledTimes(3);
  for (const failedWorker of failedWorkers) {
    expect(failedWorker.port.close).toHaveBeenCalledTimes(1);
    expect(failedWorker.port.postMessage).not.toHaveBeenCalled();
  }
});
