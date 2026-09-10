// Workspace Settings → MCP Integration → Access policy: the capability ladder.
//
// Since #21324 removed the per-mode method drawer, nothing on the page said
// what a mode actually serves, so an admin choosing between Read-only and
// Read-write had two sentences to compare. The rework puts eight capability
// rows in one ordered list of which every mode is a prefix, collapsed inside
// the Access policy section and following the picked mode while editing.
//
// Covers:
//   - M1 view: the mode chip is the subject of the card, the disclosure line
//     is the mode's description, and the retired "In force" label, audit line
//     and Authentication Required alert are gone.
//   - M2 disclosure: opening the list shows all eight rows under either mode,
//     with the tier tag on the served prefix only; "Show details" reveals the
//     sub-item line on every row at once; both preferences survive a reload.
//   - M3 edit → save: picking Read-write renders the post-save list before
//     saving, the footer names the transition, and Save persists the ceiling
//     (verified through the API) and changes the chip.
//
// State safety: the MCP setting is snapshotted in beforeAll and restored via
// API in afterAll, because a workspace left on Read-write would change what a
// later suite's MCP session is allowed to do. This file runs before
// workspace-seat-limit's license drop (directory order).

import { test, expect, type Page } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import { BytebaseApiClient } from "../framework/api-client";

test.setTimeout(120_000);

let env: TestEnv & { api: BytebaseApiClient };
let originalMCPSetting: Record<string, unknown> = { capability: "READ_ONLY" };

// The eight row titles, in list order. They are the product's claim about what
// a mode allows, and the same strings the consent page reuses.
const READ_ROWS = [
  "Read schemas and metadata",
  "Read data by running queries",
  "Read the change workflow",
];
const WRITE_ROWS = [
  "Propose changes",
  "Run rollouts and tasks",
  "Run DML and DDL statements",
  "Export query results",
  "Manage database housekeeping",
];

const READ_ONLY_SUMMARY =
  "Read schemas, data and the change workflow; statements that write are refused, nothing is exported";
const READ_WRITE_SUMMARY =
  "Read schemas, data and the change workflow; propose, run, export and manage";
const EXPORT_DETAILS = "Download results as a file. Data leaves Bytebase";

async function readCapability(): Promise<string> {
  const setting = (await env.api.getSetting("MCP")) as {
    value?: { mcp?: { capability?: string } };
  } | null;
  return setting?.value?.mcp?.capability ?? "";
}

async function setCapability(capability: string): Promise<void> {
  await env.api.upsertSetting(
    "MCP",
    { mcp: { ...originalMCPSetting, capability } },
    "value.mcp.capability"
  );
}

async function gotoMCPPage(page: Page): Promise<void> {
  await page.goto(`${env.baseURL}/integration/mcp`);
  await page.waitForLoadState("networkidle").catch(() => {});
  await expect(
    page.getByRole("heading", { name: "Access policy", exact: true })
  ).toBeVisible({ timeout: 10_000 });
}

function chip(page: Page, mode: string) {
  return page.locator(`[aria-label="Current policy: ${mode}"]`);
}

// The row's list item, so a title that also appears inside a summary sentence
// cannot satisfy the locator.
function row(page: Page, title: string) {
  return page.locator("li").filter({ hasText: title }).first();
}

// The disclosure remembers itself per browser, so a test that needs the list
// open asks for that state rather than assuming a starting point.
async function openLadder(page: Page, summary: string): Promise<void> {
  const trigger = page.getByRole("button", { name: summary });
  if ((await trigger.getAttribute("aria-expanded")) === "false") {
    await trigger.click();
  }
  await expect(row(page, READ_ROWS[0])).toBeVisible();
}

test.beforeAll(async () => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  const setting = (await env.api.getSetting("MCP")) as {
    value?: { mcp?: Record<string, unknown> };
  } | null;
  if (setting?.value?.mcp) {
    originalMCPSetting = setting.value.mcp;
  }
});

test.afterAll(async () => {
  try {
    await env.api.upsertSetting(
      "MCP",
      { mcp: originalMCPSetting },
      "value.mcp.capability"
    );
  } catch {
    /* best-effort restore */
  }
});

test.describe("MCP access policy capability ladder", () => {
  test("the chip is the subject of the view and the disclosure carries the description (M1)", async ({
    page,
  }) => {
    await setCapability("READ_ONLY");
    await gotoMCPPage(page);

    await expect(chip(page, "Read-only")).toBeVisible();
    await expect(page.getByText(READ_ONLY_SUMMARY)).toBeVisible();

    // Retired by the rework: the "In force" label, and the audit line that
    // moved into the section description because it is a fact about the
    // feature rather than about the current state.
    await expect(page.getByText("In force")).toHaveCount(0);
    await expect(
      page.getByText("MCP policy denials are recorded in the audit log.")
    ).toHaveCount(0);
    await expect(
      page.getByText(
        "The most any MCP session may do here. Sessions are also capped by each user's permissions, and policy refusals are audited."
      )
    ).toBeVisible();

    // D9: the alert's two sentences already existed elsewhere on the page, and
    // the one clause worth keeping moved into Connect a client.
    await expect(page.getByText("Authentication Required")).toHaveCount(0);
    await expect(
      page.getByText(
        "Add Bytebase to your AI client and start asking. On first connection you sign in and approve access in the browser."
      )
    ).toBeVisible();

    // Collapsed by default: the list is a disclosure, not the card.
    await expect(row(page, READ_ROWS[0])).toHaveCount(0);
  });

  test("the list shows every row under either mode, tagging only the mode's prefix (M2)", async ({
    page,
  }) => {
    await setCapability("READ_ONLY");
    await gotoMCPPage(page);
    await openLadder(page, READ_ONLY_SUMMARY);

    // Rows a mode does not serve stay visible and muted, so comparing two
    // modes never needs a second surface.
    for (const title of [...READ_ROWS, ...WRITE_ROWS]) {
      await expect(row(page, title)).toBeVisible();
    }
    // The tier tag rides the served rows only, so the word "write" never
    // appears beside something Read-only allows.
    for (const title of READ_ROWS) {
      await expect(
        row(page, title).getByText("read", { exact: true })
      ).toBeVisible();
    }
    for (const title of WRITE_ROWS) {
      await expect(
        row(page, title).getByText("write", { exact: true })
      ).toHaveCount(0);
    }
    await expect(page.getByText("Read-only stops here")).toBeVisible();
    await expect(page.getByText("Read-write stops here")).toBeVisible();
    await expect(
      page.getByText(
        "approve issues, administer the workspace, or handle credentials."
      )
    ).toBeVisible();

    // One second-level control for all eight rows, not eight expanders.
    await expect(page.getByText(EXPORT_DETAILS)).toHaveCount(0);
    await page.getByRole("button", { name: "Show details" }).click();
    await expect(page.getByText(EXPORT_DETAILS)).toBeVisible();
    await expect(
      page.getByText(
        "Issues and comments · Plans and plan checks · Rollouts, task runs and logs · Releases · Rollback previews"
      )
    ).toBeVisible();

    // Both preferences are remembered per browser.
    await page.reload();
    await page.waitForLoadState("networkidle").catch(() => {});
    await expect(page.getByText(EXPORT_DETAILS)).toBeVisible({
      timeout: 10_000,
    });
  });

  test("picking Read-write renders the post-save list, and Save persists it (M3)", async ({
    page,
  }) => {
    await setCapability("READ_ONLY");
    await gotoMCPPage(page);
    await page.getByRole("button", { name: "Edit policy" }).click();

    // The cards are an icon, the mode name and a three-word caption; the
    // "Best for" line is shown once, for the pick.
    await expect(page.getByText("Explore and query")).toBeVisible();
    await expect(
      page.getByText(
        "Best for: querying and exploring data, including by people who do not write SQL."
      )
    ).toBeVisible();

    await page.getByText("Change data and schemas").click();
    await expect(
      page.getByText(
        "Best for: making database changes through an AI agent, still capped by each user's own permissions."
      )
    ).toBeVisible();

    // No delta marks and no "adds N": the edit state renders exactly the view
    // that saving will produce.
    await openLadder(page, READ_WRITE_SUMMARY);
    for (const title of WRITE_ROWS) {
      await expect(
        row(page, title).getByText("write", { exact: true })
      ).toBeVisible();
    }

    // The footer names the transition rather than stating a general rule.
    await expect(
      page.getByText(
        "Read-only → Read-write applies to every running session's next request."
      )
    ).toBeVisible();

    await page.getByRole("button", { name: "Save policy" }).click();
    await expect.poll(readCapability, { timeout: 10_000 }).toBe("READ_WRITE");
    await expect(chip(page, "Read-write")).toBeVisible();
    await expect(page.getByText(READ_WRITE_SUMMARY)).toBeVisible();
  });
});
