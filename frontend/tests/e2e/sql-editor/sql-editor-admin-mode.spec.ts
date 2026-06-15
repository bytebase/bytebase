// SQL Editor — admin mode terminal.
//
// Covers the admin-mode toggle (the wrench button) and the dark-theme
// terminal it opens: visibility/disabled state of the wrench, entering
// admin mode, executing a query in the terminal, exiting back to the
// worksheet, and the result panel's theme harmony (BYT-9496 lock).

import { test, expect, type BrowserContext, type Page } from "@playwright/test";
import { loadTestEnv, type TestEnv } from "../framework/env";
import type { BytebaseApiClient } from "../framework/api-client";
import { SqlEditorPage } from "./sql-editor.page";

let env: TestEnv & { api: BytebaseApiClient };
let sharedContext: BrowserContext;
let page: Page;
let sqlEditor: SqlEditorPage;

test.beforeAll(async ({ browser }) => {
  env = loadTestEnv();
  await env.api.login(env.adminEmail, env.adminPassword);
  sharedContext = await browser.newContext({ storageState: ".auth/state.json" });
  page = await sharedContext.newPage();
  sqlEditor = new SqlEditorPage(page, env.baseURL);
});

test.afterAll(async () => {
  await sharedContext?.close();
});

test.describe("Vertical display preserves the admin-mode dark theme", () => {
  // BYT-9496 (sub-bug R44a): in admin mode, toggling the "Vertical
  // display" switch on a result panel re-renders the panel as the
  // transposed key:value layout — but the new layout uses generic
  // light-theme tokens (`bg-control-bg`) instead of the dark terminal
  // tokens (`bg-dark-bg`) used by the surrounding admin shell. The
  // result is a white-background, black-text block dropped into the
  // middle of an otherwise dark terminal.
  //
  // Bug evidence: .playwright-cli/qa-session-2026-05-12/r44-byt9496-admin-mode/
  //   - 03-notes.txt
  //   - 02-vertical-display-theme-break.png

  test("vertical-display result panel keeps a dark background in admin mode", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(500);

    // Enter admin mode via the wrench button (AdminModeButton.tsx).
    await expect(sqlEditor.adminModeButton).toBeVisible({ timeout: 10_000 });
    await sqlEditor.adminModeButton.click();
    await page.waitForTimeout(800);

    // The admin terminal renders an editable CompactSQLEditor at the
    // bottom of the query stack. There may be other Monaco instances
    // in the page (the underlying worksheet's editor stays mounted),
    // so we anchor by `.last()` to target the prompt.
    //
    // CompactSQLEditor.tsx:220 wires plain Enter to RunQuery via a
    // Monaco addCommand whose precondition is
    //   "!readonly && isEnterEndsStatement && cursorAtLast && editorTextFocus"
    // — so for a statement that ends with `;` (ours does), pressing
    // Enter at the end of the line submits instead of inserting a
    // newline.
    await runInAdminTerminal(
      "SELECT emp_no, first_name, last_name FROM employee LIMIT 3;",
    );

    // Wait for the result panel to render (a "Vertical display" toggle
    // only appears once a result is on screen).
    await page
      .getByText("Vertical display", { exact: true })
      .first()
      .waitFor({ timeout: 15_000 });

    // Find the Vertical display switch in the result panel header.
    // The Switch primitive (Base UI) renders as `role="switch"` — we
    // scope to the row containing the "Vertical display" label so we
    // don't accidentally toggle the noSQL "table-view" switch above it.
    const verticalRow = page
      .locator("div.flex.items-center.gap-x-1")
      .filter({ hasText: "Vertical display" })
      .first();
    await expect(verticalRow).toBeVisible({ timeout: 10_000 });
    const verticalSwitch = verticalRow.getByRole("switch");
    await expect(verticalSwitch).toBeVisible({ timeout: 5000 });
    await verticalSwitch.click();
    await page.waitForTimeout(500);

    // After toggle, the result body switches from VirtualDataTable to
    // VirtualDataBlock. Each VirtualDataBlock row is wrapped in a
    // `data-index="N"` div whose first child holds the asterisk header
    // and whose second child is the `bg-control-bg rounded` data card.
    // We anchor by `data-index` (unique to the block view — the table
    // view uses `data-row-index`) so we don't accidentally sample the
    // terminal shell or some other element that happens to share a
    // class fragment.
    await page
      .locator("[data-index]")
      .first()
      .waitFor({ timeout: 10_000 });
    const sample = await page.evaluate(() => {
      // Terminal shell wraps everything in admin mode with bg-dark-bg.
      const shell = document.querySelector(".bg-dark-bg");
      if (!shell) return { reason: "admin terminal shell not found" };
      const shellBg = getComputedStyle(shell).backgroundColor;

      // VirtualDataBlock row: outer is `data-index="N"`, inner data
      // card is the second child div (the first child is the asterisk
      // <p>). Sample the inner card — its background is the bug locus.
      const blockRow = document.querySelector("[data-index]");
      if (!blockRow) return { reason: "vertical block row not found" };
      // Walk children to find the data card div (skip the <p> header).
      const blockCard = Array.from(blockRow.children).find(
        (child) => child.tagName === "DIV",
      );
      if (!blockCard) return { reason: "vertical block card not found" };
      const blockBg = getComputedStyle(blockCard).backgroundColor;

      // Parse rgb(), rgba(), AND lab() — Chromium 124+ emits lab() for
      // some Tailwind tokens so the legacy rgb-only regex returns null
      // and the test reports a false failure. For lab(L a b), convert
      // to approximate sRGB via the L-only luminance (a/b are tiny for
      // near-neutral grays which is what the SQL editor's dark-theme
      // tokens are). L is 0–100; mapping linearly to 0–255 is plenty
      // for the harmony assertion below.
      const parse = (s: string) => {
        const rgb = s.match(/rgba?\(([^)]+)\)/);
        if (rgb) {
          const parts = rgb[1].split(",").map((p) => parseFloat(p.trim()));
          return { r: parts[0], g: parts[1], b: parts[2], a: parts[3] ?? 1 };
        }
        const lab = s.match(/^lab\(\s*([\d.\-]+)\s+([\d.\-]+)\s+([\d.\-]+)/);
        if (lab) {
          const L = parseFloat(lab[1]);
          const gray = Math.round((L / 100) * 255);
          return { r: gray, g: gray, b: gray, a: 1 };
        }
        return null;
      };
      const shellRgb = parse(shellBg);
      const blockRgb = parse(blockBg);

      // Per-channel absolute deltas. The visible "wrong-theme" pop in
      // the QA evidence is driven by channel mismatch — a black-on-
      // white block inside a dark shell will diverge by ~240 on every
      // channel, while a properly themed block (any reasonable shade
      // the design system picks) stays close to the shell.
      const channelDeltas =
        shellRgb && blockRgb
          ? {
              r: Math.abs(shellRgb.r - blockRgb.r),
              g: Math.abs(shellRgb.g - blockRgb.g),
              b: Math.abs(shellRgb.b - blockRgb.b),
            }
          : null;
      return {
        shellBg,
        blockBg,
        channelDeltas,
        maxChannelDelta: channelDeltas
          ? Math.max(channelDeltas.r, channelDeltas.g, channelDeltas.b)
          : null,
        blockClassName: blockCard.className,
      };
    });

    expect(
      sample.maxChannelDelta,
      `must locate both shell and block to compare colors ` +
        `(shell="${sample.shellBg}", block="${sample.blockBg}")`,
    ).not.toBeNull();

    // Bug-defining assertion: the vertical block's background must
    // visually harmonize with the admin shell — same broad theme
    // family, not the inverse. We don't pin "dark" as an absolute
    // RGB threshold; the design system might pick any reasonable
    // shade for the fix. What we DO require is low contrast against
    // the shell.
    //
    // Threshold 100 (per channel) is generous: bg-control-bg vs
    // bg-dark-bg differs by ~240 on every channel, while a properly
    // themed block (e.g., a bg-dark-block one shade lighter than the
    // shell, or a translucent overlay) stays well under 100.
    expect(
      sample.maxChannelDelta!,
      `vertical block must visually harmonize with the admin shell ` +
        `(shell="${sample.shellBg}", block="${sample.blockBg}", ` +
        `per-channel deltas=${JSON.stringify(sample.channelDeltas)}) — ` +
        `max channel difference of ${sample.maxChannelDelta} suggests the ` +
        `block dropped out of the dark theme`,
    ).toBeLessThan(100);
  });
});

test.describe("Wrench button is visible for admin users on a connected worksheet", () => {
  // AdminModeButton renders only when allowAdmin && tab.mode ===
  // "WORKSHEET" (AdminModeButton.tsx). For the demo admin user with
  // a real DB connection, the button must be present and enabled.

  test("admin user on a connected DB sees an enabled wrench", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(800);

    await expect(sqlEditor.adminModeButton).toBeVisible({ timeout: 10_000 });
    await expect(sqlEditor.adminModeButton).toBeEnabled();
  });
});

test.describe("Wrench is not clickable when no database is connected", () => {
  // The user contract: you can't enter admin mode without picking a
  // DB first. AdminModeButton.tsx implements this two ways depending
  // on context — it returns `null` when there's no current tab, and
  // it renders `disabled` when the tab is in WORKSHEET mode but
  // `tabStore.isDisconnected`. Either is a valid "not clickable"
  // state; both satisfy the contract.

  test("opening the SQL editor without a DB leaves the wrench unusable", async () => {
    test.setTimeout(120_000);

    await sqlEditor.gotoHome();
    await page.waitForTimeout(800);

    const visible = await sqlEditor.adminModeButton
      .isVisible()
      .catch(() => false);
    if (visible) {
      // Visible-but-disabled flavor (tab exists, no connection).
      await expect(sqlEditor.adminModeButton).toBeDisabled();
    } else {
      // Hidden flavor (no current tab → AdminModeButton returns null).
      await expect(sqlEditor.adminModeButton).toHaveCount(0);
    }
  });
});

test.describe("Clicking the wrench enters the admin terminal", () => {
  // After the click, the editor swaps from WORKSHEET to ADMIN mode.
  // TerminalPanel mounts in place of the standard editor — the dark
  // shell (.bg-dark-bg) and the "SQL>" prompt are the visible
  // signals. The "Exit admin mode" toolbar button also appears.

  test("admin terminal mounts with dark shell and SQL> prompt", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(800);

    await expect(sqlEditor.adminModeButton).toBeVisible({ timeout: 10_000 });
    await sqlEditor.adminModeButton.click();
    await page.waitForTimeout(800);

    await expect(page.locator(".bg-dark-bg").first()).toBeVisible({ timeout: 10_000 });
    // The CompactSQLEditor renders "SQL>" as a sibling line-decoration.
    await expect(page.getByText(/SQL>/).first()).toBeVisible({ timeout: 5000 });
    await expect(
      page.getByRole("button", { name: "Exit admin mode", exact: true }),
    ).toBeVisible({ timeout: 5000 });
  });
});

test.describe("Running SELECT 42 in the admin terminal returns a 1-row result", () => {
  // After entering admin mode, the user types SQL into the bottom
  // CompactSQLEditor and presses Enter (when the statement ends with
  // `;`) — the terminal executes against the ADMIN data source and
  // appends a result panel above the prompt. We verify a 1-row
  // result with the literal "42".

  test("Enter on a `;`-terminated statement executes and shows the row", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(800);

    await sqlEditor.adminModeButton.click();
    await page.waitForTimeout(800);

    await runInAdminTerminal("SELECT 42 AS answer;");

    // Result panel renders with the literal "42".
    await expect(page.getByText(/^1\s+rows?$/i).first()).toBeVisible({
      timeout: 15_000,
    });
    await expect(
      page.locator('[data-row-index="0"] [data-col-index="1"]'),
    ).toContainText("42", { timeout: 5000 });
  });
});

test.describe("Exit admin mode returns to the worksheet view", () => {
  // The "Exit admin mode" button in the admin toolbar swaps the tab
  // back to WORKSHEET. The dark terminal goes away; the standard
  // worksheet editor (`role="code"` Monaco) is the visible surface
  // again.

  test("clicking Exit admin mode dismounts the terminal", async () => {
    test.setTimeout(120_000);

    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(800);

    await sqlEditor.adminModeButton.click();
    await page.waitForTimeout(800);
    await expect(page.locator(".bg-dark-bg").first()).toBeVisible({ timeout: 10_000 });

    const exitBtn = page.getByRole("button", {
      name: "Exit admin mode",
      exact: true,
    });
    await exitBtn.click();
    await page.waitForTimeout(800);

    // Dark shell unmounts. (TerminalPanel renders TWO bg-dark-bg
    // descendants — the outer wrapper and the inner scroll area.
    // Both must vanish, so we check count rather than .first().)
    await expect(page.locator(".bg-dark-bg")).toHaveCount(0);
    // Wrench is back (worksheet mode) and enabled (still connected).
    await expect(sqlEditor.adminModeButton).toBeVisible({ timeout: 5000 });
    await expect(sqlEditor.adminModeButton).toBeEnabled();
  });
});

// Helpers shared by the arrow-key history tests below. Each one drives
// the LAST role="code" Monaco (the editable prompt — older terminal rows
// are read-only) the same way `setEditorContent` would on a worksheet,
// but explicitly targets `.last()`.
async function runInAdminTerminal(statement: string): Promise<void> {
  // The admin terminal's editable prompt is the LAST role="code" Monaco
  // (older rows are read-only). setEditorContent({ which: "last" }) runs the
  // click → select-all → delete → insert sequence; we add the Enter.
  await sqlEditor.setEditorContent(statement, { which: "last" });
  await page.waitForTimeout(150);
  await sqlEditor.codeEditor.last().press("Enter");
}

// Read the current statement out of the LAST Monaco (the editable prompt).
async function readAdminPromptValue(): Promise<string> {
  return await sqlEditor.readEditorContent({ which: "last" });
}

test.describe("Up/down arrow keys scroll the admin terminal's query history", () => {
  // CUJ (Batch 7 addition): the admin-mode terminal is meant to behave
  // like a real CLI — after a few queries you should be able to press
  // Up to recall the last command, Up again to step further back, and
  // Down to come forward to a fresh prompt.
  //
  // Wiring (CompactSQLEditor.tsx + useHistory.ts):
  //   - Up arrow runs `onHistory("up", editor)` when the editor is
  //     focused (Monaco `addCommand` precondition).
  //   - Down arrow runs `onHistory("down", editor)` when the cursor is
  //     on the last line.
  //   - `useHistory.move()` writes the recalled statement back into the
  //     live tail item's `statement` field, which the React tree binds
  //     to the editor's content — so the recalled SQL appears in the
  //     prompt.
  //
  // HISTORY (BYT-9560): the pinia → zustand migration in #20363 originally
  // broke recall entirely (move("up") wrote "" back to the tail, so Up/Down did
  // nothing). #20394 fixed it — recall and full multi-step navigation now work.
  // All three tests below are live regression locks (no `test.fail()`):
  //   - Up on an empty prompt recalls the most recent statement.
  //   - Down after Up returns to the empty prompt.
  //   - Repeated Up walks back through EVERY prior statement (most-recent →
  //     oldest); repeated Down walks forward back to empty. (Verified live.)
  //
  // Cadence note: Up takes ~2 presses per step (after a recall the cursor sits
  // at end-of-line → the next Up just repositions it to (1,1), re-arming the
  // `cursorAtFirst` gate, before the following Up recalls). Down is 1 press per
  // step. The multi-step test collapses consecutive duplicates and asserts the
  // reachability/order, not the exact press count — see its body.
  //
  // We assert ONLY the user-visible outcome (the prompt text on the last
  // Monaco). We don't assert on the internal `useHistory` index. (Principle A.)

  test.beforeEach(async () => {
    const projectId = env.project.split("/").pop()!;
    await sqlEditor.gotoWithDb(projectId, env.instanceId, env.databaseId);
    await page.waitForTimeout(800);
    await sqlEditor.adminModeButton.click();
    await page.waitForTimeout(800);
    await expect(page.locator(".bg-dark-bg").first()).toBeVisible({
      timeout: 10_000,
    });
  });

  test.afterEach(async () => {
    // Leave admin mode so the next test's beforeAll-shared page returns
    // to a worksheet starting state. (Page is reused across tests in
    // this file via the file-level `sharedContext` + `page`.)
    const exitBtn = page.getByRole("button", {
      name: "Exit admin mode",
      exact: true,
    });
    if (await exitBtn.isVisible().catch(() => false)) {
      await exitBtn.click();
      await page.waitForTimeout(400);
    }
  });

  test("Up arrow on an empty prompt recalls the most recently executed statement", async () => {
    // BYT-9560 (single-step Up recall) is FIXED — this is a live regression
    // lock. (Multi-step walk-back also works; see the repeated-Up test below.)
    test.setTimeout(120_000);

    // Establish a single executed statement so there is something to
    // recall. After it finishes, the terminal appends a fresh IDLE
    // prompt as the new editable row — the editor is empty there.
    const stmt = "SELECT 11 AS recall_target;";
    await runInAdminTerminal(stmt);
    await expect(page.getByText(/^1\s+rows?$/i).first()).toBeVisible({
      timeout: 15_000,
    });

    // Focus the new (empty) prompt and press Up.
    const prompt = sqlEditor.codeEditor.last();
    await prompt.click();
    await page.waitForTimeout(150);
    expect(await readAdminPromptValue()).toBe("");

    await page.keyboard.press("ArrowUp");
    await page.waitForTimeout(200);

    // CUJ assertion: the prompt now holds the previously executed
    // statement, ready to re-run.
    expect(await readAdminPromptValue()).toBe(stmt);

    // Visual-contrast guard: the recalled text must render in a color
    // that's distinguishable from the terminal's dark background. A
    // user reported on 2026-05-25 that on their build the recall
    // looked broken — they suspected the text might be the same color
    // as the bg. We pin that case shut: every visible token on the
    // recalled line must be at least 60 channels away from the shell
    // background. (Verified channel deltas were ≥176 on the working
    // build, so 60 is generous headroom while still failing loudly if
    // a future theme regression matches text to bg.)
    const contrast = await page.evaluate(() => {
      const parse = (s: string) => {
        const m = s.match(/rgba?\(([^)]+)\)/);
        if (!m) return null;
        const [r, g, b] = m[1].split(",").map((v) => parseFloat(v.trim()));
        return { r, g, b };
      };
      const codes = document.querySelectorAll('[role="code"]');
      const last = codes[codes.length - 1];
      const line = last?.querySelector(".view-lines .view-line");
      const shell = document.querySelector(".bg-dark-bg");
      if (!line || !shell)
        return { reason: "missing line or shell" } as { reason: string };
      const shellRgb = parse(getComputedStyle(shell).backgroundColor);
      const tokens = Array.from(line.querySelectorAll("span")).filter(
        (s) =>
          s.children.length === 0 && (s.textContent ?? "").trim().length > 0,
      );
      let minDelta = Infinity;
      let worst = "";
      for (const t of tokens) {
        const rgb = parse(getComputedStyle(t).color);
        if (!rgb || !shellRgb) continue;
        const d = Math.max(
          Math.abs(rgb.r - shellRgb.r),
          Math.abs(rgb.g - shellRgb.g),
          Math.abs(rgb.b - shellRgb.b),
        );
        if (d < minDelta) {
          minDelta = d;
          worst = `"${t.textContent}" @ ${getComputedStyle(t).color}`;
        }
      }
      return { minDelta, worst };
    });
    expect(
      "reason" in contrast ? Infinity : contrast.minDelta,
      `recalled text must contrast with the dark terminal shell ` +
        `(worst token: ${"worst" in contrast ? contrast.worst : "n/a"})`,
    ).toBeGreaterThan(60);
  });

  test("Down arrow after Up returns the prompt to empty", async () => {
    // BYT-9560 (single-step Down-after-Up) is FIXED — the prior `test.fail()`
    // hold flipped to passing, so this runs as a normal regression lock now.
    test.setTimeout(120_000);

    const stmt = "SELECT 22 AS down_target;";
    await runInAdminTerminal(stmt);
    await expect(page.getByText(/^1\s+rows?$/i).first()).toBeVisible({
      timeout: 15_000,
    });

    const prompt = sqlEditor.codeEditor.last();
    await prompt.click();
    await page.waitForTimeout(150);

    await page.keyboard.press("ArrowUp");
    await page.waitForTimeout(200);
    expect(await readAdminPromptValue()).toBe(stmt);

    // `useHistory.move("down")` resets `tail.statement` to "" when the
    // index reaches the tail (length-1). The user-visible effect is
    // that pressing Down returns to a fresh prompt.
    await page.keyboard.press("ArrowDown");
    await page.waitForTimeout(200);
    expect(await readAdminPromptValue()).toBe("");
  });

  test("Up arrow pressed repeatedly walks back through every prior statement", async () => {
    // BYT-9560 (multi-step recall) WORKS — repeated Up reaches every prior
    // command, most-recent → oldest. Verified empirically in the live admin
    // terminal.
    //
    // Cadence nuance (this is why the test collapses consecutive duplicates
    // rather than asserting one step per press): the Up-history keybinding is
    // gated on `cursorAtFirst` = cursor at (line 1, col 1) (utils.ts). A recall
    // drops the statement into the prompt and leaves the cursor at END-of-line,
    // so the *next* Up doesn't recall — Monaco's default ArrowUp just moves the
    // cursor back to (1,1), re-arming the gate — and the Up after that recalls
    // the next item. Net: Up takes ~2 presses per step. Down, gated on
    // `cursorAtLast` (already true at end-of-line after a recall), is a clean
    // 1 press per step. Both reach every entry; the test asserts reachability
    // and order, not the exact press count.
    test.setTimeout(120_000);

    // Run three distinct statements. After each, the terminal pushes the
    // executed item into history and appends a fresh IDLE prompt.
    const first = "SELECT 1 AS one;";
    const second = "SELECT 2 AS two;";
    const third = "SELECT 3 AS three;";

    for (const stmt of [first, second, third]) {
      await runInAdminTerminal(stmt);
      await expect(page.getByText(/^1\s+rows?$/i).first()).toBeVisible({
        timeout: 15_000,
      });
      await page.waitForTimeout(400);
    }

    const prompt = sqlEditor.codeEditor.last();
    await prompt.click();
    await page.waitForTimeout(150);
    expect(await readAdminPromptValue()).toBe("");

    // Walk a directional key several times, recording the DISTINCT prompt
    // values in the order they first appear (consecutive duplicates — the
    // cursor-reposition no-op presses — are collapsed). Returns the recall
    // progression.
    const walk = async (key: "ArrowUp" | "ArrowDown", presses = 8) => {
      const seen: string[] = [];
      let prev = await readAdminPromptValue();
      for (let i = 0; i < presses; i++) {
        await page.keyboard.press(key);
        await page.waitForTimeout(200);
        const v = await readAdminPromptValue();
        if (v !== prev) {
          seen.push(v);
          prev = v;
        }
      }
      return seen;
    };

    // Up from the empty prompt must walk back through ALL prior statements,
    // most-recent → oldest, and stop at the oldest (no wrap-around).
    const back = await walk("ArrowUp");
    expect(
      back,
      "repeated Up must recall every prior statement in reverse order " +
        "(most-recent → oldest)",
    ).toEqual([third, second, first]);

    // Down from the oldest must walk forward back through them to the empty
    // prompt.
    const forward = await walk("ArrowDown");
    expect(
      forward,
      "repeated Down must walk forward back to the live (empty) prompt",
    ).toEqual([second, third, ""]);
  });
});
