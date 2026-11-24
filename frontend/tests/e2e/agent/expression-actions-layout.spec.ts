import { writeFileSync } from "node:fs";
import type { Locator, Page } from "@playwright/test";
import {
  answer,
  expect,
  test,
} from "../agent-token-budget/token-budget.fixture";
import { loadTestEnv } from "../framework/env";

const expression = 'resource.column_name\n  == "email"';
const response = `Review this proposed column condition:\n\n\`\`\`cel\n${expression}\n\`\`\``;
const unsaved = "Applying replaces the current expression in this unsaved editor. It does not save the rule.";

function watchEffects(page: Page) {
  const calls: string[] = [];
  page.on("request", (request) => {
    const path = new URL(request.url()).pathname;
    const restWrite = ["PATCH", "PUT", "DELETE"].includes(request.method()) && path.startsWith("/v1/");
    const rpcWrite = request.method() === "POST" && /\/(?:Create|Update|Delete|BatchUpdate|Approve|Reject|Query|AdminExecute)[^/]*$/.test(path);
    if (restWrite || rpcWrite) calls.push(path);
  });
  return calls;
}

async function expectContained(element: Locator, container: Locator) {
  await expect(element).toBeVisible();
  await expect.poll(async () => {
    const child = await element.boundingBox();
    const parent = await container.boundingBox();
    return child !== null && parent !== null &&
      child.x >= parent.x - 1 &&
      child.x + child.width <= parent.x + parent.width + 1 &&
      child.y >= parent.y - 1 &&
      child.y + child.height <= parent.y + parent.height + 1;
  }).toBe(true);
}

for (const width of [320, 390]) {
  test(`a real CEL editor exposes two final unsaved expression actions at ${width}px`, async ({ page, provider }, testInfo) => {
    const env = loadTestEnv();
    await env.api.login(env.adminEmail, env.adminPassword);
    const policyName = `${env.project}/policies/masking_exemption`;
    const beforePolicy = await env.api.getPolicy(policyName);
    await page.setViewportSize({ width, height: 900 });
    await page.addInitScript(() => localStorage.setItem("bb.release", JSON.stringify({
      ignoreRemindModalTillNextRelease: true,
      nextCheckTs: Date.now() + 86_400_000,
    })));
    const effects = watchEffects(page);
    await page.goto(`${env.baseURL}/${env.project}/masking-exemption/create`);
    const mode = page.getByRole("radio", { name: /Use CEL Expression/ });
    await expect(mode).toBeEnabled();
    await mode.click();
    await expect(mode).toBeChecked();
    const editor = page.locator(".bb-risk-expr-editor");
    const ask = editor.getByRole("button", { name: "Ask Agent to help build this expression", exact: true });
    await expect(ask).toBeVisible();
    const originalEditor = await editor.textContent();
    provider.respond(answer(10, response));
    await ask.click();

    const agent = page.locator("[data-agent-window]");
    await expect(agent).toBeVisible();
    const generated = agent.locator("[data-agent-answer]").filter({ hasText: "Review this proposed column condition:" });
    const pre = generated.locator("pre");
    await expect(pre).toContainText(expression);
    const draft = generated.getByRole("group", { name: "Apply generated expression", exact: true });
    const hint = draft.getByText(unsaved, { exact: true });
    const actions = draft.locator("[data-agent-action-group]");
    const selector = actions.getByRole("button", { name: "Apply to selector", exact: true });
    const raw = actions.getByRole("button", { name: "Apply as raw CEL", exact: true });
    await expect(selector).toBeEnabled();
    await expect(raw).toBeEnabled();
    await expect(actions.getByRole("button")).toHaveCount(2);
    await actions.scrollIntoViewIfNeeded();
    const geometry = await actions.evaluate((node) => {
      const bounds = node.getBoundingClientRect();
      return {
        final: node.parentElement?.lastElementChild === node,
        top: bounds.top,
        right: bounds.right,
        wrap: getComputedStyle(node).flexWrap,
        buttons: Array.from(node.querySelectorAll("button"), (button) => {
          const rect = button.getBoundingClientRect();
          return {
            top: Math.round(rect.top),
            right: rect.right,
            overflow: button.scrollWidth - button.clientWidth,
          };
        }),
      };
    });
    expect(geometry.final).toBe(true);
    expect(geometry.wrap).toBe("wrap");
    const hintBounds = (await hint.boundingBox())!;
    expect(geometry.top).toBeGreaterThanOrEqual(hintBounds.y + hintBounds.height - 1);
    const rows = new Map<number, number>();
    for (const button of geometry.buttons) {
      expect(button.overflow).toBeLessThanOrEqual(1);
      rows.set(button.top, Math.max(rows.get(button.top) ?? 0, button.right));
    }
    for (const right of rows.values()) {
      expect(Math.abs(right - geometry.right)).toBeLessThanOrEqual(1);
    }
    for (const element of [pre, hint, selector, raw]) {
      await expectContained(element, agent);
    }
    expect(await generated.evaluate((node) => node.scrollWidth - node.clientWidth)).toBeLessThanOrEqual(1);
    const screenshots = [];
    for (const [surface, file, label] of [
      [agent, `expression-actions-window-${width}.png`, "CEL proposal and two unsaved actions"],
      [generated, `expression-actions-surface-${width}.png`, "Generated CEL with selector and raw action row"],
    ] as const) {
      await surface.screenshot({ path: testInfo.outputPath(file) });
      screenshots.push({ file, label });
      writeFileSync(testInfo.outputPath("screenshots.json"), JSON.stringify({ title: testInfo.title, width, screenshots }, null, 2));
    }
    expect(await editor.textContent()).toBe(originalEditor);
    expect(await env.api.getPolicy(policyName)).toEqual(beforePolicy);
    expect(effects).toEqual([]);
    expect(provider.calls).toBe(1);
  });
}
