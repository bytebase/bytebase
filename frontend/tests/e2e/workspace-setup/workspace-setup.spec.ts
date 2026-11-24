// First-run setup wizard (docs/design/onboarding-flow.md, amended by
// docs/design/onboarding-self-hosted-steps.md).
//
// The wizard is offered only to a workspace's sole member, and its sample is
// single-use, so the shared e2e server (two members, sample used) cannot show
// it. This spec boots a server of its own in first-run state and walks one
// journey on it, in the only order that keeps the wizard reachable: the two
// ways out and the own-instance path use no sample; the sample path goes last.
//
// That server is self-hosted, so the journey sees self-hosted's three steps:
// no name on the first (sign-up asked it), Finish on the project step, and a
// sample that runs on this server. Cloud's name, team step and invitations are
// covered by the wizard's unit tests.

import { expect, type Page, test } from "@playwright/test";
import {
  type BareServer,
  bareServerStartTimeoutMs,
  startBareServer,
} from "../framework/bare-server";
import { loginBrowserContext } from "../framework/sign-in";

test.setTimeout(420_000);
test.use({
  // The shared suite's signed-in state belongs to another server.
  storageState: { cookies: [], origins: [] },
  // Without these an action on a missing element waits out the whole test.
  actionTimeout: 15_000,
  navigationTimeout: 30_000,
});

let server: BareServer;

test.beforeAll(async () => {
  // The file-level timeout does not cover hooks.
  test.setTimeout(bareServerStartTimeoutMs());
  server = await startBareServer();
});

test.afterAll(async () => {
  await server?.stop();
});

const continueButton = (page: Page) =>
  page.getByRole("button", { name: "Continue", exact: true });
const finishButton = (page: Page) =>
  page.getByRole("button", { name: "Finish", exact: true });
const backButton = (page: Page) =>
  page.getByRole("button", { name: "Back", exact: true });

const openWizard = async (page: Page) => {
  await page.goto(`${server.baseURL}/auth/setup`);
  await expect(
    page.getByRole("heading", { name: "Name your workspace" })
  ).toBeVisible();
};

const choice = (page: Page, title: string) =>
  page.locator("label", { hasText: title }).getByRole("radio");

const projectNames = async () =>
  ((await server.api.listProjects()).projects ?? []).map(({ name }) => name);

// Lets the next matching request reach the server, then answers the page with
// a server error: the work is done and its answer is lost. Resolves once the
// page has that answer.
const loseAnswerOnce = async (page: Page, method: string, act: () => Promise<void>) => {
  await page.route(
    `**/bytebase.v1.${method}`,
    async (route) => {
      await route.fetch();
      await route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({ code: "internal", message: "injected" }),
      });
    },
    { times: 1 }
  );
  const lost = page.waitForResponse(
    (response) => response.url().endsWith(method) && response.status() === 500,
    { timeout: 240_000 }
  );
  await act();
  await lost;
};

test("first-run setup: skip, no project, own instance, then the sample", async ({
  page,
  context,
}) => {
  await loginBrowserContext(
    context,
    server.baseURL,
    server.adminEmail,
    server.adminPassword
  );

  await test.step("Skip setup saves the workspace name and creates no project", async () => {
    const projectsBefore = await projectNames();
    await openWizard(page);
    await expect(page.getByLabel(/^Your name/)).toHaveCount(0);
    await page.getByLabel("Workspace name").fill("Acme");
    await page.getByRole("button", { name: "Skip setup" }).click();

    await page.waitForURL(/\/landing/);
    expect(await projectNames()).toEqual(projectsBefore);
  });

  await test.step("Finish without a project saves the answers and creates no project", async () => {
    const projectsBefore = await projectNames();
    await openWizard(page);
    // Setup ended before, so this name is the one it saved.
    await expect(page.getByLabel("Workspace name")).toHaveValue("Acme");
    await continueButton(page).click();

    await expect(continueButton(page)).toBeDisabled();
    await page.getByText("Not sure yet, just exploring").click();
    await continueButton(page).click();

    // The database choice shows before the project is named.
    await expect(page.getByText(/running on this server/)).toBeVisible();
    await expect(page.getByText(/7 days/)).toHaveCount(0);
    await expect(finishButton(page)).toBeDisabled();
    await page.getByRole("button", { name: "Finish without a project" }).click();

    await page.waitForURL(/\/landing/);
    expect(await projectNames()).toEqual(projectsBefore);
  });

  await test.step("the own-instance path keeps every answer on Back and finishes on the project step", async () => {
    await openWizard(page);
    // A name only the draft holds, so Back cannot show it from the server.
    await page.getByLabel("Workspace name").fill("Acme Data");
    await continueButton(page).click();
    await page.getByText("Plan, review and roll out a database change").click();
    await continueButton(page).click();

    await page.getByLabel("Project name").fill("Payments");
    const chosenId = await page
      .getByText(/^payments-[a-z0-9]{4}$/)
      .innerText();
    await page.getByText("Connect my own instance after setup").click();

    await backButton(page).click();
    await expect(
      choice(page, "Plan, review and roll out a database change")
    ).toHaveAttribute("aria-checked", "true");
    await backButton(page).click();
    await expect(page.getByLabel("Workspace name")).toHaveValue("Acme Data");
    await continueButton(page).click();
    await continueButton(page).click();
    await expect(page.getByLabel("Project name")).toHaveValue("Payments");
    await expect(page.getByText(chosenId)).toBeVisible();
    await expect(
      choice(page, "Connect my own instance after setup")
    ).toHaveAttribute("aria-checked", "true");

    await finishButton(page).click();

    // An own-instance setup finishes on the project's instances page, where
    // the connect control carries the "Connect the instance" intro (#21594).
    // The Getting started bar has a "Connect instance" step button too.
    await page.waitForURL(new RegExp(`/projects/${chosenId}/instances`));
    await expect(
      page
        .locator("#bb-layout-main")
        .getByRole("button", { name: "Connect instance" })
    ).toBeVisible();
    await expect(page.getByText("Connect the instance")).toBeVisible();
  });

  await test.step("the sample path recovers from a create whose answer was lost and leads with the goal", async () => {
    const projectsBefore = await projectNames();
    await openWizard(page);
    await continueButton(page).click();
    // The step before stored the change goal, so a query-first landing below
    // can only come from this answer.
    await page.getByText("Query data with guardrails").click();
    await continueButton(page).click();

    await page.getByLabel("Project name").fill("Orders");
    const chosenId = await page.getByText(/^orders-[a-z0-9]{4}$/).innerText();
    await expect(
      choice(page, "Start with the sample database")
    ).toHaveAttribute("aria-checked", "true");

    // The project is created and the answer is lost; the failure shows on the
    // project step, which holds Finish on self-hosted.
    await loseAnswerOnce(page, "ProjectService/CreateProject", () =>
      finishButton(page).click()
    );
    await expect(page.getByText("Could not create Orders.")).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Finish without a project" })
    ).toHaveCount(0);

    // The retry takes that project as its own and prepares the sample.
    await page.getByRole("button", { name: "Try again" }).click();

    // The sample path finishes on the project's databases page; once the
    // sample instance has synced, the intro leads with the query goal chosen
    // above (#21594).
    await page.waitForURL(new RegExp(`/projects/${chosenId}/databases`));
    await expect(
      page.getByText("Database ready to query").first()
    ).toBeVisible({ timeout: 90_000 });
    expect((await projectNames()).sort()).toEqual(
      [...projectsBefore, `projects/${chosenId}`].sort()
    );
  });
});
