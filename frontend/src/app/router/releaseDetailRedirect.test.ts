// @vitest-environment node
import type { LoaderFunctionArgs } from "react-router";
import { beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  listPlans: vi.fn(),
}));

vi.mock("@/api", () => ({
  planServiceClientConnect: { listPlans: mocks.listPlans },
}));

import { setAppRouter } from "./navigation";
import { releaseDetailRedirectLoader } from "./releaseDetailRedirect";

const RELEASES_URL =
  "/projects/p1/releases?q=release%3Aprojects%252Fp1%252Freleases%252Fr1";

const run = (): Promise<Response | null> =>
  releaseDetailRedirectLoader({
    params: { projectId: "p1", releaseId: "r1" },
    request: new Request("http://localhost/projects/p1/releases/r1"),
  } as unknown as LoaderFunctionArgs);

beforeEach(() => {
  mocks.listPlans.mockReset();
  setAppRouter({
    navigate: vi.fn(),
    state: {
      initialized: false,
      location: { pathname: "/projects/p1/releases/r1", search: "", hash: "" },
      matches: [],
    },
  });
});

describe("releaseDetailRedirectLoader", () => {
  test("opens the rollout of the one active Plan that used the release", async () => {
    mocks.listPlans.mockResolvedValue({
      plans: [{ name: "projects/p1/plans/7", hasRollout: true }],
    });

    const response = await run();

    expect(response?.headers.get("Location")).toBe(
      "/projects/p1/plans/7/rollout"
    );
    expect(mocks.listPlans).toHaveBeenCalledWith(
      expect.objectContaining({
        parent: "projects/p1",
        filter: 'release == "projects/p1/releases/r1" && state == "ACTIVE"',
        pageSize: 2,
      })
    );
  });

  test("opens Plan Detail when that Plan has no rollout yet", async () => {
    mocks.listPlans.mockResolvedValue({
      plans: [{ name: "projects/p1/plans/7", hasRollout: false }],
    });

    const response = await run();

    expect(response?.headers.get("Location")).toBe("/projects/p1/plans/7");
  });

  test.each([
    { name: "no Plan", plans: [] },
    {
      name: "several Plans",
      plans: [{ name: "projects/p1/plans/7" }, { name: "projects/p1/plans/8" }],
    },
  ])("lists the release's Plans when it has $name", async ({ plans }) => {
    mocks.listPlans.mockResolvedValue({ plans });

    const response = await run();

    expect(response?.headers.get("Location")).toBe(RELEASES_URL);
  });

  test("lists the release's Plans when the lookup fails", async () => {
    mocks.listPlans.mockRejectedValue(new Error("unavailable"));

    const response = await run();

    expect(response?.headers.get("Location")).toBe(RELEASES_URL);
  });
});
