import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { act, render } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import {
  getRegisteredAgentSqlEditorHost,
  registerAgentSqlEditorHost,
} from "@/modules/agent/logic/sqlEditorHost";
import { useAgentStore } from "@/modules/agent/store/agent";
import { useSQLEditorEditorStore } from "@/modules/sql-editor/store/editor";
import { useSQLEditorTabsStore } from "@/modules/sql-editor/store/tab";
import { useAppStore } from "@/stores/app";
import { Engine, State } from "@/types/proto-es/v1/common_pb";
import { DatabaseSchema$ } from "@/types/proto-es/v1/database_service_pb";
import { IamPolicySchema } from "@/types/proto-es/v1/iam_policy_pb";
import {
  InstanceResourceSchema,
  InstanceSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { RoleSchema } from "@/types/proto-es/v1/role_service_pb";
import { QueryOption_RedisRunCommandsOn } from "@/types/proto-es/v1/sql_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import { WorkspaceSchema } from "@/types/proto-es/v1/workspace_service_pb";
import { SQLEditorAgentHost } from "./SQLEditorAgentHost";

const projectA = create(ProjectSchema, {
  name: "projects/alpha",
  state: State.ACTIVE,
});
const projectB = create(ProjectSchema, {
  name: "projects/beta",
  state: State.ACTIVE,
});
const workspaceDb = create(DatabaseSchema$, {
  name: "instances/shared/databases/app",
  project: projectA.name,
  state: State.ACTIVE,
  instanceResource: create(InstanceResourceSchema, {
    name: "instances/shared",
    engine: Engine.POSTGRES,
    activation: true,
    dataSources: [{ id: "read-replica", username: "private-user" }],
  }),
});
const projectDb = create(DatabaseSchema$, {
  name: "projects/beta/instances/dedicated/databases/app",
  project: projectB.name,
  state: State.ACTIVE,
  instanceResource: create(InstanceResourceSchema, {
    name: "projects/beta/instances/dedicated",
    engine: Engine.COSMOSDB,
    activation: true,
    dataSources: [{ id: "admin" }],
  }),
});
const workspaceRole = create(RoleSchema, {
  name: "roles/metadataReader",
  permissions: ["bb.databases.get"],
});
const workspace = create(WorkspaceSchema, { name: "workspaces/default" });
const member = create(UserSchema, {
  name: "users/member@example.com",
  email: "member@example.com",
  workspace: workspace.name,
});

function focusProject(project: string) {
  useSQLEditorEditorStore.getState().setProject(project);
}

function addDatabaseTab(database: typeof workspaceDb) {
  return useSQLEditorTabsStore.getState().addTab({
    connection: {
      database: database.name,
      instance: database.instanceResource?.name ?? "",
    },
  });
}

function capture() {
  return getRegisteredAgentSqlEditorHost()?.captureContext();
}

function readySnapshot() {
  const context = capture();
  return context?.kind === "ready" ? context.snapshot : undefined;
}

beforeEach(() => {
  useAppStore.setState({
    workspace,
    currentUser: member,
    currentUserName: member.name,
    workspacePolicy: create(IamPolicySchema, {
      bindings: [
        {
          role: workspaceRole.name,
          members: ["user:member@example.com"],
        },
      ],
    }),
    roles: [workspaceRole],
    projectsByName: { [projectA.name]: projectA, [projectB.name]: projectB },
    databasesByName: {
      [workspaceDb.name]: workspaceDb,
      [projectDb.name]: projectDb,
    },
    hasFeature: (feature) =>
      feature === PlanFeature.FEATURE_BATCH_QUERY ||
      feature === PlanFeature.FEATURE_DATABASE_GROUPS,
  });
  useSQLEditorTabsStore.setState({
    initProject: vi.fn().mockResolvedValue(undefined),
  });
  useSQLEditorEditorStore.setState({
    project: "",
    projectContextReady: false,
  });
  useAgentStore.getState().setWorkspaceName(workspace.name, member.email);
});

afterEach(() => {
  useSQLEditorTabsStore.setState(useSQLEditorTabsStore.getInitialState(), true);
  useSQLEditorEditorStore.setState(
    useSQLEditorEditorStore.getInitialState(),
    true
  );
  useAppStore.setState(useAppStore.getInitialState(), true);
  useAgentStore.getState().setWorkspaceName(undefined, "");
  localStorage.clear();
});

test("registers after the Agent binds when the editor mounted first", () => {
  useAgentStore.getState().setWorkspaceName(undefined, "");
  focusProject(projectA.name);
  const tab = addDatabaseTab(workspaceDb);
  const view = render(
    <SQLEditorAgentHost paneRatio={0.35}>Editor</SQLEditorAgentHost>
  );
  act(() => {
    useAgentStore.getState().setWorkspaceName(workspace.name, member.email);
  });
  expect(readySnapshot()).toMatchObject({
    project: projectA.name,
    tab: { id: tab.id },
    connection: { database: workspaceDb.name },
  });
  view.unmount();
});

test.each(["app first", "Agent first"])(
  "restores the same mounted host with an unchanged tab after owner switch (%s)",
  (order) => {
    focusProject(projectA.name);
    const tab = addDatabaseTab(workspaceDb);
    const view = render(
      <SQLEditorAgentHost paneRatio={0.35}>Editor</SQLEditorAgentHost>
    );
    const firstHost = getRegisteredAgentSqlEditorHost();
    const nextWorkspace = create(WorkspaceSchema, {
      name: "workspaces/another",
    });
    const nextMember = create(UserSchema, {
      name: "users/another@example.com",
      email: "another@example.com",
      workspace: nextWorkspace.name,
    });
    const switchApp = () =>
      useAppStore.setState({
        workspace: nextWorkspace,
        currentUser: nextMember,
        currentUserName: nextMember.name,
      });
    const switchAgent = () => {
      useAgentStore.getState().setWorkspaceName(
        nextWorkspace.name,
        nextMember.email
      );
    };

    act(order === "app first" ? switchApp : switchAgent);
    expect(getRegisteredAgentSqlEditorHost()).toBeUndefined();
    act(order === "app first" ? switchAgent : switchApp);
    expect(getRegisteredAgentSqlEditorHost()?.id).toBe(firstHost?.id);
    expect(readySnapshot()?.tab?.id).toBe(tab.id);
    expect(firstHost?.captureContext()).toEqual({ kind: "pending" });
    view.unmount();
  }
);

test("does not re-register when the Agent logs out before app identity clears", () => {
  focusProject(projectA.name);
  const view = render(
    <SQLEditorAgentHost paneRatio={0.35}>Editor</SQLEditorAgentHost>
  );
  const firstHost = getRegisteredAgentSqlEditorHost();
  act(() => {
    useAgentStore.getState().setWorkspaceName(undefined, "");
    focusProject(projectB.name);
  });
  expect(getRegisteredAgentSqlEditorHost()).toBeUndefined();
  expect(firstHost?.captureContext()).toEqual({ kind: "pending" });
  view.unmount();
});

test("does not replace a different host on editor changes or cleanup", () => {
  focusProject(projectA.name);
  const view = render(
    <SQLEditorAgentHost paneRatio={0.35}>Editor</SQLEditorAgentHost>
  );
  const host = getRegisteredAgentSqlEditorHost();
  expect(host).toBeDefined();
  const slot = document.createElement("div");
  document.body.append(slot);
  const removeReplacement = registerAgentSqlEditorHost({
    ...host!,
    id: "replacement",
    slot,
  });
  try {
    act(() => focusProject(projectB.name));
    expect(getRegisteredAgentSqlEditorHost()?.id).toBe("replacement");
    view.unmount();
    expect(getRegisteredAgentSqlEditorHost()?.id).toBe("replacement");
  } finally {
    removeReplacement();
    slot.remove();
  }
});

test.each([
  { sources: [{ id: "read-replica" }], expected: "read-replica" },
  { sources: [{ id: "admin" }], expected: undefined },
])(
  "uses current instance data sources rather than embedded database metadata ($expected)",
  ({ sources, expected }) => {
    focusProject(projectA.name);
    const tab = addDatabaseTab(workspaceDb);
    useSQLEditorTabsStore.getState().updateCurrentTab({
      connection: { ...tab.connection, dataSourceId: "read-replica" },
    });
    const view = render(
      <SQLEditorAgentHost paneRatio={0.35}>Editor</SQLEditorAgentHost>
    );
    expect(readySnapshot()?.connection?.dataSourceId).toBe("read-replica");
    const instance = create(InstanceSchema, {
      name: tab.connection.instance,
      state: State.ACTIVE,
      engine: Engine.POSTGRES,
      activation: true,
      dataSources: sources,
    });
    act(() => {
      useAppStore.setState({ instancesByName: { [instance.name]: instance } });
      if (expected) useAppStore.getState().updateDatabaseInstance(instance);
    });

    expect(readySnapshot()?.project).toBe(projectA.name);
    if (expected) {
      expect(readySnapshot()?.connection).toMatchObject({
        database: workspaceDb.name,
        dataSourceId: expected,
      });
    } else {
      expect(readySnapshot()?.connection).toBeUndefined();
    }
    view.unmount();
  }
);

test("blocks a persisted Project until ready, then captures Project-only and verified Workspace help", () => {
  useSQLEditorEditorStore.setState({ project: projectA.name });
  const view = render(
    <SQLEditorAgentHost paneRatio={0.35}>
      <div>Editor content</div>
    </SQLEditorAgentHost>
  );
  const mounted = getRegisteredAgentSqlEditorHost();
  expect(mounted?.paneRatio).toBe(0.35);
  expect(mounted?.mainColumn).toContainElement(view.getByText("Editor content"));
  expect(mounted?.mainColumn).toContainElement(mounted?.slot ?? null);
  expect(mounted?.mainColumn).toContainElement(mounted?.resizer ?? null);
  expect(capture()).toEqual({ kind: "pending" });

  const published: (string | undefined)[] = [];
  const unsubscribe = useAgentStore.subscribe((state, previous) => {
    if (previous.sqlEditorHost !== state.sqlEditorHost) {
      published.push(state.sqlEditorHost?.().id);
    }
  });
  try {
    act(() => focusProject(projectA.name));
    expect(readySnapshot()).toEqual({ version: 1, project: projectA.name });
    expect(getRegisteredAgentSqlEditorHost()).not.toBe(mounted);
    expect(mounted?.captureContext()).toEqual({
      kind: "ready",
      snapshot: { version: 1, project: projectA.name },
    });
    act(() => {
      useSQLEditorEditorStore.getState().setProjectContextReady(false);
    });
    expect(capture()).toEqual({ kind: "pending" });
    act(() => focusProject(""));
    expect(readySnapshot()).toEqual({ version: 1 });
    expect(published).toEqual([mounted?.id, mounted?.id, mounted?.id]);
    view.unmount();
    expect(getRegisteredAgentSqlEditorHost()).toBeUndefined();
  } finally {
    unsubscribe();
  }
});

test("publishes a new pane ratio without clearing the mounted context", () => {
  const view = render(
    <SQLEditorAgentHost paneRatio={0.35}>Editor</SQLEditorAgentHost>
  );
  const firstHost = getRegisteredAgentSqlEditorHost();
  const published: (number | undefined)[] = [];
  const unsubscribe = useAgentStore.subscribe((state, previous) => {
    if (previous.sqlEditorHost !== state.sqlEditorHost) {
      published.push(state.sqlEditorHost?.().paneRatio);
    }
  });
  try {
    view.rerender(
      <SQLEditorAgentHost paneRatio={0.4}>Editor</SQLEditorAgentHost>
    );
    expect(published).toEqual([0.4]);
    expect(getRegisteredAgentSqlEditorHost()?.id).toBe(firstHost?.id);
  } finally {
    unsubscribe();
    view.unmount();
  }
});

test("publishes live connection, viewer, options, tab and disconnect changes without attaching private editor data", () => {
  focusProject(projectA.name);
  const view = render(
    <SQLEditorAgentHost paneRatio={0.35}>Editor</SQLEditorAgentHost>
  );
  let firstTab = "";
  act(() => {
    firstTab = addDatabaseTab(workspaceDb).id;
    const tabs = useSQLEditorTabsStore.getState();
    tabs.updateCurrentTab({
      statement: "select 'private buffer'",
      selectedStatement: "private selection",
      connection: {
        ...tabs.tabsById.get(firstTab)!.connection,
        schema: "app_query",
        dataSourceId: "read-replica",
      },
      viewState: {
        view: "FUNCTIONS",
        schema: "app_viewer",
        detail: { func: "example_function###0", table: "stale_table" },
      },
    });
    useSQLEditorEditorStore.getState().setResultRowsLimit(250);
  });
  expect(readySnapshot()).toEqual({
    version: 1,
    project: projectA.name,
    tab: { id: firstTab, mode: "SAVED_QUERY" },
    connection: {
      database: workspaceDb.name,
      project: projectA.name,
      engine: Engine.POSTGRES,
      querySchema: "app_query",
      dataSourceId: "read-replica",
    },
    viewerObject: {
      kind: "function",
      name: "example_function",
      schema: "app_viewer",
    },
    rowLimit: 250,
    queryOptions: {
      redisRunCommandsOn: QueryOption_RedisRunCommandsOn.SINGLE_NODE,
    },
    batchActive: false,
  });
  const firstHost = getRegisteredAgentSqlEditorHost();
  act(() => {
    useSQLEditorTabsStore.getState().updateCurrentTab({
      statement: "select 'different private buffer'",
    });
  });
  expect(getRegisteredAgentSqlEditorHost()).toBe(firstHost);

  act(() => focusProject(projectB.name));
  expect(readySnapshot()?.project).toBe(projectB.name);
  expect(readySnapshot()?.connection).toBeUndefined();
  expect(readySnapshot()?.viewerObject).toBeUndefined();
  let secondTab = "";
  act(() => {
    secondTab = addDatabaseTab(projectDb).id;
    const tabs = useSQLEditorTabsStore.getState();
    tabs.updateCurrentTab({
      mode: "DATA_EXPLORER",
      connection: {
        ...tabs.tabsById.get(secondTab)!.connection,
        table: "selected_container",
      },
    });
    useSQLEditorEditorStore.getState().setRedisCommandOption(
      QueryOption_RedisRunCommandsOn.ALL_NODES
    );
  });
  expect(readySnapshot()).toMatchObject({
    tab: { id: secondTab, mode: "DATA_EXPLORER" },
    connection: {
      database: projectDb.name,
      project: projectB.name,
      engine: Engine.COSMOSDB,
      container: "selected_container",
    },
    queryOptions: {
      redisRunCommandsOn: QueryOption_RedisRunCommandsOn.ALL_NODES,
    },
  });
  const secondHost = getRegisteredAgentSqlEditorHost();
  act(() => {
    useSQLEditorTabsStore.getState().updateCurrentTab({
      connection: { instance: projectDb.instanceResource!.name, database: "" },
      mode: "ADMIN",
    });
  });
  expect(getRegisteredAgentSqlEditorHost()).not.toBe(secondHost);
  expect(readySnapshot()?.tab).toEqual({ id: secondTab, mode: "ADMIN" });
  expect(readySnapshot()?.connection).toBeUndefined();
  act(() => {
    useSQLEditorTabsStore.getState().updateCurrentTab({
      connection: { instance: "", database: "" },
    });
    useSQLEditorTabsStore.getState().setCurrentTabId(firstTab);
  });
  expect(readySnapshot()?.tab).toEqual({ id: firstTab, mode: "SAVED_QUERY" });
  expect(readySnapshot()?.connection).toBeUndefined();
  view.unmount();
});

test.each([
  ["uncached", () => useAppStore.setState({ databasesByName: {} })],
  [
    "denied",
    () =>
      useAppStore.setState({
        databaseErrorsByName: {
          [workspaceDb.name]: new ConnectError(
            "Not permitted",
            Code.PermissionDenied
          ),
        },
      }),
  ],
  [
    "no longer permitted",
    () => useAppStore.setState({ workspacePolicy: create(IamPolicySchema) }),
  ],
  [
    "deleted",
    () =>
      useAppStore.setState({
        databasesByName: {
          [workspaceDb.name]: { ...workspaceDb, state: State.DELETED },
        },
      }),
  ],
  [
    "instance archived",
    () =>
      useAppStore.setState({
        databasesByName: {
          [workspaceDb.name]: {
            ...workspaceDb,
            instanceResource: {
              ...workspaceDb.instanceResource!,
              activation: false,
            },
          },
        },
      }),
  ],
  [
    "inconsistent connection",
    () =>
      useSQLEditorTabsStore.getState().updateCurrentTab({
        connection: { database: workspaceDb.name, instance: "instances/other" },
      }),
  ],
])(
  "keeps Project help when a connected database becomes %s",
  (_name, invalidate) => {
    focusProject(projectA.name);
    addDatabaseTab(workspaceDb);
    const view = render(
      <SQLEditorAgentHost paneRatio={0.35}>Editor</SQLEditorAgentHost>
    );
    const before = getRegisteredAgentSqlEditorHost();
    expect(readySnapshot()?.connection?.database).toBe(workspaceDb.name);
    act(() => {
      invalidate();
    });
    expect(getRegisteredAgentSqlEditorHost()).not.toBe(before);
    expect(readySnapshot()?.project).toBe(projectA.name);
    expect(readySnapshot()?.connection).toBeUndefined();
    view.unmount();
  }
);

test("publishes selected batch state but not additional database or group names", () => {
  focusProject(projectA.name);
  addDatabaseTab(workspaceDb);
  const view = render(
    <SQLEditorAgentHost paneRatio={0.35}>Editor</SQLEditorAgentHost>
  );
  expect(readySnapshot()?.batchActive).toBe(false);
  act(() => {
    useSQLEditorTabsStore.getState().updateBatchQueryContext({
      databases: [workspaceDb.name, "instances/another/databases/private"],
    });
  });
  expect(readySnapshot()?.batchActive).toBe(true);
  expect(JSON.stringify(readySnapshot())).not.toContain("private");
  act(() => {
    useSQLEditorTabsStore.getState().updateBatchQueryContext({
      databases: [],
      databaseGroups: ["projects/alpha/databaseGroups/private-group"],
    });
  });
  expect(readySnapshot()?.batchActive).toBe(true);
  expect(JSON.stringify(readySnapshot())).not.toContain("private-group");
  act(() => useAppStore.setState({ hasFeature: () => false }));
  expect(readySnapshot()?.batchActive).toBe(false);
  view.unmount();
});
