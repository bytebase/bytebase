import { answer, expect, readPage, test } from "./token-budget.fixture";
import * as path from "node:path";

for (const viewport of [
  { name: "desktop", width: 1440, height: 1000 },
  { name: "mobile", width: 390, height: 844 },
]) {
  test.describe(`budget composer spacing ${viewport.name}`, () => {
    test.use({ viewport: { width: viewport.width, height: viewport.height } });

    for (const state of ["warning", "paused"] as const) {
      test(`${state} band has 12px before composer and no empty-band spacing`, async ({
        budget,
        provider,
      }) => {
        await expect(budget.region).toBeHidden();
        provider.respond(readPage(state === "warning" ? 800 : 1100));
        const release =
          state === "warning" ? provider.hold(answer(100)) : undefined;
        if (state === "paused") provider.respond(answer(100));
        await budget.send(`Budget QA spacing ${state}: inspect this page.`);
        if (state === "paused") await budget.paused();
        else
          await expect(budget.region).toContainText(
            "Approaching this run's token limit."
          );

        const container = await budget.region.evaluateHandle(
          (band) => band.parentElement!
        );
        const inputRow = budget.window.locator("[data-agent-input-row]");
        await budget.screenshot(`spacing-${state}-${viewport.name}`);
        await expect
          .poll(async () => {
            const band = await budget.region.boundingBox();
            const row = await inputRow.boundingBox();
            return band && row ? row.y - band.y - band.height : null;
          })
          .toBe(12);
        const footerName = `spacing-${state}-${viewport.name}-footer-deterministic-provider-mock.png`;
        const footerPath = process.env.BYTEBASE_TOKEN_BUDGET_SCREENSHOTS
          ? path.join(process.env.BYTEBASE_TOKEN_BUDGET_SCREENSHOTS, footerName)
          : budget.testInfo.outputPath(footerName);
        await container.asElement()!.screenshot({
          path: footerPath,
          mask: [budget.window.locator("[data-agent-context]")],
          maskColor: "#e5e7eb",
        });
        await budget.testInfo.attach(footerName, {
          path: footerPath,
          contentType: "image/png",
        });

        if (state === "warning") {
          release!();
          await expect(
            budget.window.getByText("Deterministic budget fixture completed.", {
              exact: true,
            })
          ).toBeVisible();
        } else {
          await budget.increase("2", { resume: true });
          await expect(
            budget.window.getByText("Deterministic budget fixture completed.", {
              exact: true,
            })
          ).toBeVisible();
        }
        await expect(budget.region).toBeHidden();
        await expect
          .poll(() =>
            container.evaluate((element) => {
              const row = element.querySelector("[data-agent-input-row]")!;
              const style = getComputedStyle(element);
              const contentTop =
                element.getBoundingClientRect().top +
                parseFloat(style.borderTopWidth) +
                parseFloat(style.paddingTop);
              return row.getBoundingClientRect().top - contentTop;
            })
          )
          .toBe(0);
        await budget.screenshot(`spacing-no-band-${state}-${viewport.name}`);
        expect(provider.calls).toBe(2);
        await container.dispose();
      });
    }
  });
}
