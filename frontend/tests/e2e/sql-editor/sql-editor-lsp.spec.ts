// SQL Editor — LSP smoke test (completion over the /lsp websocket).
//
// Locks the language-client wiring in
// frontend/src/react/components/monaco/lsp-client.ts: the inline
// BaseLanguageClient subclass must start against the websocket transports,
// complete the LSP initialize handshake, deliver `setMetadata` via
// workspace/executeCommand, and surface schema-aware completions in
// Monaco's suggest widget. Word-based (document-local) suggestions can't
// produce table names that never appear in the buffer, so a table-name
// completion proves the full LSP round trip.
//
// The backend LSP (backend/api/lsp/handler.go) advertises only
// completionProvider (trigger chars "." and " ") and
// executeCommandProvider — no hoverProvider — so hover is intentionally
// not asserted here.
//
// Single linear test on purpose: the assertions build on each other
// (handshake → metadata → completion → query), and a shared-page cascade
// after a mid-suite failure would only blur the signal.

import { test, expect, type BrowserContext, type Page } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import type { BytebaseApiClient } from "../framework/api-client";
import { SqlEditorPage } from "./sql-editor.page";

test.setTimeout(180_000);

let env: TestEnv & { api: BytebaseApiClient };
let sharedContext: BrowserContext;
let page: Page;
let sqlEditor: SqlEditorPage;

type SocketStats = {
  url: string;
  sentFrames: string[];
  receivedFrames: string[];
  closed: boolean;
};
const lspSockets: SocketStats[] = [];

const frameText = (payload: string | Buffer): string =>
  typeof payload === "string" ? payload : payload.toString("utf-8");

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
  sqlEditor = new SqlEditorPage(page, env.baseURL);

  page.on("websocket", (ws) => {
    if (!new URL(ws.url()).pathname.endsWith("/lsp")) return;
    const stats: SocketStats = {
      url: ws.url(),
      sentFrames: [],
      receivedFrames: [],
      closed: false,
    };
    lspSockets.push(stats);
    ws.on("framesent", (frame) => stats.sentFrames.push(frameText(frame.payload)));
    ws.on("framereceived", (frame) =>
      stats.receivedFrames.push(frameText(frame.payload))
    );
    ws.on("close", () => (stats.closed = true));
  });
});

test.afterAll(async () => {
  await sharedContext?.close();
});

test("LSP language client: handshake, schema completions, and query flow", async () => {
  const projectId = env.project.split("/").pop()!;
  await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
  await expect(sqlEditor.codeEditor).toBeVisible({ timeout: 30_000 });

  // --- 1. Websocket + LSP initialize handshake ---------------------------
  await expect
    .poll(() => lspSockets.length, { timeout: 30_000 })
    .toBeGreaterThan(0);
  // Scan ALL captured sockets, not just [0]: the client's connect retry and
  // its reconnect path each open a fresh websocket, so the handshake can
  // land on a later socket than the first attempt.
  const anySocketFrame = (
    dir: "sentFrames" | "receivedFrames",
    marker: string
  ) => lspSockets.some((s) => s[dir].some((f) => f.includes(marker)));
  // Client sent the LSP initialize request…
  await expect
    .poll(() => anySocketFrame("sentFrames", '"method":"initialize"'), {
      timeout: 30_000,
    })
    .toBe(true);
  // …and the server answered with its capabilities.
  await expect
    .poll(() => anySocketFrame("receivedFrames", '"capabilities"'), {
      timeout: 30_000,
    })
    .toBe(true);
  // The editor pushes connection metadata (instance/database scope for
  // completions) via workspace/executeCommand setMetadata.
  await expect
    .poll(() => anySocketFrame("sentFrames", '"setMetadata"'), {
      timeout: 30_000,
    })
    .toBe(true);

  // No "WebSocket connection failed" CRITICAL notification (the
  // errorHandler path in lsp-client.ts).
  await expect(page.getByText("WebSocket connection failed")).not.toBeVisible();

  // Let the connection settle (client ready + server metadata load) before
  // driving completions.
  await page.waitForTimeout(2_000);

  // Clear-and-type with verification. ControlOrMeta+a select-all is
  // unreliable on macOS headless Chrome (observed dropping the selection,
  // which concatenates buffers), so clear by backspacing the measured
  // content length instead.
  const setBuffer = async (sql: string) => {
    await sqlEditor.codeEditor.click();
    await page.waitForTimeout(200);
    await page.keyboard.press("Escape"); // dismiss any open suggest widget
    for (let round = 0; round < 5; round++) {
      const len = (await sqlEditor.readEditorContent()).length;
      if (len === 0) break;
      await page.keyboard.press("ControlOrMeta+ArrowDown").catch(() => {});
      await page.keyboard.press("End");
      for (let j = 0; j < len + 5; j++) {
        await page.keyboard.press("Backspace", { delay: 5 });
      }
    }
    await page.keyboard.type(sql, { delay: 50 });
  };

  // --- 2. Schema-aware completion --------------------------------------
  // "employee" never appears in the buffer, so only the LSP (fed by
  // setMetadata) can suggest it. Quick-suggest usually opens the widget on
  // its own; fall back to an explicit invoke (Ctrl/Cmd+Space) once if the
  // auto-trigger raced the typing.
  const suggestRow = page
    .locator(".suggest-widget .monaco-list-row")
    .filter({ hasText: "employee" })
    .first();

  await setBuffer("SELECT * FROM emp");
  let visible = await suggestRow
    .waitFor({ state: "visible", timeout: 10_000 })
    .then(() => true)
    .catch(() => false);
  if (!visible) {
    await page.keyboard.press("ControlOrMeta+Space");
    visible = await suggestRow
      .waitFor({ state: "visible", timeout: 10_000 })
      .then(() => true)
      .catch(() => false);
  }
  expect(visible).toBe(true);
  // The row itself is the LSP proof; accepting it via click/keyboard is
  // flaky under headless Chrome and adds nothing, so just dismiss.
  await page.keyboard.press("Escape");

  // --- 3. Trigger-character path ("." is a server trigger char) ----------
  await setBuffer("SELECT * FROM public.");
  await expect(
    page
      .locator(".suggest-widget .monaco-list-row")
      .filter({ hasText: "employee" })
      .first()
  ).toBeVisible({ timeout: 10_000 });
  await page.keyboard.press("Escape");

  // --- 4. Editor still executes queries after the whole LSP flow ---------
  await setBuffer("SELECT 1");
  await sqlEditor.runButton.click();
  await expect(page.getByText(/1 rows?/i).first()).toBeVisible({
    timeout: 15_000,
  });
  // The LSP socket must still be alive at the end.
  expect(lspSockets.some((s) => !s.closed)).toBe(true);
});
