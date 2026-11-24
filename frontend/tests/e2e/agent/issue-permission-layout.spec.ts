import { writeFileSync } from "node:fs";
import type { Locator, Page } from "@playwright/test";
import {
  answer,
  expect,
  test,
} from "../agent-token-budget/token-budget.fixture";
import { loadTestEnv } from "../framework/env";
import { loginBrowserContext } from "../framework/sign-in";

test.setTimeout(90_000);

async function expectContained(element: Locator, container: Locator) {
  await expect(container).toBeVisible();
  await expect(element).toBeVisible();
  await expect.poll(() => element.boundingBox()).not.toBeNull();
  const bounds = await element.boundingBox();
  const parent = await container.boundingBox();
  expect(bounds).not.toBeNull();
  expect(parent).not.toBeNull();
  expect(bounds!.x).toBeGreaterThanOrEqual(parent!.x - 1);
  expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(parent!.x + parent!.width + 1);
  expect(bounds!.y).toBeGreaterThanOrEqual(parent!.y - 1);
  expect(bounds!.y + bounds!.height).toBeLessThanOrEqual(parent!.y + parent!.height + 1);
  expect(await element.evaluate((node) => node.scrollWidth - node.clientWidth)).toBeLessThanOrEqual(1);
}

function protectedDecisions(page: Page) {
  const requests: string[] = [];
  page.on("request", (request) => {
    const path = new URL(request.url()).pathname;
    if (request.method() === "POST" && /\/(?:ApproveIssue|RejectIssue|UpdateIssue|BatchRunTasks)$/.test(path)) {
      requests.push(path);
    }
  });
  return requests;
}

for (const width of [320, 390]) {
  test(`Issue decision previews retain consequences and final actions at ${width}px without approving`, async ({ page, context, provider }, testInfo) => {
    const env = loadTestEnv();
    await env.api.login(env.adminEmail, env.adminPassword);
    const previous = (await env.api.getSetting("WORKSPACE_APPROVAL"))?.value as {
      workspaceApproval?: { rules?: unknown[] };
    } | undefined;
    if (!previous?.workspaceApproval) {
      throw new Error("The licensed E2E server must have its owned approval fixture.");
    }
    const projectId = `agent-impact-${width}-${Date.now()}`;
    const project = await env.api.createProject(projectId, "Agent permission preview fixture");
    let configured = false;
    try {
      await env.api.upsertSetting("WORKSPACE_APPROVAL", {
        workspaceApproval: {
          ...previous.workspaceApproval,
          rules: [{
            source: "REQUEST_ROLE",
            condition: { expression: `resource.project_id == "${projectId}"` },
            template: {
              title: "Owned role request approval",
              flow: { roles: ["roles/workspaceAdmin"] },
            },
          }, ...(previous.workspaceApproval.rules ?? [])],
        },
      }, "value.workspace_approval");
      configured = true;
      const title = "Review this temporary project role request before granting access";
      const issue = await env.api.createRoleGrantIssue(project.name, {
        role: "roles/projectViewer",
        user: `users/${env.adminEmail}`,
      }, title);
      await expect.poll(async () => (await env.api.getIssue(issue.name)).approvalStatus).toBe("PENDING");

      await page.setViewportSize({ width, height: 900 });
      await loginBrowserContext(context, env.baseURL, env.adminEmail, env.adminPassword);
      await page.addInitScript(() => localStorage.setItem("bb.release", JSON.stringify({
        ignoreRemindModalTillNextRelease: true,
        nextCheckTs: Date.now() + 86_400_000,
      })));
      await page.goto(`${env.baseURL}/landing`);
      await page.getByRole("button", { name: "Agent", exact: true }).click();
      const agent = page.locator("[data-agent-window]");
      await expect(agent).toBeVisible();
      const requests = protectedDecisions(page);
      const screenshots: { file: string; label: string }[] = [];
      for (const action of ["ApproveIssue", "RejectIssue"] as const) {
        const operationId = `bytebase.v1.IssueService.${action}`;
        const suffix = action === "ApproveIssue" ? "approval" : "rejection";
        for (const [name, args] of [
          ["search_api", { operationId }],
          ["call_api", { operationId, body: { name: issue.name } }],
        ] as const) {
          provider.respond({ toolCalls: [{ id: `issue-${suffix}-${name}`, name, arguments: JSON.stringify(args) }], stopReason: "TURN_COMPLETED" });
        }
        const completed = `The ${suffix} request was denied without changing the Issue.`;
        provider.respond(answer(10, completed));
        await agent.locator("textarea").last().fill(`Prepare this exact ${suffix} request for review: ${issue.name}. Do not execute it without permission.`);
        await agent.getByRole("button", { name: "Send", exact: true }).click();
        const allow = agent.getByRole("button", { name: "Allow once", exact: true });
        const deny = agent.getByRole("button", { name: "Deny", exact: true });
        await expect(allow).toBeEnabled();
        await expect(agent.getByText(title, { exact: true })).toBeVisible();
        const consequence = action === "ApproveIssue"
          ? "This is the final approval step. Approval may immediately grant the requested project role, without a rollout."
          : "Rejecting this issue stops its current approval flow. The rejection must be addressed before approval can continue.";
        await expect(agent.getByText(consequence, { exact: true })).toBeVisible();
        const preview = agent.locator("pre").filter({ hasText: issue.name });
        await expect(preview).toContainText(`"name": "${issue.name}"`);
        const actions = allow.locator("xpath=ancestor::*[@data-agent-action-group][1]");
        await expect(actions).toHaveCSS("flex-wrap", "wrap");
        await expect.poll(async () => {
          const group = await actions.boundingBox();
          const last = await actions.getByRole("button").last().boundingBox();
          return group !== null && last !== null && Math.abs(last.x + last.width - group.x - group.width) <= 1;
        }).toBe(true);
        for (const element of [preview, allow, deny]) {
          await expectContained(element, agent);
        }
        expect(requests).toEqual([]);
        expect((await env.api.getIssue(issue.name)).approvalStatus).toBe("PENDING");
        const file = `issue-${suffix}-impact-${width}.png`;
        await agent.screenshot({ path: testInfo.outputPath(file) });
        screenshots.push({ file, label: `Issue ${suffix} consequences and exact request` });
        writeFileSync(testInfo.outputPath("screenshots.json"), JSON.stringify({ title: testInfo.title, width, screenshots }, null, 2));
        await deny.click();
        await expect(agent.getByText(completed, { exact: true })).toBeVisible();
        await expect(allow).toBeHidden();
        expect(requests).toEqual([]);
        expect((await env.api.getIssue(issue.name)).approvalStatus).toBe("PENDING");
      }
      expect(provider.calls).toBe(6);
    } finally {
      if (configured) {
        await env.api.upsertSetting("WORKSPACE_APPROVAL", previous, "value.workspace_approval");
      }
      await env.api.deleteProject(project.name);
    }
  });
}
