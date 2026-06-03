import type { Browser } from "@playwright/test";

// Sign a user into a fresh BrowserContext and persist its storageState to
// `storagePath`.
//
// Why POST /v1/auth/login rather than filling the sign-in form: the React
// auth form's inputs aren't reliably reachable via role/name selectors, so
// a form-driven login hangs. `web: true` is the switch that tells
// finalizeLogin() to return the JWT as an HTTP-only cookie instead of only
// in the response body — without it the response is a well-formed 200 but
// the context has no cookies, so the UI stays logged out and any subsequent
// page.goto() bounces to /auth (see backend/api/v1/auth_service.go).
//
// Also pre-warms "/" to set the localStorage flag that suppresses the
// "New version" modal before capturing storageState.
export async function signInBrowserAs(
  browser: Browser,
  baseURL: string,
  email: string,
  password: string,
  storagePath: string,
): Promise<void> {
  const context = await browser.newContext();
  try {
    const resp = await context.request.post(`${baseURL}/v1/auth/login`, {
      data: { email, password, web: true },
      headers: { "Content-Type": "application/json" },
    });
    if (!resp.ok()) {
      throw new Error(
        `login failed for ${email}: ${resp.status()} ${await resp.text()}`,
      );
    }
    const page = await context.newPage();
    await page.goto(`${baseURL}/`);
    await page.evaluate(() => {
      localStorage.setItem(
        "bb.release",
        JSON.stringify({
          ignoreRemindModalTillNextRelease: true,
          nextCheckTs: Date.now() + 86_400_000,
        }),
      );
    });
    await context.storageState({ path: storagePath });
    await page.close();
  } finally {
    await context.close();
  }
}
