import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  clients: [] as MockLanguageClient[],
  errorNotification: vi.fn(),
  initializeMonacoServices: vi.fn(async () => undefined),
  refreshTokens: vi.fn(async () => undefined),
  sleep: vi.fn(async () => undefined),
}));

class MockLanguageClient {
  readonly clientOptions: {
    errorHandler?: {
      closed?: () => { action: number };
    };
  };
  readonly onDidChangeState = vi.fn();
  readonly sendRequest = vi.fn(async () => undefined);
  readonly start = vi.fn(async () => undefined);
  readonly dispose = vi.fn();

  constructor(
    _id: string,
    _name: string,
    clientOptions: MockLanguageClient["clientOptions"]
  ) {
    this.clientOptions = clientOptions;
    mocks.clients.push(this);
  }
}

class MockWebSocket extends EventTarget {
  static readonly OPEN = 1;
  static instances: MockWebSocket[] = [];

  readonly close = vi.fn(() => {
    this.readyState = 3;
    this.dispatchClose();
  });
  readonly send = vi.fn();
  readyState = 0;

  constructor(readonly url: string) {
    super();
    MockWebSocket.instances.push(this);
  }

  open() {
    this.readyState = MockWebSocket.OPEN;
    this.dispatchEvent(new Event("open"));
  }

  closeBeforeOpen() {
    this.readyState = 3;
    this.dispatchClose();
  }

  private dispatchClose() {
    const event = new Event("close") as CloseEvent;
    Object.defineProperties(event, {
      code: { value: 1006 },
      reason: { value: "closed" },
    });
    this.dispatchEvent(event);
  }
}

vi.mock("vscode-languageclient", () => ({
  BaseLanguageClient: MockLanguageClient,
  CloseAction: {
    DoNotRestart: 1,
  },
  ErrorAction: {
    Continue: 1,
  },
  State: {
    Running: 2,
  },
}));

vi.mock("vscode-ws-jsonrpc", () => ({
  toSocket: (ws: WebSocket) => ws,
  WebSocketMessageReader: class {
    constructor(readonly _socket: WebSocket) {}
  },
  WebSocketMessageWriter: class {
    constructor(readonly _socket: WebSocket) {}
  },
}));

vi.mock("@/api/refreshToken", () => ({
  refreshTokens: mocks.refreshTokens,
}));

vi.mock("@/utils", () => ({
  sleep: mocks.sleep,
}));

vi.mock("./services", () => ({
  initializeMonacoServices: mocks.initializeMonacoServices,
}));

vi.mock("./utils", () => ({
  createUrl: (_host: string, path: string) =>
    new URL(`ws://example.com${path}`),
  errorNotification: mocks.errorNotification,
  MAX_RETRIES: 5,
  messages: {
    disconnected: () => "WebSocket disconnected",
  },
  progressiveDelay: () => 0,
  WEBSOCKET_HEARTBEAT_INTERVAL: 10_000,
  WEBSOCKET_HEARTBEAT_TIMEOUT: 30_000,
  WEBSOCKET_TIMEOUT: 5_000,
}));

const flushPromises = async () => {
  await Promise.resolve();
  await Promise.resolve();
};

const deferred = <T>() => {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, reject, resolve };
};

beforeEach(() => {
  vi.useFakeTimers();
  vi.resetModules();
  vi.clearAllMocks();
  mocks.clients.length = 0;
  MockWebSocket.instances = [];
  vi.stubGlobal("WebSocket", MockWebSocket);
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

describe("LSP client connection recovery", () => {
  test("closes the WebSocket when heartbeat ping does not return", async () => {
    const { initializeLSPClient } = await import("./lsp-client");

    const initializing = initializeLSPClient();
    await flushPromises();
    const ws = MockWebSocket.instances[0];
    ws.open();
    await initializing;

    const client = mocks.clients[0];
    client.sendRequest.mockReturnValueOnce(new Promise(() => undefined));

    await vi.advanceTimersByTimeAsync(10_000);
    expect(client.sendRequest).toHaveBeenCalledWith("$ping", {
      state: {
        counter: 1,
        timestamp: expect.any(Number),
      },
    });

    await vi.advanceTimersByTimeAsync(30_000);
    expect(ws.close).toHaveBeenCalled();
  });

  test("ensureLSPConnection reconnects after the connection reaches closed state", async () => {
    const {
      ensureLSPConnection,
      getConnectionStateSnapshot,
      initializeLSPClient,
    } = await import("./lsp-client");

    const failedInitialConnection = initializeLSPClient().catch(
      () => undefined
    );
    for (let i = 0; i < 5; i++) {
      await flushPromises();
      MockWebSocket.instances[i].closeBeforeOpen();
    }
    await failedInitialConnection;
    expect(getConnectionStateSnapshot().state).toBe("closed");

    const reconnecting = ensureLSPConnection();
    await flushPromises();
    const ws = MockWebSocket.instances[5];
    ws.open();
    await reconnecting;

    expect(mocks.refreshTokens).toHaveBeenCalled();
    expect(getConnectionStateSnapshot().state).toBe("ready");
  });

  test("marks the connection closed when reconnect cannot refresh tokens", async () => {
    const { getConnectionStateSnapshot, initializeLSPClient } = await import(
      "./lsp-client"
    );

    const initializing = initializeLSPClient();
    await flushPromises();
    MockWebSocket.instances[0].open();
    await initializing;
    expect(getConnectionStateSnapshot().state).toBe("ready");

    mocks.refreshTokens.mockRejectedValueOnce(new Error("offline"));
    mocks.clients[0].clientOptions.errorHandler?.closed?.();
    await flushPromises();

    expect(getConnectionStateSnapshot().state).toBe("closed");
  });

  test("automatically retries a closed connection after reconnect fails", async () => {
    const { getConnectionStateSnapshot, initializeLSPClient } = await import(
      "./lsp-client"
    );

    const initializing = initializeLSPClient();
    await flushPromises();
    MockWebSocket.instances[0].open();
    await initializing;

    mocks.refreshTokens.mockRejectedValueOnce(new Error("offline"));
    mocks.clients[0].clientOptions.errorHandler?.closed?.();
    await flushPromises();
    expect(getConnectionStateSnapshot().state).toBe("closed");

    await vi.advanceTimersByTimeAsync(10_000);
    await flushPromises();
    MockWebSocket.instances[1].open();
    await flushPromises();

    expect(mocks.clients).toHaveLength(2);
    expect(getConnectionStateSnapshot().state).toBe("ready");
  });

  test("keeps retrying disconnected connections on the heartbeat interval until success", async () => {
    const { getConnectionStateSnapshot, initializeLSPClient } = await import(
      "./lsp-client"
    );

    const initializing = initializeLSPClient();
    await flushPromises();
    MockWebSocket.instances[0].open();
    await initializing;

    mocks.refreshTokens
      .mockRejectedValueOnce(new Error("offline"))
      .mockRejectedValueOnce(new Error("still offline"));
    mocks.clients[0].clientOptions.errorHandler?.closed?.();
    await flushPromises();
    expect(getConnectionStateSnapshot().state).toBe("closed");

    await vi.advanceTimersByTimeAsync(10_000);
    await flushPromises();
    expect(getConnectionStateSnapshot().state).toBe("closed");

    await vi.advanceTimersByTimeAsync(10_000);
    await flushPromises();
    MockWebSocket.instances[1].open();
    await flushPromises();

    expect(mocks.refreshTokens).toHaveBeenCalledTimes(3);
    expect(mocks.clients).toHaveLength(2);
    expect(getConnectionStateSnapshot().state).toBe("ready");
  });

  test("shares one reconnect attempt across concurrent callers", async () => {
    const {
      ensureLSPConnection,
      getConnectionStateSnapshot,
      initializeLSPClient,
    } = await import("./lsp-client");

    const failedInitialConnection = initializeLSPClient().catch(
      () => undefined
    );
    for (let i = 0; i < 5; i++) {
      await flushPromises();
      MockWebSocket.instances[i].closeBeforeOpen();
    }
    await failedInitialConnection;
    expect(getConnectionStateSnapshot().state).toBe("closed");

    const refresh = deferred<undefined>();
    mocks.refreshTokens.mockReturnValueOnce(refresh.promise);

    const first = ensureLSPConnection();
    const second = ensureLSPConnection();

    expect(first).toBe(second);
    expect(mocks.refreshTokens).toHaveBeenCalledTimes(1);

    refresh.resolve(undefined);
    await flushPromises();
    await flushPromises();
    MockWebSocket.instances[5].open();
    await Promise.all([first, second]);

    expect(mocks.clients).toHaveLength(1);
    expect(getConnectionStateSnapshot().state).toBe("ready");
  });
});
