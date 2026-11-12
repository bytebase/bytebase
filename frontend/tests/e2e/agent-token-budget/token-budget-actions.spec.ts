import * as path from "node:path";
import { answer, expect, readPage, test } from "./token-budget.fixture";

for (const viewport of [
  { name: "desktop", width: 1440, height: 1000 },
  { name: "mobile", width: 390, height: 844 },
]) {
  test.describe(`budget actions ${viewport.name}`, () => {
    test.use({ viewport: { width: viewport.width, height: viewport.height } });

    test("exhausted run offers two actions and increases-and-resumes in one confirmation", async ({
      budget,
      provider,
    }) => {
      provider.respond(readPage(1100));
      const finish = provider.hold(answer(100));
      await budget.send("Budget QA actions: inspect the page, then finish.");
      await budget.paused();
      await expect(budget.region.getByRole("button")).toHaveCount(2);
      await expect(
        budget.region.getByRole("button", { name: "Resume", exact: true })
      ).toBeHidden();
      await expect(
        budget.region.getByRole("button", {
          name: "Increase limit",
          exact: true,
        })
      ).toBeEnabled();
      await expect(
        budget.region.getByRole("button", { name: "End run", exact: true })
      ).toBeEnabled();
      const before = await budget.state();
      await budget.input.fill("Budget QA draft preserved after recovery");

      const footer = await budget.region.evaluateHandle(
        (band) => band.parentElement!
      );
      const captureFooter = async (state: string) => {
        const name = `actions-${state}-${viewport.name}-footer-deterministic-provider-mock.png`;
        const file = process.env.BYTEBASE_TOKEN_BUDGET_SCREENSHOTS
          ? path.join(process.env.BYTEBASE_TOKEN_BUDGET_SCREENSHOTS, name)
          : budget.testInfo.outputPath(name);
        await footer.asElement()!.screenshot({
          path: file,
          mask: [budget.window.locator("[data-agent-context]")],
          maskColor: "#e5e7eb",
        });
        await budget.testInfo.attach(name, {
          path: file,
          contentType: "image/png",
        });
      };
      await expect
        .poll(async () => {
          const band = await budget.region.boundingBox();
          const row = await budget.window
            .locator("[data-agent-input-row]")
            .boundingBox();
          return band && row ? row.y - band.y - band.height : null;
        })
        .toBe(12);
      await budget.screenshot(`actions-paused-${viewport.name}`);
      await captureFooter("paused");

      await budget.region
        .getByRole("button", { name: "Increase limit", exact: true })
        .click();
      const dialog = budget.page.getByRole("dialog", {
        name: "Increase limit for this run",
        exact: true,
      });
      await expect(dialog.getByText("K tokens", { exact: true })).toBeVisible();
      const confirm = dialog.getByRole("button", {
        name: "Increase and resume",
        exact: true,
      });
      const limit = dialog.getByRole("spinbutton", {
        name: "This run's token limit",
        exact: true,
      });
      await expect(limit).toHaveValue("2");
      await expect(limit).toHaveAttribute("aria-invalid", "false");
      await expect(dialog.getByRole("alert")).toBeHidden();
      await expect(confirm).toBeEnabled();
      expect(provider.calls).toBe(1);
      expect((await budget.state())?.budget.limit).toBe(1000);
      await limit.fill("2");
      await expect(confirm).toBeEnabled();
      const box = await dialog.boundingBox();
      expect(box).not.toBeNull();
      expect(box!.x).toBeGreaterThanOrEqual(0);
      expect(box!.y).toBeGreaterThanOrEqual(0);
      expect(box!.x + box!.width).toBeLessThanOrEqual(viewport.width);
      expect(box!.y + box!.height).toBeLessThanOrEqual(viewport.height);
      await budget.screenshot(`actions-confirm-${viewport.name}`);
      const editorName = `actions-confirm-${viewport.name}-dialog-deterministic-provider-mock.png`;
      const editorFile = process.env.BYTEBASE_TOKEN_BUDGET_SCREENSHOTS
        ? path.join(process.env.BYTEBASE_TOKEN_BUDGET_SCREENSHOTS, editorName)
        : budget.testInfo.outputPath(editorName);
      await dialog.screenshot({ path: editorFile });
      await budget.testInfo.attach(editorName, {
        path: editorFile,
        contentType: "image/png",
      });
      await confirm.click();
      await expect.poll(() => provider.calls).toBe(2);
      await expect(dialog).toBeHidden();
      await expect(
        budget.window.getByRole("button", { name: "Stop", exact: true })
      ).toBeVisible();
      const resumed = await budget.state();
      expect(resumed?.runId).toBe(before?.runId);
      expect(resumed?.budget.limit).toBe(2000);
      expect(resumed?.preference.tokenLimitPerRun).toBe(1000);
      expect(resumed?.userRows).toBe(before?.userRows);
      expect(provider.readToolResultIds).toEqual(before?.readToolCallIds);
      await expect(budget.input).toHaveValue(
        "Budget QA draft preserved after recovery"
      );
      finish();
      await expect(
        budget.window.getByText("Deterministic budget fixture completed.", {
          exact: true,
        })
      ).toBeVisible();
      await expect(budget.region).toBeHidden();
      expect((await budget.state())?.budget.totalTokensUsed).toBe(1200);
      expect(provider.calls).toBe(2);
      await budget.screenshot(`actions-completed-${viewport.name}`);
      await captureFooter("completed");
      await footer.dispose();
    });
  });
}
