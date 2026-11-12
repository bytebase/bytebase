import { create } from "@bufbuild/protobuf";
import { act, cleanup, render } from "@testing-library/react";
import { SHA256 } from "crypto-js";
import { useState } from "react";
import { afterEach, expect, test, vi } from "vitest";
import type { AgentSqlFixTargetAdapter } from "@/modules/agent/logic/sqlFix";
import { Plan_SpecSchema, PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import type { PlanDetailPageState } from "../shell/hooks/types";
import { PlanDetailProvider } from "../shell/PlanDetailContext";
import { PlanSqlFixTargetProvider, useRegisterPlanSqlFixEditor } from "./usePlanSqlFixTarget";

const mocks = vi.hoisted(() => ({
  adapter: undefined as AgentSqlFixTargetAdapter | undefined,
  notifications: 0,
  push: vi.fn(),
  routeParams: { projectId: "example", planId: "42", specId: "spec-1" },
}));
vi.mock("@/app/router", () => ({ router: { push: mocks.push, get currentRoute() { return { value: { params: mocks.routeParams } }; } } }));
vi.mock("@/modules/agent/logic/sqlFix", () => ({
  getAgentSqlFixIdentity: () => '["workspaces/example","users/member@example.com"]',
  registerAgentSqlFixTarget: (adapter: AgentSqlFixTargetAdapter) => { mocks.adapter = adapter; return () => { mocks.adapter = undefined; }; },
  notifyAgentSqlFixTargetChanged: () => { mocks.notifications++; },
}));
vi.mock("@/stores/app", () => ({ useAppStore: Object.assign((select: (state: unknown) => unknown) => select({ dbGroupsByName: {} }), { getState: () => ({ dbGroupsByName: {}, fetchSheet: async (name: string) => create(SheetSchema, { name, content: new TextEncoder().encode("SELECT 1;"), contentSize: 9n }) }) }) }));

afterEach(() => { cleanup(); vi.clearAllMocks(); mocks.notifications = 0; });

test("does not finish revealing a different saved spec when the leave guard canceled navigation", async () => {
  const sql = "SELECT 1;";
  const sheet = `projects/example/sheets/${SHA256(sql).toString()}`;
  const spec = create(Plan_SpecSchema, { id: "spec-2", config: { case: "changeDatabaseConfig", value: { sheet, targets: ["instances/main/databases/books"] } } });
  const page = {
    pageKey: "example/42", projectId: "example", planId: "42", ready: true, isCreating: false, readonly: false,
    plan: create(PlanSchema, { name: "projects/example/plans/42", creator: "users/member@example.com", specs: [spec] }),
    currentUser: create(UserSchema, { name: "users/member@example.com" }),
    project: create(ProjectSchema, { name: "projects/example" }),
    expandPhase: vi.fn(),
  } as unknown as PlanDetailPageState;
  render(<PlanDetailProvider value={page}><PlanSqlFixTargetProvider selectedSpecId="spec-1" onSelectedSpecIdChange={() => {}}>Editor</PlanSqlFixTargetProvider></PlanDetailProvider>);
  const source = await mocks.adapter!.capture({ resource: { kind: "plan", name: page.plan.name, project: "projects/example", label: "Plan" }, sourceId: `${page.plan.name}/planCheckRun`, kind: "sql-check", summary: "Warning", sqlFixSource: { specId: "spec-2", sha256: SHA256(sql).toString(), target: "instances/main/databases/books" } });
  mocks.push.mockResolvedValueOnce(undefined);
  expect(await mocks.adapter!.reveal(source!)).toBe(false);
  expect(mocks.push).toHaveBeenCalledWith({ name: expect.any(String), params: { projectId: "example", planId: "42", specId: "spec-2" } }, { preventScrollReset: true });
  mocks.push.mockImplementationOnce(async () => { mocks.routeParams = { ...mocks.routeParams, specId: "spec-2" }; });
  expect(await mocks.adapter!.reveal(source!)).toBe(true);
});

test("does not reveal a route-selected source over another pending change's editor", async () => {
  const sql = "SELECT 1;";
  const sheet = `projects/example/sheets/${SHA256(sql).toString()}`;
  const targets = ["instances/main/databases/books"];
  const spec = create(Plan_SpecSchema, { id: "spec-1", config: { case: "changeDatabaseConfig", value: { sheet, targets } } });
  const page = {
    pageKey: "example/42", projectId: "example", planId: "42", ready: true, isCreating: false, readonly: false,
    plan: create(PlanSchema, { name: "projects/example/plans/42", creator: "users/member@example.com", specs: [spec] }),
    currentUser: create(UserSchema, { name: "users/member@example.com" }),
    project: create(ProjectSchema, { name: "projects/example" }),
    expandPhase: vi.fn(),
  } as unknown as PlanDetailPageState;
  const apply = vi.fn(() => true);
  function PendingEditor() {
    useRegisterPlanSqlFixEditor({ pageKey: page.pageKey, plan: page.plan.name, specId: "pending-spec", sheet: "projects/example/sheets/-102", targets, currentSql: "Unfinished SQL", status: "ready" }, apply);
    return <div>Unfinished SQL</div>;
  }
  const selectSpec = vi.fn();
  const view = render(<PlanDetailProvider value={page}><PlanSqlFixTargetProvider selectedSpecId="spec-1" onSelectedSpecIdChange={selectSpec}><PendingEditor /></PlanSqlFixTargetProvider></PlanDetailProvider>);
  const source = await mocks.adapter!.capture({ resource: { kind: "plan", name: page.plan.name, project: "projects/example", label: "Plan" }, sourceId: `${page.plan.name}/planCheckRun`, kind: "sql-check", summary: "Warning", sqlFixSource: { specId: "spec-1", sha256: SHA256(sql).toString(), target: targets[0] } });
  expect(await mocks.adapter!.reveal(source!)).toBe(false);
  expect(mocks.push).not.toHaveBeenCalled();
  expect(selectSpec).not.toHaveBeenCalled();
  expect(apply).not.toHaveBeenCalled();
  expect(view.container).toHaveTextContent("Unfinished SQL");
});

test("registers a page-owned adapter while editors mount, update, and unmount", async () => {
  const sql = "SELECT 1;";
  const sheet = `projects/example/sheets/${SHA256(sql).toString()}`;
  const targets = ["instances/main/databases/books"];
  const spec = create(Plan_SpecSchema, { id: "spec-1", config: { case: "changeDatabaseConfig", value: { sheet, targets } } });
  const page = {
    pageKey: "example/42", ready: true, isCreating: false, readonly: false,
    plan: create(PlanSchema, { name: "projects/example/plans/42", creator: "users/member@example.com", specs: [spec] }),
    currentUser: create(UserSchema, { name: "users/member@example.com" }),
    project: create(ProjectSchema, { name: "projects/example" }),
    expandPhase: vi.fn(),
  } as unknown as PlanDetailPageState;
  let setSql!: (sql: string) => void;
  function Editor() {
    const [value, setValue] = useState(sql);
    setSql = setValue;
    useRegisterPlanSqlFixEditor({ pageKey: page.pageKey, plan: page.plan.name, specId: spec.id, sheet, targets, currentSql: value, status: "ready" }, (expected, next) => { if (value !== expected) return false; setValue(next); return true; });
    return <div>{value}</div>;
  }
  const selectSpec = vi.fn();
  const view = (editor: boolean) => <PlanDetailProvider value={page}><PlanSqlFixTargetProvider selectedSpecId="spec-1" onSelectedSpecIdChange={selectSpec}>{editor && <Editor />}</PlanSqlFixTargetProvider></PlanDetailProvider>;
  const rendered = render(view(false));
  const source = await mocks.adapter!.capture({ resource: { kind: "plan", name: page.plan.name, project: "projects/example", label: "Plan" }, sourceId: `${page.plan.name}/planCheckRun`, kind: "sql-check", summary: "Warning", sqlFixSource: { sha256: SHA256(sql).toString(), target: targets[0] } });
  expect(mocks.adapter!.inspect(source!)).toEqual({ status: "loading" });
  await act(async () => { await mocks.adapter!.reveal(source!); });
  expect(page.expandPhase).toHaveBeenCalledWith("changes");
  expect(mocks.push).not.toHaveBeenCalled();
  rendered.rerender(view(true));
  expect(mocks.adapter!.inspect(source!)).toEqual({ status: "ready", currentSql: sql });
  act(() => setSql("SELECT 2;"));
  expect(mocks.adapter!.inspect(source!).status).toBe("stale");
  expect(mocks.adapter!.apply(source!, "SELECT 3;")).toBe(false);
  expect(rendered.container).toHaveTextContent("SELECT 2;");
  expect(mocks.notifications).toBeGreaterThan(1);
  rendered.rerender(view(false));
  expect(mocks.adapter!.inspect(source!).status).toBe("loading");
  rendered.unmount();
  expect(mocks.adapter).toBeUndefined();
});
