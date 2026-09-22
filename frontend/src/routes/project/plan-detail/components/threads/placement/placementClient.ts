import type { PlacementRequest, PlacementResponse } from "./placementWorker";

// The subset of `Worker` the client drives, so tests can supply a fake.
export interface PlacementWorkerPort {
  postMessage(message: PlacementRequest): void;
  onmessage: ((event: MessageEvent<PlacementResponse>) => void) | null;
  onerror: ((event: ErrorEvent) => void) | null;
  terminate(): void;
}

export type PlacementWorkerFactory = () => PlacementWorkerPort;

export const createPlacementWorker: PlacementWorkerFactory = () =>
  new Worker(new URL("./placement.worker.ts", import.meta.url), {
    type: "module",
  });

export type PlacementFailure = "deadline" | "cancelled" | "worker-error";

export class PlacementError extends Error {
  constructor(readonly reason: PlacementFailure) {
    super(`placement ${reason}`);
    this.name = "PlacementError";
  }
}

export interface PlacementClient {
  // Runs one request in a fresh worker. Rejects with `PlacementError` when
  // the deadline expires, the worker fails, or a newer request or `cancel`
  // supersedes it. Only one request is in flight; a new one cancels the last.
  compute(
    request: PlacementRequest,
    deadlineMs: number
  ): Promise<PlacementResponse>;
  cancel(): void;
}

interface Timers {
  setTimeout(callback: () => void, ms: number): unknown;
  clearTimeout(handle: unknown): void;
}

const defaultTimers: Timers = {
  setTimeout: (callback, ms) => globalThis.setTimeout(callback, ms),
  clearTimeout: (handle) =>
    globalThis.clearTimeout(handle as ReturnType<typeof setTimeout>),
};

// A worker per request keeps termination simple: expiring the deadline or
// cancelling kills the worker outright, so no partial work can leak into a
// later run.
export function createPlacementClient(
  factory: PlacementWorkerFactory = createPlacementWorker,
  timers: Timers = defaultTimers
): PlacementClient {
  let active:
    | { port: PlacementWorkerPort; fail: (reason: PlacementFailure) => void }
    | undefined;

  const cancel = () => {
    active?.fail("cancelled");
  };

  const compute = (request: PlacementRequest, deadlineMs: number) =>
    new Promise<PlacementResponse>((resolve, reject) => {
      cancel();
      const port = factory();
      let settled = false;
      const finish = () => {
        settled = true;
        timers.clearTimeout(timer);
        port.onmessage = null;
        port.onerror = null;
        port.terminate();
        if (active?.port === port) active = undefined;
      };
      const fail = (reason: PlacementFailure) => {
        if (settled) return;
        finish();
        reject(new PlacementError(reason));
      };
      const timer = timers.setTimeout(() => fail("deadline"), deadlineMs);
      port.onmessage = (event) => {
        if (settled) return;
        // A response for another generation cannot come from this worker, but
        // guard the contract anyway.
        if (event.data.generation !== request.generation) return;
        finish();
        resolve(event.data);
      };
      port.onerror = () => fail("worker-error");
      active = { port, fail };
      try {
        port.postMessage(request);
      } catch {
        fail("worker-error");
      }
    });

  return { compute, cancel };
}
