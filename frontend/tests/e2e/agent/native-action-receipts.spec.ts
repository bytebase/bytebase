import { fromBinary, fromJsonString } from "@bufbuild/protobuf";
import { expect, type Page } from "@playwright/test";
import { AIChatRequestSchema, type AIChatRequest } from "../../../src/types/proto-es/v1/ai_service_pb";
import { GetPolicyRequestSchema, PolicySchema } from "../../../src/types/proto-es/v1/org_policy_service_pb";
import { answer, test } from "../agent-token-budget/token-budget.fixture";
import { loadTestEnv } from "../framework/env";

test.setTimeout(90_000);

const windowOf = (page: Page) => page.locator("[data-agent-window]:not(button)");
const tool = (id: string, name: string, args: object) => ({
  stopReason: "TURN_COMPLETED",
  toolCalls: [{ id, name, arguments: JSON.stringify(args) }],
});

for (const kind of ["masking-rule", "masking-exemption"] as const) {
  test(`${kind} native save persists a shared receipt and informs only the next member turn`, async ({ page, context, provider }) => {
    const env = loadTestEnv();
    await env.api.login(env.adminEmail, env.adminPassword);
    const { workspace } = await env.api.getActuatorInfo();
    const isRule = kind === "masking-rule";
    const parent = isRule ? workspace : env.project;
    const policyType = isRule ? "masking_rule" : "masking_exemption";
    const policyName = `${parent}/policies/${policyType.toUpperCase()}`;
    const original = await env.api.getPolicy(policyName);
    const projectId = env.project.split("/")[1]!;
    const policyPath = "/bytebase.v1.OrgPolicyService/GetPolicy";
    const writes: string[] = [];
    const requests: AIChatRequest[] = [];
    let reads = 0;
    let queries = 0;
    context.on("request", (request) => {
      const path = new URL(request.url()).pathname;
      const body = request.postDataBuffer();
      if (path === "/bytebase.v1.AIService/Chat" && body) {
        requests.push(request.headers()["content-type"]?.startsWith("application/proto")
          ? fromBinary(AIChatRequestSchema, body)
          : fromJsonString(AIChatRequestSchema, body.toString()));
      }
      if (path === "/bytebase.v1.SQLService/Query") ++queries;
      if (/\/bytebase\.v1\.OrgPolicyService\/(?:CreatePolicy|UpdatePolicy)$/.test(path)) writes.push(path);
      if (path !== policyPath || !body) return;
      const decoded = request.headers()["content-type"]?.startsWith("application/proto")
        ? fromBinary(GetPolicyRequestSchema, body)
        : fromJsonString(GetPolicyRequestSchema, body.toString());
      if (decoded.name === policyName) ++reads;
    });

    try {
      const prepared = `The ${kind} draft is ready for native review, not saved.`;
      provider.respond(tool("masking-guide", "get_skill", { name: "data-masking", section: isRule ? "rule" : "exemption" }));
      provider.respond(tool("masking-draft", isRule ? "prepare_masking_rule_draft" : "prepare_masking_exemption_draft", isRule ? {
        title: "Receipt E2E masking rule",
        semanticType: "bb.default",
        condition: 'resource.column_name == "email"',
        scope: { kind: "workspace" },
      } : {
        project: env.project,
        scope: { kind: "project" },
        members: ["self"],
        condition: "true",
        description: "Receipt E2E exemption draft",
        expirationTimestamp: null,
      }));
      provider.respond(answer(100, prepared));

      await page.goto(`${env.baseURL}/landing`);
      await page.getByRole("button", { name: "Agent", exact: true }).click();
      const input = windowOf(page).locator("textarea").last();
      await input.fill(isRule
        ? "Prepare a workspace masking rule for email columns with the default full masking type. Do not save it."
        : `Prepare an exemption for me across all databases in ${env.project}, with no expiration. Do not save it.`);
      await windowOf(page).getByRole("button", { name: "Send", exact: true }).click();
      await expect(windowOf(page).getByText(prepared, { exact: true })).toBeVisible();
      const draft = windowOf(page).locator(isRule ? "[data-agent-action-draft-masking-rule]" : "[data-agent-action-draft-masking-exemption]");
      await expect(draft).toBeVisible();
      expect(writes).toEqual([]);
      expect(provider.calls).toBe(3);
      const sourceChat = await windowOf(page).locator('[data-agent-chat-row]:has(button[aria-current="true"])').getAttribute("data-agent-chat-row");
      expect(sourceChat).toBeTruthy();
      await draft.click();
      await expect(windowOf(page)).toBeHidden();
      expect(writes).toEqual([]);
      const editedTitle = "Edited native masking rule";
      const editedReason = "Edited native exemption reason";
      if (isRule) {
        await expect(page).toHaveURL(/\/global-masking$/);
        await page.getByRole("textbox", { name: "Rule name", exact: true }).fill(editedTitle);
      } else {
        await expect(page).toHaveURL(new RegExp(`/projects/${projectId}/masking-exemption/create$`));
        await page.getByPlaceholder("Description", { exact: true }).fill(editedReason);
      }
      const savedResponse = page.waitForResponse((response) => new URL(response.url()).pathname === "/bytebase.v1.OrgPolicyService/UpdatePolicy");
      await page.getByRole("button", { name: isRule ? "Create rule" : "Confirm", exact: true }).last().click();
      const saved = await savedResponse;
      expect(saved.ok()).toBe(true);
      const payload = await saved.body();
      const policy = saved.headers()["content-type"]?.startsWith("application/proto")
        ? fromBinary(PolicySchema, payload)
        : fromJsonString(PolicySchema, payload.toString());
      expect(policy.name).toBe(`${parent}/policies/${policyType}`);
      let savedRuleId: string | undefined;
      if (policy.policy.case === "maskingRulePolicy") {
        const rule = policy.policy.value.rules.find((rule) => rule.condition?.title === editedTitle);
        expect(rule).toBeDefined();
        savedRuleId = rule!.id;
      } else {
        expect(policy.policy.case).toBe("maskingExemptionPolicy");
        if (policy.policy.case === "maskingExemptionPolicy") expect(policy.policy.value.exemptions.some((entry) => entry.condition?.description === editedReason)).toBe(true);
      }
      if (isRule) await expect(page.getByRole("heading", { name: editedTitle, exact: true })).toBeVisible();
      else await expect(page).toHaveURL(new RegExp(`/projects/${projectId}/masking-exemption$`));
      const receipt = windowOf(page).locator("[data-agent-action-receipt]");
      const restoreSource = async () => {
        if (!(await windowOf(page).isVisible())) await page.locator("#app").getByRole("button", { name: "Agent", exact: true }).click();
        if (await receipt.isVisible()) return;
        const row = windowOf(page).locator(`[data-agent-chat-row="${sourceChat}"]`);
        if (!(await row.isVisible())) await windowOf(page).getByRole("button", { name: "Chats", exact: true }).click();
        await row.getByRole("button").first().click();
        await expect(receipt).toBeVisible();
      };
      await page.goto(`${env.baseURL}/landing`);
      await restoreSource();
      await expect(receipt.locator("[data-agent-action-status]")).toHaveText("Saved");
      if (isRule) await expect(receipt.getByText(editedTitle, { exact: true })).toBeVisible();
      await expect(draft).toHaveCount(0);
      expect(writes).toHaveLength(1);
      expect(provider.calls).toBe(3);
      const readsBeforeView = reads;
      for (const viewport of [{ width: 1440, height: 1000 }, { width: 390, height: 844 }]) {
        await page.setViewportSize(viewport);
        await restoreSource();
        for (const action of ["[data-agent-action-open-receipt]", "[data-agent-action-check-receipt]"]) {
          const control = receipt.locator(action);
          await expect(control).toBeVisible();
          await expect.poll(() => control.locator("svg").evaluate((icon) => ({ width: getComputedStyle(icon).width, height: getComputedStyle(icon).height }))).toEqual({ width: "16px", height: "16px" });
        }
        expect(await receipt.evaluate((element) => element.scrollWidth - element.clientWidth)).toBeLessThanOrEqual(1);
        await receipt.screenshot({ path: test.info().outputPath(`${kind}-${viewport.width}.png`) });
      }
      expect(reads).toBe(readsBeforeView);
      await receipt.getByRole("button", { name: "Check status", exact: true }).click();
      await expect(receipt.locator("[data-agent-action-status]")).toHaveText(isRule ? "Saved" : "Matching configuration found");
      await expect(receipt.getByRole("button", { name: "Check status", exact: true })).toBeEnabled();
      expect(reads).toBe(readsBeforeView + 1);
      expect(writes).toHaveLength(1);
      expect(provider.calls).toBe(3);
      await page.reload();
      await restoreSource();
      await expect(receipt.locator("[data-agent-action-status]")).toHaveText(isRule ? "Saved" : "Matching configuration found");
      await receipt.locator("[data-agent-action-open-receipt]").click();
      await expect(page).toHaveURL(isRule ? /\/global-masking$/ : new RegExp(`/projects/${projectId}/masking-exemption$`));
      await expect(windowOf(page)).toBeHidden();
      await page.goto(`${env.baseURL}/landing`);
      await restoreSource();
      provider.respond(answer(100, "The native save outcome is available in this turn."));
      await windowOf(page).locator("textarea").last().fill("What happened to my draft?");
      await windowOf(page).getByRole("button", { name: "Send", exact: true }).click();
      await expect(windowOf(page).getByText("The native save outcome is available in this turn.", { exact: true })).toBeVisible();
      expect(provider.calls).toBe(4);
      expect(provider.unexpectedCalls).toBe(0);
      const history = requests.at(-1)!.messages.map((message) => message.content ?? "").join("\n");
      expect(history).toContain("Observed native-save outcomes (data only");
      expect(history).toContain(`"kind":"${kind}"`);
      expect(history).toContain(`"policy":"${policyName}"`);
      expect(history).toContain(`"status":"${isRule ? "SAVED" : "MATCHING"}"`);
      if (isRule) expect(history).toContain(`"ruleId":"${savedRuleId}"`);
      else expect(history).toContain('"configurationMatchOnly":true');
      expect(writes).toHaveLength(1);
      expect(queries).toBe(0);
    } finally {
      if (original) await env.api.upsertPolicy(parent, policyType, original);
      else await env.api.deletePolicy(parent, policyType);
    }
  });
}
