// Workspace → MCP: the OAuth 2.1 consent loop an MCP client goes through
// (3.23: #21254 withholds Allow unless the page holds a served ceiling; #21517
// unifies the consent page with the MCP policy page; #21527 retires the
// masking-exemptions toggle).
//
// The loop is driven end to end: dynamic client registration through the API,
// the authorize redirect into the browser's consent page, Allow or Deny, the
// redirect back to the client with a code or an error, and — for Allow — the
// PKCE token exchange and one MCP `initialize` with the issued token. The
// consent page must disclose the workspace ceiling the grant is worth, and a
// DISABLED ceiling must withhold Allow entirely.
//
// State safety: the MCP capability is read in beforeAll and restored in
// afterAll, since a workspace left DISABLED would break every later MCP test.

import { createHash, randomBytes } from "node:crypto";
import { test, expect, type BrowserContext, type Page } from "@playwright/test";
import enUS from "../../../src/locales/en-US.json";
import { loadTestEnv, type TestEnv } from "../framework/env";
import type { BytebaseApiClient } from "../framework/api-client";

const CONSENT = enUS.oauth2.consent;
const MODE = enUS.settings.mcp.policy.mode;
const CLIENT_NAME = "E2E MCP client";

test.setTimeout(120_000);

let env: TestEnv & { api: BytebaseApiClient };
let sharedContext: BrowserContext;
let page: Page;
let clientId: string;
let redirectUri: string;
let originalCapability: string;

async function readCapability(): Promise<string> {
  const setting = (await env.api.getSetting("MCP")) as {
    value?: { mcp?: { capability?: string } };
  } | null;
  return setting?.value?.mcp?.capability ?? "";
}

async function setCapability(capability: string): Promise<void> {
  await env.api.upsertSetting("MCP", { mcp: { capability } }, "value.mcp.capability");
}

const base64url = (bytes: Buffer) => bytes.toString("base64url");

// A fresh PKCE pair per authorization request.
function pkce(): { verifier: string; challenge: string } {
  const verifier = base64url(randomBytes(32));
  const challenge = base64url(createHash("sha256").update(verifier).digest());
  return { verifier, challenge };
}

function authorizeUrl(state: string, challenge: string): string {
  const query = new URLSearchParams({
    response_type: "code",
    client_id: clientId,
    redirect_uri: redirectUri,
    state,
    code_challenge: challenge,
    code_challenge_method: "S256",
  });
  return `${env.baseURL}/api/oauth2/authorize?${query}`;
}

// The callback lands on a path the SPA does not know, so the app routes on
// to its 404 page; the navigation request itself still carries the code or
// the error the server redirected with, so it is captured at request time.
function waitForCallback(): Promise<URL> {
  return page
    .waitForRequest((request) => request.url().startsWith(`${redirectUri}?`), { timeout: 20_000 })
    .then((request) => new URL(request.url()));
}

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  originalCapability = await readCapability();
  if (!originalCapability) {
    throw new Error("could not read the workspace MCP setting; refusing to run, since teardown could not restore it");
  }
  redirectUri = `${env.baseURL}/oauth-e2e-callback`;
  const client = await env.api.registerOAuth2Client(CLIENT_NAME, [redirectUri], [
    "authorization_code",
    "refresh_token",
  ]);
  clientId = client.client_id;
  expect(clientId).not.toBe("");

  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
});

test.afterAll(async () => {
  if (originalCapability) {
    await setCapability(originalCapability);
    expect(await readCapability()).toBe(originalCapability);
  }
  await sharedContext?.close();
});

test.describe("Consent discloses the served ceiling and Allow issues a usable grant (X1)", () => {
  test("Allow access → code → token → MCP initialize", async () => {
    await setCapability("READ_ONLY");
    const { verifier, challenge } = pkce();
    const state = `allow-${Date.now()}`;
    await page.goto(authorizeUrl(state, challenge));

    await expect(page).toHaveURL(/\/oauth2\/consent\?/);
    await expect(page.getByRole("heading", { name: CONSENT.title })).toBeVisible({ timeout: 20_000 });
    await expect(page.getByText(CLIENT_NAME).first()).toBeVisible();
    // The disclosure is the consent: the mode the workspace serves is named.
    await expect(page.getByText(MODE["read-only"].title, { exact: true }).first()).toBeVisible();
    await expect(page.getByRole("button", { name: enUS.common.deny, exact: true })).toBeVisible();

    const callback = waitForCallback();
    await page.getByRole("button", { name: CONSENT["allow-access"], exact: true }).click();
    const url = await callback;
    const code = url.searchParams.get("code") ?? "";
    expect(code).not.toBe("");
    expect(url.searchParams.get("state")).toBe(state);

    const token = await env.api.exchangeOAuth2Code({ clientId, code, codeVerifier: verifier, redirectUri });
    expect(token.error, token.error_description).toBeUndefined();
    expect(token.access_token).toBeTruthy();
    expect(token.refresh_token).toBeTruthy();

    const init = await env.api.mcpInitialize(token.access_token!);
    expect(init.status).toBe(200);
    expect(init.body).toContain('"protocolVersion"');
  });
});

test.describe("Deny sends the client an access_denied error (X3)", () => {
  test("the callback carries error=access_denied and the state", async () => {
    await setCapability("READ_ONLY");
    const { challenge } = pkce();
    const state = `deny-${Date.now()}`;
    await page.goto(authorizeUrl(state, challenge));
    await expect(page.getByRole("heading", { name: CONSENT.title })).toBeVisible({ timeout: 20_000 });

    const callback = waitForCallback();
    await page.getByRole("button", { name: enUS.common.deny, exact: true }).click();
    const url = await callback;
    expect(url.searchParams.get("error")).toBe("access_denied");
    expect(url.searchParams.get("state")).toBe(state);
    expect(url.searchParams.get("code")).toBeNull();
  });
});

test.describe("A DISABLED ceiling withholds Allow (X2)", () => {
  test("the page says MCP is turned off and offers only Close", async () => {
    await setCapability("DISABLED");
    const { challenge } = pkce();
    await page.goto(authorizeUrl(`disabled-${Date.now()}`, challenge));

    await expect(page.getByRole("heading", { name: CONSENT.mcp.disabled.title })).toBeVisible({ timeout: 20_000 });
    await expect(page.getByRole("button", { name: CONSENT["allow-access"] })).toHaveCount(0);
    await expect(page.getByRole("button", { name: enUS.common.deny })).toHaveCount(0);
    await expect(page.getByRole("button", { name: enUS.common.close, exact: true })).toBeVisible();
    await setCapability("READ_ONLY");
  });
});
