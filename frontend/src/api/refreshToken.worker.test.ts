import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({ refresh: vi.fn() }));
vi.mock("@connectrpc/connect", () => ({ createClient: () => mocks }));
vi.mock("@connectrpc/connect-web", () => ({ createConnectTransport: vi.fn() }));

function connect() {
  const port = {
    onmessage: undefined as undefined | (() => Promise<void>),
    postMessage: vi.fn(),
  };
  (globalThis as unknown as { onconnect: (event: unknown) => void }).onconnect({
    ports: [port],
  });
  return port;
}

beforeEach(async () => {
  vi.resetModules();
  mocks.refresh.mockReset();
  await import("./refreshToken.worker");
});

afterEach(() => vi.useRealTimers());

describe("shared refresh worker", () => {
  test("lets a slow rotation finish without a client deadline", async () => {
    vi.useFakeTimers();
    const response = Promise.withResolvers<void>();
    mocks.refresh.mockReturnValue(response.promise);
    const a = connect();
    const first = a.onmessage!();
    expect(mocks.refresh).toHaveBeenCalledExactlyOnceWith({});
    await vi.advanceTimersByTimeAsync(40000);
    const b = connect();
    const second = b.onmessage!();
    expect(mocks.refresh).toHaveBeenCalledTimes(1);
    expect(a.postMessage).toHaveBeenCalledExactlyOnceWith({ ready: true });
    expect(b.postMessage).toHaveBeenCalledExactlyOnceWith({ ready: true });
    response.resolve();
    await Promise.all([first, second]);
    expect(a.postMessage).toHaveBeenLastCalledWith({});
    expect(b.postMessage).toHaveBeenLastCalledWith({});
  });

  test("two tabs share one request and a later refresh can rotate again", async () => {
    const response = Promise.withResolvers<void>();
    mocks.refresh
      .mockReturnValueOnce(response.promise)
      .mockResolvedValue(undefined);
    const a = connect();
    const b = connect();
    const first = a.onmessage!();
    const second = b.onmessage!();
    expect(mocks.refresh).toHaveBeenCalledTimes(1);
    expect(a.postMessage).toHaveBeenCalledExactlyOnceWith({ ready: true });
    expect(b.postMessage).toHaveBeenCalledExactlyOnceWith({ ready: true });
    response.resolve();
    await Promise.all([first, second]);
    expect(a.postMessage).toHaveBeenCalledWith({});
    expect(b.postMessage).toHaveBeenCalledWith({});
    await b.onmessage!();
    expect(mocks.refresh).toHaveBeenCalledTimes(2);
  });

  test("all waiting tabs receive a failed refresh and later calls can recover", async () => {
    const response = Promise.withResolvers<void>();
    mocks.refresh
      .mockReturnValueOnce(response.promise)
      .mockResolvedValue(undefined);
    const a = connect();
    const b = connect();
    const first = a.onmessage!();
    const second = b.onmessage!();
    response.reject(new Error("invalid refresh token"));
    await Promise.all([first, second]);
    expect(mocks.refresh).toHaveBeenCalledTimes(1);
    expect(a.postMessage).toHaveBeenCalledWith({
      error: "invalid refresh token",
    });
    expect(b.postMessage).toHaveBeenCalledWith({
      error: "invalid refresh token",
    });
    await a.onmessage!();
    expect(a.postMessage).toHaveBeenLastCalledWith({});
  });
});
