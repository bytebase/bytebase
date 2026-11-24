import { writeFileSync } from "node:fs";
import type { Locator, Page, TestInfo } from "@playwright/test";
import {
  answer,
  expect,
  test,
} from "../agent-token-budget/token-budget.fixture";
import { loadTestEnv } from "../framework/env";
import { waitForPlanChecksDone } from "../plan-detail/plan-helpers";
import { PlanDetailPage } from "../plan-detail/plan-detail.page";

test.setTimeout(180_000);

type Env = ReturnType<typeof loadTestEnv>;

async function createSqlFixFixture(env: Env) {
  await env.api.login(env.adminEmail, env.adminPassword);
  const id = `sql-fix-layout-${Date.now()}`;
  const table = id.replaceAll("-", "_");
  const sql = [
    "CREATE TABLE",
    `public.${table} (`,
    "v TEXT",
    ");",
  ].join("\n");
  const replacement = [
    "CREATE TABLE",
    `public.${table} (`,
    "v TEXT",
    "NOT NULL",
    ");",
  ].join("\n");
  const originalTag = await env.api.getPolicy(`${env.project}/policies/tag`);
  let plan = "";
  let reviewConfig = "";
  const cleanup = async () => {
    const errors: unknown[] = [];
    for (const restore of [
      async () => {
        if (plan) await env.api.closePlan(plan);
      },
      async () => {
        if (originalTag) {
          await env.api.upsertPolicy(env.project, "tag", originalTag);
        } else {
          await env.api.deletePolicy(env.project, "tag");
        }
      },
      async () => {
        if (reviewConfig) await env.api.deleteReviewConfig(reviewConfig);
      },
    ]) {
      try {
        await restore();
      } catch (error) {
        errors.push(error);
      }
    }
    if (errors.length) {
      throw new AggregateError(errors, "SQL-fix layout fixture cleanup failed");
    }
  };

  try {
    reviewConfig = (
      await env.api.upsertReviewConfig(
        id,
        "SQL-fix action layout",
        [{ type: "COLUMN_NO_NULL", level: "WARNING", engine: "POSTGRES" }],
        true
      )
    ).name;
    await env.api.upsertReviewConfigTag(env.project, reviewConfig);
    const sheet = await env.api.createSheet(env.project, sql);
    plan = (
      await env.api.createPlan(env.project, "SQL-fix action layout", [
        { id: "sql-fix-layout-spec", targets: [env.database], sheet },
      ])
    ).name;
    await env.api.runPlanChecks(plan);
    await waitForPlanChecksDone(env.api, plan);
    const checks = await env.api.getPlanCheckRun(plan);
    expect(checks.results).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ status: "WARNING", title: "COLUMN_NO_NULL" }),
      ])
    );
    return { plan, sheet, sql, replacement, cleanup };
  } catch (error) {
    await cleanup();
    throw error;
  }
}

function captureScreenshots(testInfo: TestInfo, width: number) {
  const screenshots: { file: string; label: string }[] = [];
  return async (surface: Locator, file: string, label: string) => {
    await surface.screenshot({ path: testInfo.outputPath(file) });
    screenshots.push({ file, label });
    writeFileSync(
      testInfo.outputPath("screenshots.json"),
      JSON.stringify({ title: testInfo.title, width, screenshots }, null, 2)
    );
  };
}

async function assertDialogActionLayout(page: Page, dialog: Locator) {
  const apply = dialog.getByRole("button", {
    name: "Apply to draft",
    exact: true,
  });
  const actions = apply.locator(
    "xpath=ancestor::*[@data-agent-action-group][1]"
  );
  const geometry = await actions.evaluate((node) => {
    const bounds = node.getBoundingClientRect();
    const previous = node.previousElementSibling?.getBoundingClientRect();
    const style = getComputedStyle(node);
    const buttons = Array.from(node.querySelectorAll("button"));
    return {
      final: node.parentElement?.lastElementChild === node,
      top: bounds.top,
      right: bounds.right,
      bottom: bounds.bottom,
      previousBottom: previous?.bottom,
      flexWrap: style.flexWrap,
      rowGap: style.rowGap,
      columnGap: style.columnGap,
      buttons: buttons.map((button) => {
        const rect = button.getBoundingClientRect();
        return {
          left: rect.left,
          right: rect.right,
          top: rect.top,
          bottom: rect.bottom,
          width: rect.width,
          overflow: button.scrollWidth - button.clientWidth,
        };
      }),
    };
  });
  const dialogBounds = await dialog.boundingBox();
  expect(dialogBounds).not.toBeNull();
  const viewport = page.viewportSize()!;
  expect(dialogBounds!.x).toBeGreaterThanOrEqual(0);
  expect(dialogBounds!.x + dialogBounds!.width).toBeLessThanOrEqual(
    viewport.width + 1
  );
  expect(dialogBounds!.y).toBeGreaterThanOrEqual(0);
  expect(dialogBounds!.y + dialogBounds!.height).toBeLessThanOrEqual(
    viewport.height + 1
  );
  expect(geometry.final).toBe(true);
  expect(geometry.flexWrap).toBe("wrap");
  expect(geometry.rowGap).toBe("8px");
  expect(geometry.columnGap).toBe("8px");
  expect(geometry.previousBottom).toBeDefined();
  expect(geometry.top).toBeGreaterThanOrEqual(geometry.previousBottom! - 1);
  expect(geometry.bottom).toBeLessThanOrEqual(
    dialogBounds!.y + dialogBounds!.height + 1
  );
  const last = geometry.buttons.at(-1)!;
  expect(Math.abs(last.right - geometry.right)).toBeLessThanOrEqual(1);
  for (const button of geometry.buttons) {
    expect(button.left).toBeGreaterThanOrEqual(dialogBounds!.x - 1);
    expect(button.right).toBeLessThanOrEqual(geometry.right + 1);
    expect(button.top).toBeGreaterThanOrEqual(geometry.top - 1);
    expect(button.bottom).toBeLessThanOrEqual(geometry.bottom + 1);
    expect(button.width).toBeGreaterThan(0);
    expect(button.overflow).toBeLessThanOrEqual(1);
  }
  if (geometry.buttons.length === 3) {
    const rows = new Map<number, number>();
    for (const button of geometry.buttons) {
      const center = Math.round((button.top + button.bottom) / 2);
      rows.set(center, Math.max(rows.get(center) ?? 0, button.right));
    }
    expect(rows.size).toBeGreaterThan(1);
    for (const right of rows.values()) {
      expect(Math.abs(right - geometry.right)).toBeLessThanOrEqual(1);
    }
  }
}

async function waitForRenderedDiff(dialog: Locator) {
  const diff = dialog.locator(".monaco-diff-editor");
  await expect(diff).toBeVisible();
  const lines = diff.locator(".modified .view-lines");
  await expect(lines).toBeVisible();
  await expect(lines).toContainText(/CREATE\s+TABLE/);
  await expect(lines).toContainText(/NOT\s+NULL/);
  await expect(lines.locator(".view-line").filter({
    hasText: /^NOT\s+NULL$/,
  })).toBeVisible();
}

for (const width of [320, 390]) {
  test(`runtime SQL-fix ready and stale actions fit at ${width}px`, async ({
    page,
    provider,
  }, testInfo) => {
    const env = loadTestEnv();
    const fixture = await createSqlFixFixture(env);
    const capture = captureScreenshots(testInfo, width);
    try {
      await page.setViewportSize({ width, height: 900 });
      await page.addInitScript(() => {
        localStorage.setItem("bb.language", "en-US");
        localStorage.setItem("bb.release", JSON.stringify({
          ignoreRemindModalTillNextRelease: true,
          nextCheckTs: Date.now() + 86_400_000,
        }));
      });
      provider.respond(answer(
        10,
        `Add NOT NULL to the proposed column. No statements were executed.\n\n\`\`\`sql replacement\n${fixture.replacement}\n\`\`\``
      ));
      const planPage = new PlanDetailPage(page, env.baseURL);
      await planPage.goto(
        env.project.split("/").at(-1)!,
        fixture.plan.split("/").at(-1)!
      );
      await planPage.dismissModals();
      await planPage.expandSection("Changes");
      const changes = page.locator("#plan-phase-changes");
      const editSql = changes
        .getByRole("button", { name: "Edit", exact: true })
        .filter({ hasText: /^Edit$/ });
      await expect(editSql).toBeEnabled();
      await changes.getByRole("button", { name: "Checks", exact: true }).click();
      const checks = page.getByRole("dialog", { name: "Checks", exact: true });
      await expect(checks).toBeVisible();
      const effectfulCalls: string[] = [];
      page.on("request", (request) => {
        const path = new URL(request.url()).pathname;
        if (
          request.method() === "POST" &&
          /\/(?:Create|Update|Delete|BatchRun|BatchCancel|Approve|Reject|Query)[^/]*$/.test(path)
        ) {
          effectfulCalls.push(path);
        }
      });
      await checks.getByRole("button", { name: "Ask Agent", exact: true }).click();
      const agent = page.locator("[data-agent-window]");
      const review = agent.locator("[data-agent-action-review-sql-fix]");
      await expect(review).toBeEnabled({ timeout: 30_000 });
      await review.click();
      const dialog = page.getByRole("dialog", {
        name: "Review SQL fix",
        exact: true,
      });
      await expect(dialog).toBeVisible();
      await waitForRenderedDiff(dialog);
      const apply = dialog.getByRole("button", {
        name: "Apply to draft",
        exact: true,
      });
      await expect(apply).toBeEnabled();
      await expect(dialog.getByRole("alert")).toHaveCount(0);
      await assertDialogActionLayout(page, dialog);
      await capture(dialog, `sql-fix-ready-${width}.png`, "Runtime SQL-fix review ready to apply to unsaved draft");
      await dialog.getByRole("button", { name: "Cancel", exact: true }).click();

      await agent.getByRole("button", { name: "Minimize", exact: true }).click();
      await checks.getByRole("button", { name: "Close", exact: true }).click();
      await expect(checks).toBeHidden();
      await planPage.expandSection("Changes");
      await editSql.click();
      const editor = changes.locator(".monaco-editor").first();
      await expect(editor).toBeVisible();
      await editor.click();
      await page.keyboard.press("Control+a");
      const changedSql = `${fixture.sql}\n-- Changed locally after diagnosis`;
      await page.keyboard.insertText(changedSql);
      await expect(editor.locator(".view-lines")).toContainText("Changed locally after diagnosis");
      await expect(changes.getByRole("button", { name: "Save", exact: true })).toBeEnabled();

      await page
        .getByRole("button", { name: "Agent", exact: true })
        .and(page.locator("[data-agent-window]"))
        .click();
      await expect(review).toBeEnabled();
      await review.click();
      await expect(dialog).toBeVisible();
      await waitForRenderedDiff(dialog);
      await expect(dialog.getByRole("alert")).toContainText(
        "SQL changed since this fix was prepared."
      );
      await expect(apply).toBeDisabled();
      await expect(dialog.getByRole("button", {
        name: "Review current SQL",
        exact: true,
      })).toBeEnabled();
      await expect(dialog.locator("[data-agent-action-group] button")).toHaveCount(3);
      await assertDialogActionLayout(page, dialog);
      await capture(dialog, `sql-fix-stale-${width}.png`, "Stale SQL-fix review with Review current SQL, Cancel, and disabled Apply to draft");
      await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
      expect(provider.calls).toBe(1);
      expect(effectfulCalls).toEqual([]);
      const persisted = await env.api.getPlan(fixture.plan);
      expect(persisted.specs?.[0]?.changeDatabaseConfig?.sheet).toBe(fixture.sheet);
    } finally {
      await fixture.cleanup();
    }
  });
}
