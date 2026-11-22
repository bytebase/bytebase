import { create } from "@bufbuild/protobuf";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import {
  databaseServiceClientConnect,
  settingServiceClientConnect,
  roleServiceClientConnect,
  workspaceServiceClientConnect,
  projectServiceClientConnect,
  userServiceClientConnect,
} from "@/api";
import type { DomActionParams } from "@/modules/agent/dom/actions";
import {
  type AgentDomContext,
  captureAgentDomBinding,
  captureDomScope,
  isCapturedDomMutationCurrent,
  prepareCapturedInteraction,
  refreshCommittedDomMutation,
  validateCapturedDomMutation,
} from "@/modules/agent/dom/interactionRegistry";
import { publishDomSnapshot } from "@/modules/agent/dom/snapshotIndex";
import { useAppStore } from "@/stores/app";
import { STUB_ENVIRONMENTS } from "@/test-utils/environmentSelectStub";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  DatabaseSchema$ as DatabaseSchema,
  UpdateDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { IamPolicySchema } from "@/types/proto-es/v1/iam_policy_pb";
import { ListRolesResponseSchema } from "@/types/proto-es/v1/role_service_pb";
import { SettingSchema } from "@/types/proto-es/v1/setting_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import { DatabaseSettingsPanel } from "./DatabaseSettingsPanel";

vi.mock("react-i18next", async () => {
  const stub = (await import("@/test-utils/i18n")).reactI18nextStub();
  const translation = stub.useTranslation();
  return { ...stub, useTranslation: () => translation };
});
vi.mock("@/app/router", () => ({
  router: { currentRoute: { value: { fullPath: "/db/settings" } } },
}));
vi.mock("@/utils", async (original) => ({
  ...(await original<typeof import("@/utils")>()),
  hasProjectPermissionV2: () => true,
}));
vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: React.ReactNode }) => children,
}));
vi.mock("@/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({ environment }: { environment: { title: string } }) => (
    <span>{environment.title}</span>
  ),
}));
vi.mock("@/stores", async (original) => ({
  ...(await original<typeof import("@/stores")>()),
  pushNotification: vi.fn(),
}));
const db = create(DatabaseSchema, {
  name: "instances/prod/databases/app",
  project: "projects/alpha",
  state: State.ACTIVE,
  labels: { team: "ops" },
  environment: "environments/staging",
  effectiveEnvironment: "environments/staging",
  instanceResource: { name: "instances/prod" },
});
const context: AgentDomContext = {
  actor: "users/dev@example.com",
  workspace: "workspaces/demo",
  chatId: "chat",
  runId: "run",
  runMode: "default",
  pageKey: "/db/settings",
  projectName: db.project,
  pageGeneration: 1,
  isCurrent: () => true,
};
let server = db;
function Host() {
  const database = useAppStore((s) => s.databasesByName[db.name]);
  return (
    <>
      <input aria-label="title" />
      <DatabaseSettingsPanel database={database} />
    </>
  );
}
function index(element: Element) {
  const entry = {
    ref: "e1",
    element,
    generation: 1,
    tag: element.tagName,
    label: "",
    actions: "",
    binding: captureAgentDomBinding(element),
    scope: captureDomScope(context),
  };
  publishDomSnapshot([entry]);
  return entry;
}
async function prepare(
  element: Element,
  type: DomActionParams["type"],
  value?: string
) {
  const entry = index(element);
  return act(() =>
    prepareCapturedInteraction({ ref: entry.ref, type, value }, entry, context)
  );
}
const manual = { kind: "result", result: { success: false } };
const done = { kind: "result", result: { success: true } };
const trigger = () => document.querySelector(".max-w-md > div")!;
const button = (name: string) => screen.getByRole("button", { name });
const publish = (database: typeof db) =>
  act(() => useAppStore.setState({ databasesByName: { [db.name]: database } }));
beforeEach(() => {
  server = db;
  useAppStore.setState({
    databasesByName: { [db.name]: db },
    environmentList: STUB_ENVIRONMENTS,
    currentUser: create(UserSchema, {
      name: context.actor,
      workspace: context.workspace,
    }),
    currentUserName: context.actor,
    workspaceResourceName: () => context.workspace,
    batchFetchProjects: vi.fn().mockResolvedValue([]),
  });
  vi.spyOn(databaseServiceClientConnect, "getDatabase").mockImplementation(
    async () => server
  );
  vi.spyOn(databaseServiceClientConnect, "updateDatabase").mockImplementation(
    async (input) => {
      const req = create(UpdateDatabaseRequestSchema, input);
      server = create(DatabaseSchema, {
        ...server,
        ...(req.updateMask?.paths.includes("labels")
          ? { labels: req.database!.labels }
          : {
              environment: req.database?.environment,
              effectiveEnvironment: req.database?.environment,
            }),
      });
      return server;
    }
  );
  vi.spyOn(settingServiceClientConnect, "getSetting").mockResolvedValue(
    create(SettingSchema, {
      name: "settings/ENVIRONMENT",
      value: {
        value: {
          case: "environment",
          value: { environments: [{ id: "staging" }, { id: "prod" }] },
        },
      },
    })
  );
  vi.spyOn(userServiceClientConnect, "getCurrentUser").mockResolvedValue(
    create(UserSchema, {
      name: context.actor,
      email: "dev@example.com",
      state: State.ACTIVE,
      workspace: context.workspace,
    })
  );
  vi.spyOn(roleServiceClientConnect, "listRoles").mockResolvedValue(
    create(ListRolesResponseSchema, {
      roles: [{ name: "roles/updater", permissions: ["bb.databases.update"] }],
    })
  );
  vi.spyOn(workspaceServiceClientConnect, "getIamPolicy").mockResolvedValue(
    create(IamPolicySchema)
  );
  vi.spyOn(projectServiceClientConnect, "getIamPolicy").mockResolvedValue(
    create(IamPolicySchema, {
      bindings: [{ role: "roles/updater", members: ["user:dev@example.com"] }],
    })
  );
});
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});
test("label setters preserve a focused title, emit no events or writes and reject recycling a focused row", async () => {
  render(<Host />);
  const title = screen.getByLabelText("title");
  title.focus();
  const timer = vi.spyOn(window, "setTimeout");
  const event = vi.fn();
  for (const type of ["blur", "input", "click", "change"])
    document.addEventListener(type, event, true);
  try {
    expect(
      await prepare(screen.getByDisplayValue("ops"), "input", "svc")
    ).toMatchObject(done);
    expect(await prepare(button("label.add-label"), "click")).toMatchObject(
      done
    );
    const added = screen.getAllByRole("textbox").at(-1)!;
    expect(await prepare(added, "input", "on")).toMatchObject(done);
    expect(document.activeElement).toBe(title);
    expect(event).not.toHaveBeenCalled();
    expect(timer).not.toHaveBeenCalled();
    added.focus();
    expect(
      await prepare(
        screen.getAllByRole("button", { name: "common.delete" })[0],
        "click"
      )
    ).toMatchObject(manual);
    expect(document.activeElement).toBe(added);
    title.focus();
    expect(
      await prepare(
        screen.getAllByRole("button", { name: "common.delete" })[1],
        "click"
      )
    ).toMatchObject(done);
    expect(await prepare(button("common.revert"), "click")).toMatchObject(done);
    expect(screen.getByDisplayValue("ops")).toBeTruthy();
    expect(document.activeElement).toBe(title);
    expect(databaseServiceClientConnect.getDatabase).not.toHaveBeenCalled();
    expect(databaseServiceClientConnect.updateDatabase).not.toHaveBeenCalled();
  } finally {
    for (const type of ["blur", "input", "click", "change"])
      document.removeEventListener(type, event, true);
  }
});
test("labels survive equivalent and divergent reads; approval requires the fresh map and actual project", async () => {
  render(<Host />);
  fireEvent.change(screen.getByDisplayValue("ops"), {
    target: { value: "svc" },
  });
  const p = await prepare(button("common.save"), "click");
  expect(p.kind).toBe("confirm-api");
  if (p.kind !== "confirm-api") throw Error();
  expect(JSON.parse(p.call.body)).toEqual({
    database: { name: db.name, labels: { team: "svc" } },
    updateMask: "labels",
  });
  expect(p.origin.preview.changes).toEqual([
    { field: "team", before: "ops", after: "svc" },
  ]);
  publish({ ...db, labels: { ...db.labels } });
  expect(screen.getByDisplayValue("svc")).toBeTruthy();
  expect(await validateCapturedDomMutation(p.origin, context)).toBe(true);
  server = { ...db, project: "projects/other" };
  expect(await validateCapturedDomMutation(p.origin, context)).toBe(false);
  server = { ...db, labels: { team: "central" } };
  expect(await validateCapturedDomMutation(p.origin, context)).toBe(false);
  publish(server);
  expect(screen.getByDisplayValue("svc")).toBeTruthy();
  expect(await prepare(button("common.revert"), "click")).toMatchObject(done);
  expect(screen.getByDisplayValue("central")).toBeTruthy();
  expect(databaseServiceClientConnect.updateDatabase).not.toHaveBeenCalled();
});
test("environment prepares one frozen selection at a time without touching the picker, accepts native clear", async () => {
  render(<Host />);
  const el = trigger();
  const title = screen.getByLabelText("title");
  title.focus();
  const first = await prepare(el, "select", "environments/prod");
  const clear = await prepare(el, "select", "");
  expect(first.kind).toBe("confirm-api");
  expect(clear.kind).toBe("confirm-api");
  if (first.kind !== "confirm-api" || clear.kind !== "confirm-api")
    throw Error();
  expect(JSON.parse(clear.call.body)).toEqual({
    database: { name: db.name, environment: "" },
    updateMask: "environment",
  });
  expect(JSON.parse(first.call.body)).toEqual({
    database: { name: db.name, environment: "environments/prod" },
    updateMask: "environment",
  });
  expect(await prepare(el, "select", "Prod")).toMatchObject(manual);
  expect(await prepare(el, "click")).toMatchObject(manual);
  expect(screen.queryByRole("listbox")).toBeNull();
  expect(el.textContent).toContain("Staging");
  expect(document.activeElement).toBe(title);
  vi.mocked(settingServiceClientConnect.getSetting).mockResolvedValue(
    create(SettingSchema, {
      name: "settings/ENVIRONMENT",
      value: {
        value: {
          case: "environment",
          value: { environments: [{ id: "staging" }] },
        },
      },
    })
  );
  expect(await validateCapturedDomMutation(first.origin, context)).toBe(false);
  expect(await validateCapturedDomMutation(clear.origin, context)).toBe(true);
  expect(databaseServiceClientConnect.updateDatabase).not.toHaveBeenCalled();
});
test("inherited effective environment is a proven no-op, not an override, and cannot be cleared", async () => {
  server = {
    ...db,
    environment: undefined,
    instanceResource: {
      ...db.instanceResource!,
      environment: "environments/staging",
    },
  };
  publish(server);
  render(<Host />);
  expect(
    await prepare(trigger(), "select", "environments/staging")
  ).toMatchObject(done);
  expect(await prepare(trigger(), "select", "")).toMatchObject(manual);
  server = {
    ...server,
    environment: "environments/prod",
    effectiveEnvironment: "environments/prod",
  };
  expect(
    await prepare(trigger(), "select", "environments/staging")
  ).toMatchObject(manual);
  expect(databaseServiceClientConnect.updateDatabase).not.toHaveBeenCalled();
});
test("native label save and environment selection do not consume or wait for an Agent permit", async () => {
  render(<Host />);
  fireEvent.change(screen.getByDisplayValue("ops"), {
    target: { value: "svc" },
  });
  const p = await prepare(button("common.save"), "click");
  if (p.kind !== "confirm-api") throw Error("expected label confirmation");
  const env = await prepare(trigger(), "select", "environments/prod");
  if (env.kind !== "confirm-api") throw Error("expected env confirmation");
  await act(async () => {
    fireEvent.click(button("common.save"));
  });
  expect(databaseServiceClientConnect.updateDatabase).toHaveBeenCalledTimes(1);
  expect(await validateCapturedDomMutation(p.origin, context)).toBe(false);
  expect(await validateCapturedDomMutation(env.origin, context)).toBe(true);
  fireEvent.click(trigger());
  await act(async () => {
    fireEvent.click(screen.getByRole("option", { name: /Prod/i }));
  });
  expect(databaseServiceClientConnect.updateDatabase).toHaveBeenCalledTimes(2);
  expect(await validateCapturedDomMutation(env.origin, context)).toBe(false);
});
test("read-only post-write refresh keeps newer edits and will not overwrite a newer native cache entry", async () => {
  render(<Host />);
  fireEvent.change(screen.getByDisplayValue("ops"), {
    target: { value: "committed" },
  });
  const p = await prepare(button("common.save"), "click");
  if (p.kind !== "confirm-api") throw Error("expected confirmation");
  fireEvent.change(screen.getByDisplayValue("committed"), {
    target: { value: "newer" },
  });
  server = { ...db, labels: { team: "committed" } };
  await act(() => refreshCommittedDomMutation(p.origin, context));
  expect(screen.getByDisplayValue("newer")).toBeTruthy();
  const later = { ...db, labels: { team: "other native" } };
  let resolve!: (v: typeof db) => void;
  vi.mocked(databaseServiceClientConnect.getDatabase).mockImplementationOnce(
    () =>
      new Promise((r) => {
        resolve = r;
      })
  );
  const read = refreshCommittedDomMutation(p.origin, context);
  publish(later);
  resolve(server);
  await act(() => read);
  expect(useAppStore.getState().databasesByName[db.name]).toBe(later);
  expect(await prepare(button("common.revert"), "click")).toMatchObject(done);
  expect(screen.getByDisplayValue("other native")).toBeTruthy();
  expect(databaseServiceClientConnect.updateDatabase).not.toHaveBeenCalled();
});
test("observed revocation or an unavailable policy leaves dirty rows for the member", async () => {
  render(<Host />);
  fireEvent.change(screen.getByDisplayValue("ops"), {
    target: { value: "svc" },
  });
  const p = await prepare(button("common.save"), "click");
  if (p.kind !== "confirm-api") throw Error("expected confirmation");
  vi.mocked(projectServiceClientConnect.getIamPolicy).mockResolvedValue(
    create(IamPolicySchema)
  );
  expect(await validateCapturedDomMutation(p.origin, context)).toBe(false);
  vi.mocked(projectServiceClientConnect.getIamPolicy).mockRejectedValue(
    new Error("unavailable")
  );
  expect(await prepare(button("common.save"), "click")).toMatchObject(manual);
  expect(screen.getByDisplayValue("svc")).toBeTruthy();
  expect(databaseServiceClientConnect.updateDatabase).not.toHaveBeenCalled();
});
test("database baseline ABA within one React batch invalidates a permission without replacing the draft", async () => {
  render(<Host />);
  fireEvent.change(screen.getByDisplayValue("ops"), {
    target: { value: "svc" },
  });
  const p = await prepare(button("common.save"), "click");
  if (p.kind !== "confirm-api") throw Error("expected confirmation");
  act(() => {
    useAppStore.setState({
      databasesByName: { [db.name]: { ...db, labels: { team: "other" } } },
    });
    useAppStore.setState({ databasesByName: { [db.name]: db } });
  });
  expect(await validateCapturedDomMutation(p.origin, context)).toBe(false);
  expect(screen.getByDisplayValue("svc")).toBeTruthy();
});
test("unmount during a post-success read cannot publish its result", async () => {
  const view = render(<Host />);
  const p = await prepare(trigger(), "select", "environments/prod");
  if (p.kind !== "confirm-api") throw Error("expected confirmation");
  let resolve!: (v: typeof db) => void;
  vi.mocked(databaseServiceClientConnect.getDatabase).mockImplementationOnce(
    () =>
      new Promise((r) => {
        resolve = r;
      })
  );
  const read = refreshCommittedDomMutation(p.origin, context);
  view.unmount();
  resolve({ ...db, effectiveEnvironment: "environments/prod" });
  await read.catch(() => {});
  expect(useAppStore.getState().databasesByName[db.name]).toBe(db);
  expect(databaseServiceClientConnect.updateDatabase).not.toHaveBeenCalled();
});
test("selected environment removal after validation invalidates the final synchronous dispatch guard", async () => {
  render(<Host />);
  const p = await prepare(trigger(), "select", "environments/prod");
  if (p.kind !== "confirm-api") throw Error("expected confirmation");
  expect(await validateCapturedDomMutation(p.origin, context)).toBe(true);
  act(() =>
    useAppStore.setState({
      environmentList: STUB_ENVIRONMENTS.filter((env) => env.id !== "prod"),
    })
  );
  expect(isCapturedDomMutationCurrent(p.origin, context)).toBe(false);
});
