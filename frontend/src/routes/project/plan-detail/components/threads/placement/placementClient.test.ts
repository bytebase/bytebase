import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  createPlacementClient,
  PlacementError,
  type PlacementWorkerPort,
} from "./placementClient";
import {
  type PlacementRequest,
  type PlacementResponse,
  runPlacementBatch,
} from "./placementWorker";

const request = (generation: number): PlacementRequest => ({
  generation,
  limits: { maxLinesPerSheet: 100, maxWorkPerPair: 1000, maxWorkPerRun: 1000 },
  sheets: { s: "A\nB", c: "X\nA\nB" },
  pairs: [
    {
      key: "p",
      saved: "s",
      current: "c",
      anchors: [{ id: "1-1", startLine: 1, endLine: 1 }],
    },
  ],
});

class FakePort implements PlacementWorkerPort {
  onmessage: PlacementWorkerPort["onmessage"] = null;
  onerror: PlacementWorkerPort["onerror"] = null;
  terminated = false;
  received: PlacementRequest[] = [];
  constructor(
    private readonly behavior: (
      port: FakePort,
      request: PlacementRequest
    ) => void = () => {}
  ) {}
  postMessage(message: PlacementRequest) {
    this.received.push(message);
    this.behavior(this, message);
  }
  terminate() {
    this.terminated = true;
  }
  respond(response: PlacementResponse) {
    this.onmessage?.({ data: response } as MessageEvent<PlacementResponse>);
  }
}

const trackingClient = (
  behavior?: ConstructorParameters<typeof FakePort>[0]
) => {
  const ports: FakePort[] = [];
  const client = createPlacementClient(() => {
    const port = new FakePort(behavior);
    ports.push(port);
    return port;
  });
  return { client, ports };
};

const failure = (promise: Promise<unknown>) =>
  promise.then(
    () => undefined,
    (error: unknown) => error
  );

describe("createPlacementClient", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  test("resolves with the worker's response and terminates the worker", async () => {
    const { client, ports } = trackingClient((self, req) =>
      queueMicrotask(() => self.respond(runPlacementBatch(req)))
    );
    const response = await client.compute(request(1), 1000);
    expect(response.results.p["1-1"]).toEqual({
      state: "CURRENT",
      range: { startLine: 2, endLine: 2 },
    });
    expect(ports).toHaveLength(1);
    expect(ports[0].received).toEqual([request(1)]);
    expect(ports[0].terminated).toBe(true);
  });

  test("rejects with deadline and terminates a worker that never answers", async () => {
    const { client, ports } = trackingClient();
    const outcome = failure(client.compute(request(1), 500));
    vi.advanceTimersByTime(499);
    expect(ports[0].terminated).toBe(false);
    vi.advanceTimersByTime(1);
    const error = await outcome;
    expect(error).toBeInstanceOf(PlacementError);
    expect((error as PlacementError).reason).toBe("deadline");
    expect(ports[0].terminated).toBe(true);
    // A late answer after the deadline is ignored.
    expect(() => ports[0].respond(runPlacementBatch(request(1)))).not.toThrow();
  });

  test("rejects with worker-error when the worker fails", async () => {
    const { client } = trackingClient((self) =>
      queueMicrotask(() => self.onerror?.({} as ErrorEvent))
    );
    await expect(client.compute(request(1), 1000)).rejects.toMatchObject({
      reason: "worker-error",
    });
  });

  test("rejects with worker-error when the worker cannot be posted to", async () => {
    const client = createPlacementClient(() => {
      const port = new FakePort();
      port.postMessage = () => {
        throw new Error("structured clone failed");
      };
      return port;
    });
    await expect(client.compute(request(1), 1000)).rejects.toMatchObject({
      reason: "worker-error",
    });
  });

  test("a newer request cancels the one in flight", async () => {
    const { client, ports } = trackingClient();
    const first = failure(client.compute(request(1), 1000));
    const second = client.compute(request(2), 1000);
    expect(await first).toMatchObject({ reason: "cancelled" });
    expect(ports[0].terminated).toBe(true);
    expect(ports[1].terminated).toBe(false);
    ports[1].respond(runPlacementBatch(request(2)));
    expect((await second).generation).toBe(2);
    expect(ports[1].terminated).toBe(true);
  });

  test("cancel rejects the in-flight request and is a no-op when idle", async () => {
    const { client, ports } = trackingClient();
    client.cancel();
    const pending = failure(client.compute(request(1), 1000));
    client.cancel();
    expect(await pending).toMatchObject({ reason: "cancelled" });
    expect(ports[0].terminated).toBe(true);
    client.cancel();
    expect(ports).toHaveLength(1);
  });

  test("ignores a response from a different generation", async () => {
    const { client, ports } = trackingClient();
    const pending = failure(client.compute(request(3), 1000));
    ports[0].respond({ generation: 2, results: {}, incomplete: [], work: 0 });
    expect(ports[0].terminated).toBe(false);
    vi.advanceTimersByTime(1000);
    expect(await pending).toMatchObject({ reason: "deadline" });
  });
});
