import { answer, expect, readPage, test } from "./token-budget.fixture";
import { TokenBudgetPage } from "./token-budget.page";
import zhCN from "../../../src/locales/zh-CN.json";

test("fresh browser preferences retain 100K default, send shortcut and opening controls", async ({
  budget,
  provider,
}) => {
  expect(budget.initialDefault).toBe("100");
  await budget.setDefaultLimit("100");
  await budget.window
    .getByRole("button", { name: "Agent preferences", exact: true })
    .click();
  await expect(
    budget.window.getByRole("switch", { name: "Send with Enter", exact: true })
  ).toBeChecked();
  await expect(
    budget.window.getByRole("radio", { name: "Show landing page", exact: true })
  ).toBeChecked();
  await expect(
    budget.window.getByRole("spinbutton", {
      name: "Token limit per run",
      exact: true,
    })
  ).toHaveValue("100");
  await budget.screenshot("00-preferences-100k-desktop-en-light");
  expect(provider.calls).toBe(0);
});

test("pending and below-80-percent usage never claims a missing report or approaches the limit", async ({
  budget,
  provider,
}) => {
  const first = provider.hold(readPage(790));
  const final = provider.hold(answer(0));
  await budget.send("Budget QA TB-01: inspect the current page and finish.");
  await expect.poll(() => provider.calls).toBe(1);
  await expect(budget.region).toBeHidden();
  await budget.screenshot("17-in-flight-desktop-en-light");
  await budget.window
    .getByRole("button", { name: "Agent preferences", exact: true })
    .click();
  await budget.window
    .getByRole("button", { name: "Back to chat", exact: true })
    .click();
  await expect(budget.region).toBeHidden();
  await budget.window
    .getByRole("button", { name: "Minimize", exact: true })
    .click();
  await budget.page.locator("button[data-agent-window]").click();
  await expect(budget.region).toBeHidden();
  expect(provider.calls).toBe(1);
  first();
  await expect.poll(() => provider.calls).toBe(2);
  await expect(budget.region).toBeHidden();
  await budget.openUsage();
  await expect(budget.usage.locator("[data-agent-run-usage]")).toContainText(
    "790"
  );
  await budget.screenshot("01-normal-usage-desktop-en-light");
  await budget.page.keyboard.press("Escape");
  final();
  await expect(
    budget.window.getByText("Deterministic budget fixture completed.", {
      exact: true,
    })
  ).toBeVisible();
  expect((await budget.state())?.budget.totalTokensUsed).toBe(790);
});

test("pause, reload, validate and increase-and-resume retain one logical run and draft", async ({
  budget,
  provider,
}) => {
  provider.respond(readPage(1100, 2));
  provider.respond(answer(100));
  await budget.send(
    "Budget QA TB-03: inspect the current page twice, then finish."
  );
  await budget.paused();
  await budget.screenshot("03-paused-desktop-en-light");
  expect(provider.calls).toBe(1);
  const before = await budget.state();
  expect(before?.budget.totalTokensUsed).toBe(1100);
  expect(before?.readToolCallIds).toHaveLength(2);
  expect(before?.readToolResultIds).toEqual(before?.readToolCallIds);
  expect(new Set(before?.readToolCallIds).size).toBe(2);
  expect(before?.attempts).toHaveLength(1);
  await expect(
    budget.region.getByRole("button", { name: "Resume", exact: true })
  ).toBeHidden();
  await budget.input.fill("Budget QA unsent draft");
  await budget.input.press("Enter");
  await expect(budget.input).toHaveValue("Budget QA unsent draft");
  await budget.page.reload();
  await budget.reopen();
  await budget.paused();
  await expect(budget.input).toHaveValue("Budget QA unsent draft");
  expect(provider.calls).toBe(1);
  const restored = await budget.state();
  expect(restored?.runId).toBe(before?.runId);
  expect(restored?.budget.limit).toBe(1000);
  await budget.openUsage();
  await expect(budget.usage).toContainText("1,100");
  await budget.screenshot("25-rehydrated-usage-desktop-en-light");
  await budget.page.keyboard.press("Escape");
  await budget.region
    .getByRole("button", { name: "Increase limit", exact: true })
    .click();
  const limit = budget.page.getByRole("spinbutton", {
    name: "This run's token limit",
    exact: true,
  });
  const editor = budget.page.getByRole("dialog", {
    name: "Increase limit for this run",
    exact: true,
  });
  await expect(editor.getByText("K tokens", { exact: true })).toBeVisible();
  await expect(limit).toHaveAttribute("step", "0.001");
  await expect(limit).toHaveValue("2");
  await expect(limit).toHaveAttribute("aria-invalid", "false");
  await expect(editor.getByRole("alert")).toBeHidden();
  await expect(
    editor.getByRole("button", { name: "Increase and resume", exact: true })
  ).toBeEnabled();
  expect(provider.calls).toBe(1);
  expect((await budget.state())?.budget.limit).toBe(1000);
  for (const value of [
    "",
    "0",
    "-1",
    "1.1005",
    "1",
    "1.1",
    "9007199254740992",
  ]) {
    await limit.fill(value);
    await expect(limit).toHaveAttribute("aria-invalid", "true");
    await expect(
      budget.page.getByRole("button", {
        name: "Increase and resume",
        exact: true,
      })
    ).toBeDisabled();
  }
  await budget.screenshot("04-invalid-overflow-desktop-en-light");
  await limit.fill("2");
  await budget.page
    .getByRole("button", { name: "Cancel", exact: true })
    .click();
  expect((await budget.state())?.budget.limit).toBe(1000);
  await budget.increase("2", { resume: true });
  await expect(
    budget.window.getByText("Deterministic budget fixture completed.", {
      exact: true,
    })
  ).toBeVisible();
  expect(provider.calls).toBe(2);
  expect(provider.readToolResultIds).toEqual(before?.readToolCallIds);
  const completed = await budget.state();
  expect(completed?.budget.limit).toBe(2000);
  expect(completed?.preference.tokenLimitPerRun).toBe(1000);
  expect(completed?.budget.runId).toBe(before?.budget.runId);
  expect(completed?.userRows).toBe(before?.userRows);
  expect(completed?.readToolCallIds).toEqual(before?.readToolCallIds);
  expect(completed?.readToolResultIds).toEqual(before?.readToolResultIds);
  expect(completed?.attempts).toHaveLength(2);
  expect(
    new Set(
      completed?.attempts.map(
        (attempt: { attemptId: string }) => attempt.attemptId
      )
    ).size
  ).toBe(2);
  await expect(budget.input).toHaveValue("Budget QA unsent draft");
  await budget.openUsage();
  await expect(budget.usage).toContainText("1,200");
  await budget.screenshot("08-completed-usage-desktop-en-light");
});

test("K-token run limit prefills a whole K and preserves single-token precision above non-round usage", async ({
  budget,
  provider,
}) => {
  provider.respond(readPage(7202));
  provider.respond(answer(0));
  await budget.send("Budget QA TB-39: inspect this page, then finish.");
  await budget.paused();
  const before = await budget.state();
  await budget.region
    .getByRole("button", { name: "Increase limit", exact: true })
    .click();
  const editor = budget.page.getByRole("dialog", {
    name: "Increase limit for this run",
    exact: true,
  });
  const limit = editor.getByRole("spinbutton", {
    name: "This run's token limit",
    exact: true,
  });
  const confirm = editor.getByRole("button", {
    name: "Increase and resume",
    exact: true,
  });
  await expect(editor.getByText("K tokens", { exact: true })).toBeVisible();
  await expect(limit).toHaveAttribute("step", "0.001");
  await expect(limit).toHaveAttribute("min", "7.203");
  await expect(limit).toHaveValue("8");
  await expect(limit).toHaveAttribute("aria-invalid", "false");
  await expect(editor.getByRole("alert")).toBeHidden();
  await expect(confirm).toBeEnabled();
  expect(provider.calls).toBe(1);
  expect((await budget.state())?.budget.limit).toBe(1000);
  await limit.fill("7.202");
  await expect(limit).toHaveAttribute("aria-invalid", "true");
  await expect(editor).toContainText("7.202 K tokens");
  await expect(confirm).toBeDisabled();
  await budget.screenshot("39-k-limit-invalid-minimum-desktop-en-light");
  await limit.fill("7.203");
  await expect(confirm).toBeEnabled();
  await confirm.click();
  await expect(
    budget.window.getByText("Deterministic budget fixture completed.", {
      exact: true,
    })
  ).toBeVisible();
  const completed = await budget.state();
  expect(completed?.budget.limit).toBe(7203);
  expect(completed?.budget.totalTokensUsed).toBe(7202);
  expect(completed?.budget.runId).toBe(before?.budget.runId);
  expect(completed?.preference.tokenLimitPerRun).toBe(1000);
  expect(provider.calls).toBe(2);
});

test("80 percent warning is nonmodal and an active increase neither sends nor resets usage", async ({
  budget,
  provider,
}) => {
  provider.respond(readPage(800));
  const release = provider.hold(answer(100));
  await budget.send("Budget QA TB-02: inspect the current page, then finish.");
  await expect(budget.region).toContainText(
    "Approaching this run's token limit."
  );
  await expect.poll(() => provider.calls).toBe(2);
  await expect(
    budget.region.getByRole("button", { name: "Resume", exact: true })
  ).toBeHidden();
  await budget.screenshot("02-warning-desktop-en-light");
  await budget.increase("2");
  expect(provider.calls).toBe(2);
  release();
  await expect(
    budget.window.getByText("Deterministic budget fixture completed.", {
      exact: true,
    })
  ).toBeVisible();
  expect((await budget.state())?.budget.limit).toBe(2000);
  expect((await budget.state())?.budget.totalTokensUsed).toBe(900);
});

test("End run preserves draft and only explicit Send gets a fresh allowance", async ({
  budget,
  provider,
}) => {
  provider.respond(readPage(1100));
  provider.respond(answer(50, "Fresh deterministic run completed."));
  await budget.send("Budget QA TB-11: inspect this page.");
  await budget.paused();
  const before = await budget.state();
  await budget.input.fill("Budget QA fresh request");
  await budget.region
    .getByRole("button", { name: "End run", exact: true })
    .click();
  await expect(
    budget.region.getByRole("button", { name: "Resume", exact: true })
  ).toBeHidden();
  await expect(budget.input).toHaveValue("Budget QA fresh request");
  await budget.screenshot("11-ended-desktop-en-light");
  expect(provider.calls).toBe(1);
  await budget.window
    .getByRole("button", { name: "Send", exact: true })
    .click();
  await expect(
    budget.window.getByText("Fresh deterministic run completed.", {
      exact: true,
    })
  ).toBeVisible();
  const after = await budget.state();
  expect(after?.budget.runId).not.toBe(before?.budget.runId);
  expect(after?.budget.limit).toBe(1000);
  expect(after?.userRows).toBe(before!.userRows + 1);
  expect(provider.calls).toBe(2);
});

for (const total of [1000, 1100]) {
  test(`terminal answer at ${total} does not acquire a Resume lock`, async ({
    budget,
    provider,
  }) => {
    provider.respond(answer(total));
    await budget.send(`Budget QA TB-13: return a final answer at ${total}.`);
    await expect(
      budget.window.getByText("Deterministic budget fixture completed.", {
        exact: true,
      })
    ).toBeVisible();
    await expect(
      budget.region.getByRole("button", { name: "Resume", exact: true })
    ).toBeHidden();
    if (total === 1000)
      await expect(
        budget.window.getByText("Run completed above its token limit.", {
          exact: true,
        })
      ).toBeHidden();
    await budget.input.fill("Budget QA next request");
    await expect(
      budget.window.getByRole("button", { name: "Send", exact: true })
    ).toBeEnabled();
    await budget.screenshot(`13-completed-${total}-desktop-en-light`);
    expect(provider.calls).toBe(1);
    const completedRunId = (await budget.state())?.budget.runId;
    provider.respond(answer(25, "Next deterministic request completed."));
    await budget.window
      .getByRole("button", { name: "Send", exact: true })
      .click();
    await expect(
      budget.window.getByText("Next deterministic request completed.", {
        exact: true,
      })
    ).toBeVisible();
    expect((await budget.state())?.budget.runId).not.toBe(completedRunId);
    expect((await budget.state())?.budget.limit).toBe(1000);
    expect((await budget.state())?.budget.totalTokensUsed).toBe(25);
    expect(provider.calls).toBe(2);
  });
}

test("missing and zero usage remain distinct, with failed attempts separately counted", async ({
  budget,
  provider,
}) => {
  provider.respond(readPage(600));
  provider.respond(answer(undefined));
  await budget.send("Budget QA TB-15: inspect the current page and finish.");
  await expect(
    budget.window.getByText("Deterministic budget fixture completed.", {
      exact: true,
    })
  ).toBeVisible();
  await budget.openUsage();
  await expect(budget.usage).toContainText("At least 600");
  await expect(budget.usage).toContainText("Not reported");
  await budget.screenshot("15-partial-desktop-en-light");
  await budget.page.keyboard.press("Escape");
  provider.respond(answer(0, "Zero deterministic usage."));
  await budget.send("Budget QA TB-15: return explicit zero usage.");
  await expect(
    budget.window.getByText("Zero deterministic usage.", { exact: true })
  ).toBeVisible();
  await budget.openUsage();
  await expect(budget.usage.locator("[data-agent-run-usage]")).toContainText(
    "0"
  );
  await expect(budget.usage.locator("[data-agent-cached-input]")).toContainText(
    "Not reported"
  );
  await budget.screenshot("15-zero-desktop-en-light");
  await budget.page.keyboard.press("Escape");
  provider.respond(readPage(100));
  provider.fail();
  provider.fail();
  provider.fail();
  await budget.send(
    "Budget QA TB-16: inspect the current page before failure."
  );
  await expect(
    budget.window.locator("[data-agent-error-summary]")
  ).toBeVisible();
  await budget.openUsage();
  await expect(budget.usage).toContainText("Failed or interrupted attempts: 3");
  await expect(budget.usage).toContainText("At least 100");
  await budget.screenshot("16-failed-usage-desktop-en-light");
});

test("Stop and Retry retain run identity without presenting an ordinary token pause", async ({
  budget,
  provider,
}) => {
  const release = provider.hold(answer(100));
  await budget.send("Budget QA TB-23: wait for a controlled response.");
  await expect.poll(() => provider.calls).toBe(1);
  const before = await budget.state();
  await budget.window
    .getByRole("button", { name: "Stop", exact: true })
    .click();
  await budget.screenshot("23-stopped-desktop-en-light");
  await expect(
    budget.window.getByRole("button", { name: "Retry", exact: true })
  ).toBeVisible();
  await expect(budget.region).not.toContainText(
    "Run paused at its token limit."
  );
  release();
  provider.respond(answer(100, "Retried deterministic response."));
  await budget.input.fill("Budget QA draft after Stop");
  await budget.window
    .getByRole("button", { name: "Retry", exact: true })
    .click();
  await expect(
    budget.window.getByText("Retried deterministic response.", { exact: true })
  ).toBeVisible();
  expect((await budget.state())?.budget.runId).toBe(before?.budget.runId);
  await expect(budget.input).toHaveValue("Budget QA draft after Stop");
  expect(provider.calls).toBe(2);
  await budget.openUsage();
  await budget.screenshot("23-retry-usage-desktop-en-light");
});

test("shared tabs contend for one increase-and-resume and reloading a held call does not replay it", async ({
  budget,
  provider,
  context,
}) => {
  provider.respond(readPage(1100));
  const release = provider.hold(answer(100));
  await budget.send(
    "Budget QA TB-28: inspect the current page before pausing."
  );
  await budget.paused();
  const secondPage = await context.newPage();
  const second = new TokenBudgetPage(
    secondPage,
    budget.baseURL,
    budget.testInfo
  );
  await second.open();
  await second.window.getByRole("button", { name: /^Budget QA TB-28/ }).click();
  await second.paused();
  for (const tab of [budget, second]) {
    await tab.region
      .getByRole("button", { name: "Increase limit", exact: true })
      .click();
    await tab.page
      .getByRole("spinbutton", { name: "This run's token limit", exact: true })
      .fill("2");
  }
  await Promise.all([
    budget.page
      .getByRole("button", { name: "Increase and resume", exact: true })
      .click(),
    second.page
      .getByRole("button", { name: "Increase and resume", exact: true })
      .click(),
  ]);
  await expect.poll(() => provider.calls).toBe(2);
  const running = provider.ownerPage === budget.page ? budget : second;
  provider.phase = "owner-reload";
  await running.page.reload();
  await running.reopen();
  await running.window
    .getByRole("button", { name: /^Budget QA TB-28/ })
    .click();
  await expect(
    running.window.getByRole("button", { name: "Retry", exact: true })
  ).toBeVisible();
  expect(provider.calls).toBe(2);
  await running.screenshot("26-reload-interrupted-desktop-en-light");
  release();
  provider.phase = "retry-after-reload";
  provider.respond(answer(100));
  await running.window
    .getByRole("button", { name: "Retry", exact: true })
    .click();
  await expect(
    running.window.getByText("Deterministic budget fixture completed.", {
      exact: true,
    })
  ).toBeVisible();
  expect(provider.calls).toBe(3);
  provider.phase = "second-tab-close";
  await secondPage.close();
});

test.describe("mobile viewport", () => {
  test.use({ viewport: { width: 390, height: 844 } });
  test("mobile usage and increase popovers stay inside the Agent layer with OS dark preference", async ({
    budget,
    provider,
    page,
  }) => {
    await page.emulateMedia({ colorScheme: "dark" });
    provider.respond(readPage(1100));
    await budget.send("Budget QA TB-36: inspect the current page.");
    await budget.paused();
    await budget.openUsage();
    const box = await budget.usage.boundingBox();
    expect(box).not.toBeNull();
    expect(box!.x).toBeGreaterThanOrEqual(0);
    expect(box!.x + box!.width).toBeLessThanOrEqual(390);
    expect(box!.y).toBeGreaterThanOrEqual(0);
    expect(box!.y + box!.height).toBeLessThanOrEqual(844);
    await expect(
      budget.usage.locator("xpath=ancestor::*[@data-bb-layer-family='agent']")
    ).toHaveCount(1);
    await budget.screenshot("36-paused-usage-mobile-en-osdark-agent-light");
    await page.keyboard.press("Escape");
    await budget.region
      .getByRole("button", { name: "Increase limit", exact: true })
      .click();
    await expect(
      page.getByRole("spinbutton", {
        name: "This run's token limit",
        exact: true,
      })
    ).toBeVisible();
    const editor = page.getByRole("dialog", {
      name: "Increase limit for this run",
      exact: true,
    });
    await expect(
      editor.getByRole("heading", {
        name: "Increase limit for this run",
        exact: true,
      })
    ).toBeVisible();
    await expect(editor.getByText("K tokens", { exact: true })).toBeVisible();
    await expect
      .poll(async () => {
        const box = await editor.boundingBox();
        return (
          box !== null &&
          box.x >= 0 &&
          box.y >= 0 &&
          box.x + box.width <= 390 &&
          box.y + box.height <= 844
        );
      })
      .toBe(true);
    const editorBox = await editor.boundingBox();
    expect(editorBox).not.toBeNull();
    expect(editorBox!.x).toBeGreaterThanOrEqual(0);
    expect(editorBox!.x + editorBox!.width).toBeLessThanOrEqual(390);
    expect(editorBox!.y).toBeGreaterThanOrEqual(0);
    expect(editorBox!.y + editorBox!.height).toBeLessThanOrEqual(844);
    await budget.screenshot("36-increase-mobile-en-osdark-agent-light");
    expect(provider.calls).toBe(1);
  });
});

test("Chinese mobile budget labels, invalid editor and usage remain reachable after reload", async ({
  budget,
  provider,
  page,
}) => {
  provider.respond(readPage(1100));
  await budget.send("Budget QA TB-35: inspect the current page.");
  await budget.paused();
  await page.evaluate(() =>
    localStorage.setItem("bb.language", JSON.stringify("zh-CN"))
  );
  await page.setViewportSize({ width: 390, height: 844 });
  await page.reload();
  await page
    .getByRole("button", { name: zhCN.agent.self, exact: true })
    .click();
  await budget.window
    .getByRole("button", { name: zhCN.agent["chat-list-label"], exact: true })
    .click();
  await budget.window.getByRole("button", { name: /^Budget QA TB-35/ }).click();
  const region = page.getByRole("region", {
    name: zhCN.agent["token-budget"].title,
    exact: true,
  });
  await expect(region).toContainText(zhCN.agent["token-budget"].paused);
  await region
    .getByRole("button", {
      name: zhCN.agent["token-budget"].increase,
      exact: true,
    })
    .click();
  const input = page.getByRole("spinbutton", {
    name: zhCN.agent["token-budget"]["run-limit"],
    exact: true,
  });
  const editor = page.getByRole("dialog", {
    name: zhCN.agent["token-budget"]["increase-title"],
    exact: true,
  });
  await expect(
    editor.getByText(zhCN.agent.preferences["thousand-tokens"], { exact: true })
  ).toBeVisible();
  await input.fill("1.1");
  await expect(input).toHaveAttribute("aria-invalid", "true");
  await expect(
    page.getByRole("button", {
      name: zhCN.agent["token-budget"]["increase-and-resume"],
      exact: true,
    })
  ).toBeDisabled();
  await budget.screenshot("35-invalid-mobile-zh-light");
  await page.keyboard.press("Escape");
  await budget.window
    .getByRole("button", { name: /^\u804a\u5929 token/ })
    .click();
  await expect(budget.usage).toContainText(
    zhCN.agent["token-usage"]["run-usage"]
  );
  await budget.screenshot("35-paused-usage-mobile-zh-light");
  expect(provider.calls).toBe(1);
});

test("stored v3 run becomes read-only when Web Locks are unavailable", async ({
  budget,
  provider,
  page,
}) => {
  provider.respond(readPage(1100));
  await budget.send("Budget QA TB-32: inspect the current page.");
  await budget.paused();
  await page.addInitScript(() =>
    Object.defineProperty(navigator, "locks", {
      value: undefined,
      configurable: true,
    })
  );
  await page.reload();
  await budget.reopen();
  await budget.window.getByRole("button", { name: /^Budget QA TB-32/ }).click();
  await expect(
    budget.window.getByText(
      "This browser can't safely continue stored Agent runs. Use a supported, up-to-date browser.",
      { exact: true }
    )
  ).toBeVisible();
  for (const name of ["Increase limit", "End run"]) {
    await expect(
      budget.region.getByRole("button", { name, exact: true })
    ).toBeDisabled();
  }
  await expect(
    budget.region.getByRole("button", { name: "Resume", exact: true })
  ).toBeHidden();
  await budget.screenshot("32-unsupported-v3-desktop-en-light");
  expect(provider.calls).toBe(1);
});
