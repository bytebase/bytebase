import { authServiceClientConnect } from "@/connect";

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
  const receivedBroadcast = await waitForBroadcast();

  // Only retry if timed out (missed broadcast or other tab failed)
  if (!receivedBroadcast) {
    await tryAcquireAndRefresh();
  }
}

async function tryAcquireAndRefresh(): Promise<boolean> {
  return navigator.locks.request(
    LOCK_NAME,
    { ifAvailable: true },
    async (lock) => {
      if (!lock) {
        return false;
      }
      await authServiceClientConnect.refresh({});
      // Only broadcast on success - failure lets waiting tabs timeout and retry
      const channel = new BroadcastChannel(CHANNEL_NAME);
      channel.postMessage("complete");
      channel.close();
      return true;
    }
  );
}

function waitForBroadcast(): Promise<boolean> {
  return new Promise((resolve) => {
    const channel = new BroadcastChannel(CHANNEL_NAME);

    const cleanup = (received: boolean) => {
      channel.close();
      resolve(received);
    };

    const timeout = setTimeout(() => cleanup(false), WAIT_TIMEOUT_MS);

    channel.onmessage = () => {
      clearTimeout(timeout);
      cleanup(true);
    };
  });
}
