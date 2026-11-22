import { create } from "@bufbuild/protobuf";
import { beforeEach, expect, test, vi } from "vitest";
import type { AgentDomContext } from "@/modules/agent/dom/interactionRegistry";
import { State } from "@/types/proto-es/v1/common_pb";
import { DatabaseSchema$ as DatabaseSchema } from "@/types/proto-es/v1/database_service_pb";
import { readAuthorizedDatabase } from "./databaseSettingsPermission";

const mocks = vi.hoisted(() => ({
  db: vi.fn(),
  user: vi.fn(),
  roles: vi.fn(),
  workspace: vi.fn(),
  project: vi.fn(),
  getState: vi.fn(),
  setState: vi.fn(),
  check: vi.fn(),
}));
vi.mock("@/api", () => ({
  databaseServiceClientConnect: { getDatabase: mocks.db },
  userServiceClientConnect: { getCurrentUser: mocks.user },
  roleServiceClientConnect: { listRoles: mocks.roles },
  workspaceServiceClientConnect: { getIamPolicy: mocks.workspace },
  projectServiceClientConnect: { getIamPolicy: mocks.project },
}));
vi.mock("@/stores/app", () => ({
  useAppStore: { getState: mocks.getState, setState: mocks.setState },
}));
vi.mock("@/modules/agent/logic/executionContext", () => ({
  isAgentExecutionContextCurrent: mocks.check,
}));
const context: AgentDomContext = {
  actor: "users/dev@example.com",
  workspace: "workspaces/demo",
  projectName: "projects/alpha",
  pageKey: "/database",
  pageGeneration: 1,
  chatId: "chat",
  runId: "run",
  runMode: "default",
  isCurrent: () => true,
};
const database = create(DatabaseSchema, {
  name: "instances/prod/databases/app",
  instanceResource: { name: "instances/prod" },
  project: "projects/alpha",
  state: State.ACTIVE,
});
test("does not use a partial database or a database carrying another instance's environment", async () => {
  for (const instanceResource of [undefined, { name: "instances/other" }]) {
    mocks.db.mockResolvedValue({ ...database, instanceResource });
    expect(await read()).toBeUndefined();
  }
});
const binding = { role: "roles/updater", members: ["user:dev@example.com"] };
const user = {
  name: context.actor,
  email: "dev@example.com",
  workspace: context.workspace,
  state: State.ACTIVE,
  groups: ["groups/team@example.com"],
};
const read = () => readAuthorizedDatabase(database, context);
beforeEach(() => {
  vi.resetAllMocks();
  mocks.check.mockReturnValue(true);
  mocks.db.mockResolvedValue(database);
  mocks.getState.mockReturnValue({ serverInfo: { saas: false } });
  mocks.user.mockResolvedValue(user);
  mocks.roles.mockResolvedValue({
    roles: [{ name: binding.role, permissions: ["bb.databases.update"] }],
  });
  mocks.workspace.mockResolvedValue({ bindings: [] });
  mocks.project.mockResolvedValue({ bindings: [binding] });
});
test("requires four independent observations for the database's actual project without publishing", async () => {
  expect(await read()).toBe(database);
  expect(mocks.project).toHaveBeenCalledWith(
    expect.objectContaining({ resource: database.project }),
    expect.objectContaining({ timeoutMs: expect.any(Number) })
  );
  expect(mocks.workspace).toHaveBeenCalledWith(
    expect.objectContaining({ resource: context.workspace }),
    expect.anything()
  );
  expect(mocks.user).toHaveBeenCalledTimes(1);
  expect(mocks.roles).toHaveBeenCalledTimes(1);
  expect(mocks.setState).not.toHaveBeenCalled();
});
test.each([
  { members: ["group:team@example.com"] },
  { members: ["users/dev@example.com"] },
  { members: ["dev@example.com"] },
  { members: ["user:allUsers"] },
  { condition: { expression: "true" } },
  {
    condition: {
      expression: "request.time < timestamp('2099-01-01T00:00:00Z')",
    },
  },
  { condition: {} },
])(
  "requires an independent unconditional canonical grant (%j)",
  async (part) => {
    mocks.project.mockResolvedValue({ bindings: [{ ...binding, ...part }] });
    expect(await read()).toBeUndefined();
    mocks.project.mockResolvedValue({
      bindings: [{ ...binding, ...part }, binding],
    });
    expect(await read()).toBe(database);
  }
);
test("workspace allUsers is only evidence for a known self-hosted workspace", async () => {
  mocks.project.mockResolvedValue({ bindings: [] });
  mocks.workspace.mockResolvedValue({
    bindings: [{ ...binding, members: ["allUsers"] }],
  });
  expect(await read()).toBe(database);
  for (const serverInfo of [undefined, { saas: true }]) {
    mocks.getState.mockReturnValue({ serverInfo });
    expect(await read()).toBeUndefined();
  }
  mocks.project.mockResolvedValue({
    bindings: [{ ...binding, members: ["allUsers"] }],
  });
  expect(await read()).toBe(database);
});
test("rejects incomplete/inconsistent evidence or a context changing during reads", async () => {
  for (const patch of [
    { name: "users/other@example.com" },
    { workspace: "workspaces/other" },
    { state: State.DELETED },
    { groups: undefined },
  ]) {
    mocks.user.mockResolvedValue({ ...user, ...patch });
    expect(await read()).toBeUndefined();
  }
  mocks.user.mockResolvedValue(user);
  mocks.roles.mockResolvedValueOnce({
    roles: [
      { name: binding.role, permissions: [] },
      { name: binding.role, permissions: ["bb.databases.update"] },
    ],
  });
  expect(await read()).toBeUndefined();
  mocks.workspace.mockRejectedValueOnce(new Error("forbidden"));
  expect(await read()).toBeUndefined();
  mocks.project.mockImplementationOnce(() => {
    mocks.check.mockReturnValue(false);
    return { bindings: [binding] };
  });
  expect(await read()).toBeUndefined();
});
test("does not query policies for a moved or inactive database, or a mismatched project-instance name", async () => {
  for (const patch of [
    { project: "projects/beta" },
    { state: State.DELETED },
    { name: "projects/beta/instances/prod/databases/app" },
  ]) {
    mocks.db.mockResolvedValue({ ...database, ...patch });
    expect(await read()).toBeUndefined();
  }
  const wrong = {
    ...database,
    name: "projects/beta/instances/prod/databases/app",
  };
  expect(await readAuthorizedDatabase(wrong, context)).toBeUndefined();
  expect(mocks.project).not.toHaveBeenCalled();
});
