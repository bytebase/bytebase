import type { LoaderFunctionArgs } from "react-router";
import { beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  getIssue: vi.fn(),
  getPlan: vi.fn(),
}));

vi.mock("@/api", () => ({
  issueServiceClientConnect: { getIssue: mocks.getIssue },
  planServiceClientConnect: { getPlan: mocks.getPlan },
}));

import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import { issueDetailRedirectLoader } from "./issueDetailRedirect";
import { setAppRouter } from "./navigation";

const PLAN_NAME = "projects/p1/plans/456";

const makeIssue = (type: Issue_Type, plan = "", draft = false): Issue =>
  ({
    name: "projects/p1/issues/123",
    type,
    plan,
    draft,
  }) as unknown as Issue;

const makePlan = (cases: string[]): Plan =>
  ({
    specs: cases.map((caseName) => ({ config: { case: caseName, value: {} } })),
  }) as unknown as Plan;

const run = (
  params: { projectId?: string; issueId?: string },
  url = "http://localhost/projects/p1/issues/123"
): Promise<Response | null> =>
  issueDetailRedirectLoader({
    params,
    request: new Request(url),
  } as unknown as LoaderFunctionArgs);

beforeEach(() => {
  mocks.getIssue.mockReset();
  mocks.getPlan.mockReset();
  setAppRouter({
    navigate: vi.fn(),
    state: {
      initialized: false,
      location: {
        pathname: "/projects/p1/issues/123",
        search: "",
        hash: "",
      },
      matches: [],
    },
  });
});

describe("issueDetailRedirectLoader", () => {
  test("redirects a schema/data change issue to the Plan Detail root", async () => {
    mocks.getIssue.mockResolvedValue(
      makeIssue(Issue_Type.DATABASE_CHANGE, PLAN_NAME)
    );
    mocks.getPlan.mockResolvedValue(makePlan(["changeDatabaseConfig"]));

    const res = await run({ projectId: "p1", issueId: "123" });

    expect(res).toBeInstanceOf(Response);
    expect(res?.status).toBe(302);
    expect(res?.headers.get("Location")).toBe("/projects/p1/plans/456");
    expect(res?.headers.get("X-Remix-Replace")).toBe("true");
    expect(mocks.getPlan).toHaveBeenCalledOnce();
  });

  test("preserves the referring entry for an in-app issue navigation", async () => {
    setAppRouter({
      navigate: vi.fn(),
      state: {
        initialized: true,
        location: {
          pathname: "/projects/p1/issues",
          search: "",
          hash: "",
        },
        matches: [],
        navigation: { historyAction: "PUSH" },
      },
    });
    mocks.getIssue.mockResolvedValue(
      makeIssue(Issue_Type.DATABASE_CHANGE, PLAN_NAME)
    );
    mocks.getPlan.mockResolvedValue(makePlan(["changeDatabaseConfig"]));

    const res = await run({ projectId: "p1", issueId: "123" });

    expect(res?.headers.get("Location")).toBe("/projects/p1/plans/456");
    expect(res?.headers.get("X-Remix-Replace")).toBeNull();
  });

  test("redirects a Draft Review Issue URL to its Plan", async () => {
    mocks.getIssue.mockResolvedValue(
      makeIssue(Issue_Type.DATABASE_EXPORT, PLAN_NAME, true)
    );

    const res = await run({ projectId: "p1", issueId: "123" });

    expect(res?.status).toBe(302);
    expect(res?.headers.get("Location")).toBe("/projects/p1/plans/456");
    expect(mocks.getPlan).not.toHaveBeenCalled();
  });

  test.each(["changes", "review", "deploy"])(
    "preserves an explicit %s phase intent",
    async (phase) => {
      mocks.getIssue.mockResolvedValue(
        makeIssue(Issue_Type.DATABASE_CHANGE, PLAN_NAME)
      );
      mocks.getPlan.mockResolvedValue(makePlan(["changeDatabaseConfig"]));

      const res = await run(
        { projectId: "p1", issueId: "123" },
        `http://localhost/projects/p1/issues/123?phase=${phase}`
      );

      expect(res?.headers.get("Location")).toBe(
        `/projects/p1/plans/456?phase=${phase}`
      );
    }
  );

  test("drops an invalid phase intent", async () => {
    mocks.getIssue.mockResolvedValue(
      makeIssue(Issue_Type.DATABASE_CHANGE, PLAN_NAME)
    );
    mocks.getPlan.mockResolvedValue(makePlan(["changeDatabaseConfig"]));

    const res = await run(
      { projectId: "p1", issueId: "123" },
      "http://localhost/projects/p1/issues/123?phase=unknown"
    );

    expect(res?.headers.get("Location")).toBe("/projects/p1/plans/456");
  });

  test("preserves neutral query state but drops incompatible selectors", async () => {
    mocks.getIssue.mockResolvedValue(
      makeIssue(Issue_Type.DATABASE_CHANGE, PLAN_NAME)
    );
    mocks.getPlan.mockResolvedValue(makePlan(["changeDatabaseConfig"]));

    const res = await run(
      { projectId: "p1", issueId: "123" },
      "http://localhost/projects/p1/issues/123?stageId=7&foo=bar"
    );

    expect(res?.headers.get("Location")).toBe("/projects/p1/plans/456?foo=bar");
  });

  test("keeps a create-database issue on Issue Detail", async () => {
    mocks.getIssue.mockResolvedValue(
      makeIssue(Issue_Type.DATABASE_CHANGE, PLAN_NAME)
    );
    mocks.getPlan.mockResolvedValue(makePlan(["createDatabaseConfig"]));

    expect(await run({ projectId: "p1", issueId: "123" })).toBeNull();
  });

  test("keeps an export issue on Issue Detail without fetching the plan", async () => {
    mocks.getIssue.mockResolvedValue(
      makeIssue(Issue_Type.DATABASE_EXPORT, PLAN_NAME)
    );

    expect(await run({ projectId: "p1", issueId: "123" })).toBeNull();
    expect(mocks.getPlan).not.toHaveBeenCalled();
  });

  test("keeps a role-grant issue on Issue Detail", async () => {
    mocks.getIssue.mockResolvedValue(makeIssue(Issue_Type.ROLE_GRANT));

    expect(await run({ projectId: "p1", issueId: "123" })).toBeNull();
    expect(mocks.getPlan).not.toHaveBeenCalled();
  });

  test("keeps an access-grant issue on Issue Detail", async () => {
    mocks.getIssue.mockResolvedValue(makeIssue(Issue_Type.ACCESS_GRANT));

    expect(await run({ projectId: "p1", issueId: "123" })).toBeNull();
  });

  test("stays put when a change issue has no plan", async () => {
    mocks.getIssue.mockResolvedValue(makeIssue(Issue_Type.DATABASE_CHANGE, ""));

    expect(await run({ projectId: "p1", issueId: "123" })).toBeNull();
    expect(mocks.getPlan).not.toHaveBeenCalled();
  });

  test("does not fetch or redirect for the 'create' pseudo-issue", async () => {
    expect(await run({ projectId: "p1", issueId: "create" })).toBeNull();
    expect(mocks.getIssue).not.toHaveBeenCalled();
  });

  test("falls through to Issue Detail when the issue fetch fails", async () => {
    mocks.getIssue.mockRejectedValue(new Error("not found"));

    expect(await run({ projectId: "p1", issueId: "123" })).toBeNull();
  });

  test("falls through to Issue Detail when the plan fetch fails", async () => {
    mocks.getIssue.mockResolvedValue(
      makeIssue(Issue_Type.DATABASE_CHANGE, PLAN_NAME)
    );
    mocks.getPlan.mockRejectedValue(new Error("boom"));

    expect(await run({ projectId: "p1", issueId: "123" })).toBeNull();
  });
});
