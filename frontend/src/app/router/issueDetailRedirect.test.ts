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
});

describe("issueDetailRedirectLoader", () => {
  test("redirects a schema/data change issue to Plan Detail", async () => {
    mocks.getIssue.mockResolvedValue(
      makeIssue(Issue_Type.DATABASE_CHANGE, PLAN_NAME)
    );
    mocks.getPlan.mockResolvedValue(makePlan(["changeDatabaseConfig"]));

    const res = await run({ projectId: "p1", issueId: "123" });

    expect(res).toBeInstanceOf(Response);
    expect(res?.status).toBe(302);
    expect(res?.headers.get("Location")).toBe("/projects/p1/plans/456");
    expect(mocks.getPlan).toHaveBeenCalledOnce();
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

  test("preserves the query string on redirect", async () => {
    mocks.getIssue.mockResolvedValue(
      makeIssue(Issue_Type.DATABASE_CHANGE, PLAN_NAME)
    );
    mocks.getPlan.mockResolvedValue(makePlan(["changeDatabaseConfig"]));

    const res = await run(
      { projectId: "p1", issueId: "123" },
      "http://localhost/projects/p1/issues/123?stageId=7&foo=bar"
    );

    expect(res?.headers.get("Location")).toBe(
      "/projects/p1/plans/456?stageId=7&foo=bar"
    );
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
