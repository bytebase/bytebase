import { type PlacementRequest, runPlacementBatch } from "./placementWorker";

// Dedicated worker entry: one request in, one response out.
const scope = globalThis as unknown as {
  addEventListener(
    type: "message",
    listener: (event: MessageEvent<PlacementRequest>) => void
  ): void;
  postMessage(message: unknown): void;
};

scope.addEventListener("message", (event) => {
  scope.postMessage(runPlacementBatch(event.data));
});
