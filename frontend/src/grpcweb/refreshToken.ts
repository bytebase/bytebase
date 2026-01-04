import { authServiceClientConnect } from "@/grpcweb";

const LOCK_NAME = "bb_token_refresh";
const CHANNEL_NAME = "bb_token_refresh";
const WAIT_TIMEOUT_MS = 10000;

let localPromise: Promise<void> | null = null;

/**
 * Refresh the access token using the refresh token cookie.
 * Uses Web Lock API for cross-tab coordination.
 * Only one tab performs the refresh; others wait for completion.
 */
export async function refreshTokens(): Promise<void> {
  // Same-tab deduplication
  if (localPromise) {
    return localPromise;
  }

  localPromise = doRefresh().finally(() => {
    localPromise = null;
  });

  return localPromise;
}

async function doRefresh(): Promise<void> {
  if (await tryAcquireAndRefresh()) {
    return;
  }

  // Another tab is refreshing - wait for broadcast
  await waitForBroadcast();

  // After wait, try once more (handles race condition or failed refresh)
  await tryAcquireAndRefresh();
}

async function tryAcquireAndRefresh(): Promise<boolean> {
  return navigator.locks.request(
    LOCK_NAME,
    { ifAvailable: true },
    async (lock) => {
      if (!lock) {
        return false;
      }
      try {
        await authServiceClientConnect.refresh({});
      } finally {
        // Notify waiting tabs regardless of success/failure
        const channel = new BroadcastChannel(CHANNEL_NAME);
        channel.postMessage("complete");
        channel.close();
      }
      return true;
    }
  );
}

function waitForBroadcast(): Promise<void> {
  return new Promise((resolve) => {
    const channel = new BroadcastChannel(CHANNEL_NAME);

    const cleanup = () => {
      channel.close();
      resolve();
    };

    const timeout = setTimeout(cleanup, WAIT_TIMEOUT_MS);

    channel.onmessage = () => {
      clearTimeout(timeout);
      cleanup();
    };
  });
}
