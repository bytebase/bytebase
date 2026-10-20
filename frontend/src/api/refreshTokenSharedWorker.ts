let worker: SharedWorker | undefined;
let workerError: Error | undefined;
let workerReady = false;
let localPromise: Promise<void> | null = null;

const WAIT_TIMEOUT_MS = 35000;

/** Refresh shared session cookies through the same worker in every tab. */
export function refreshTokensInWorker(): Promise<void> {
  if (localPromise) {
    return localPromise;
  }
  localPromise = refreshInWorker().finally(() => {
    localPromise = null;
  });
  return localPromise;
}

async function refreshInWorker(): Promise<void> {
  if (workerError) throw workerError;
  if (!worker) {
    worker = new SharedWorker(
      new URL("./refreshToken.worker.ts", import.meta.url),
      {
        type: "module",
        name: "bb_token_refresh",
      }
    );
    const startingWorker = worker;
    worker.addEventListener("error", () => {
      if (worker === startingWorker && workerReady) {
        workerError = new Error("Session refresh worker failed");
      }
    });
    worker.port.start();
  }
  const currentWorker = worker;
  return new Promise<void>((resolve, reject) => {
    const cleanup = () => {
      clearTimeout(timeout);
      currentWorker.port.removeEventListener("message", onMessage);
      currentWorker.port.removeEventListener("messageerror", onFailure);
      currentWorker.removeEventListener("error", onFailure);
    };
    const onFailure = () => {
      cleanup();
      const error = new Error("Session refresh worker failed");
      if (workerReady) {
        workerError ??= error;
      } else {
        // No refresh was dispatched before readiness, so retrying startup is safe.
        currentWorker.port.close();
        worker = undefined;
      }
      reject(workerError ?? error);
    };
    const sendRefresh = () => {
      try {
        currentWorker.port.postMessage("refresh");
      } catch {
        onFailure();
      }
    };
    const onMessage = (
      event: MessageEvent<{ ready?: boolean; error?: string }>
    ) => {
      if (event.data.ready) {
        if (!workerReady) {
          workerReady = true;
          sendRefresh();
        }
        return;
      }
      cleanup();
      if (event.data.error) reject(new Error(event.data.error));
      else resolve();
    };
    // After readiness, a timed-out worker may still be rotating cookies.
    // Retrying could reuse the consumed token.
    const timeout = setTimeout(onFailure, WAIT_TIMEOUT_MS);
    currentWorker.port.addEventListener("message", onMessage);
    currentWorker.port.addEventListener("messageerror", onFailure);
    currentWorker.addEventListener("error", onFailure);
    if (workerReady) sendRefresh();
  });
}
