import { authServiceClientConnect } from "@/grpcweb";

const LOCK_KEY = "bb_refresh_lock";
const LOCK_TIMEOUT_MS = 10000; // 10 seconds max lock hold time
const POLL_INTERVAL_MS = 50;

let localPromise: Promise<void> | null = null;

/**
 * Refresh the access token using the refresh token cookie.
 * Uses localStorage lock for cross-tab coordination.
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
  // Try to acquire lock
  if (!tryAcquireLock()) {
    // Another tab is refreshing - wait for it to complete
    await waitForLockRelease();
    return;
  }

  try {
    await authServiceClientConnect.refresh({});
  } finally {
    releaseLock();
  }
}

function tryAcquireLock(): boolean {
  const now = Date.now();
  const existing = localStorage.getItem(LOCK_KEY);

  if (existing) {
    const lockTime = parseInt(existing, 10);
    // Lock is still valid (not expired)
    if (now - lockTime < LOCK_TIMEOUT_MS) {
      return false;
    }
    // Lock expired - we can take over
  }

  localStorage.setItem(LOCK_KEY, now.toString());

  // Double-check we got the lock (handles near-simultaneous writes)
  const check = localStorage.getItem(LOCK_KEY);
  return check === now.toString();
}

function releaseLock(): void {
  localStorage.removeItem(LOCK_KEY);
}

async function waitForLockRelease(): Promise<void> {
  const startTime = Date.now();

  return new Promise((resolve) => {
    const checkLock = () => {
      const existing = localStorage.getItem(LOCK_KEY);

      // Lock released or expired
      if (!existing || Date.now() - parseInt(existing, 10) >= LOCK_TIMEOUT_MS) {
        resolve();
        return;
      }

      // Safety timeout - don't wait forever
      if (Date.now() - startTime >= LOCK_TIMEOUT_MS) {
        resolve();
        return;
      }

      setTimeout(checkLock, POLL_INTERVAL_MS);
    };

    checkLock();
  });
}
