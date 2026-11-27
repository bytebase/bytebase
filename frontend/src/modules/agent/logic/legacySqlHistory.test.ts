import { create } from "@bufbuild/protobuf";
import {
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  type MockInstance,
  vi,
} from "vitest";
import {
  databaseServiceClientConnect,
  projectServiceClientConnect,
} from "@/api";
import { State } from "@/types/proto-es/v1/common_pb";
import { DatabaseSchema$ as DatabaseSchema } from "@/types/proto-es/v1/database_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import { storageKeyAiConversations } from "@/utils/storage-keys";
import {
  legacySqlTranscript,
  readLegacySqlHistory,
  resolveLegacySqlHistoryProjects,
} from "./legacySqlHistory";

it("does not read storage until Workspace and email are ready", () => {
  expect(readLegacySqlHistory({ workspace: "", email: "" }).status).toBe(
    "identity-unavailable"
  );
});

const identity = {
  workspace: "workspaces/example",
  email: "member@example.com",
};
const key = storageKeyAiConversations(identity.workspace, identity.email);
const record = {
  id: "source",
  created_ts: 1,
  name: "History",
  instance: "instances/test",
  database: "instances/test/databases/demo",
  messageList: [],
};
it.each([
  "{",
  JSON.stringify({ version: 2, conversations: [] }),
  JSON.stringify({
    version: 1,
    conversations: Array.from({ length: 21 }, (_, i) => ({
      ...record,
      id: String(i),
    })),
  }),
  JSON.stringify({
    version: 1,
    conversations: [{ ...record, name: "漢".repeat(350000) }],
  }),
  JSON.stringify({
    version: 1,
    conversations: [{ ...record, messageList: [{ id: "x", author: "tool" }] }],
  }),
  JSON.stringify({ version: 1, conversations: [record, record] }),
  JSON.stringify({
    version: 1,
    conversations: [{ ...record, created_ts: 1e20 }],
  }),
  JSON.stringify({
    version: 1,
    conversations: [
      {
        ...record,
        messageList: [
          {
            id: "m",
            author: "USER",
            created_ts: -1e20,
            content: "x",
            error: "",
            status: "DONE",
          },
        ],
      },
    ],
  }),
])(
  "rejects a malformed or oversized source without mutating storage (%#)",
  (raw) => {
    localStorage.setItem(key, raw);
    expect(readLegacySqlHistory(identity)).toEqual({
      status: "invalid-source",
    });
    expect(localStorage.getItem(key)).toBe(raw);
    localStorage.removeItem(key);
  }
);

it("only reads the exact Workspace and email key", () => {
  const getItem = vi.spyOn(Storage.prototype, "getItem");
  expect(readLegacySqlHistory(identity)).toEqual({
    status: "ready",
    conversations: [],
  });
  expect(getItem).toHaveBeenCalledExactlyOnceWith(key);
  getItem.mockRestore();
});

describe("legacy SQL Project resolution", () => {
  const project = "projects/example";
  const database = "instances/test/databases/demo";
  const transcript = (patch: Partial<typeof record> = {}) =>
    legacySqlTranscript(identity, { ...record, ...patch });
  let getDatabase: MockInstance<
    typeof databaseServiceClientConnect.getDatabase
  >;
  let getProject: MockInstance<typeof projectServiceClientConnect.getProject>;

  beforeEach(() => {
    getDatabase = vi
      .spyOn(databaseServiceClientConnect, "getDatabase")
      .mockImplementation(async ({ name }) =>
        create(DatabaseSchema, { name, project, state: State.ACTIVE })
      );
    getProject = vi
      .spyOn(projectServiceClientConnect, "getProject")
      .mockImplementation(async ({ name }) =>
        create(ProjectSchema, { name, state: State.ACTIVE })
      );
  });
  afterEach(() => vi.restoreAllMocks());

  it("resolves exact databases and deduplicates database and Project reads", async () => {
    const inputs = [
      transcript(),
      transcript({ id: "short", database: "demo" }),
      transcript({ id: "other", database: "instances/test/databases/other" }),
    ];
    const before = structuredClone(inputs);
    const projects = await resolveLegacySqlHistoryProjects(inputs, () => true);
    expect([...projects.values()]).toEqual([project, project, project]);
    expect(getDatabase.mock.calls.map(([request]) => request.name)).toEqual([
      database,
      "instances/test/databases/other",
    ]);
    expect(getProject).toHaveBeenCalledTimes(1);
    expect(inputs).toEqual(before);
  });

  it("keeps colliding short names isolated by their full instance scope", async () => {
    const second = "projects/other/instances/test/databases/demo";
    getDatabase.mockImplementation(async ({ name }) =>
      create(DatabaseSchema, {
        name,
        project: name === second ? "projects/other" : project,
        state: State.ACTIVE,
      })
    );
    const inputs = [
      transcript({ database: "demo" }),
      transcript({
        id: "project-scoped",
        instance: "projects/other/instances/test",
        database: "demo",
      }),
    ];
    const projects = await resolveLegacySqlHistoryProjects(inputs, () => true);
    expect([...projects.values()]).toEqual([project, "projects/other"]);
    expect(getDatabase.mock.calls.map(([request]) => request.name)).toEqual([
      database,
      second,
    ]);
  });

  it.each([
    { database: "instances/test/databases/demo/metadata" },
    { database: "instances/other/databases/demo" },
    { database: "instances/test/databases/-" },
    { database: "instances/test/databases/.." },
    { database: "instances/test/databases/demo\n" },
    { database: "demo", instance: "instances/-" },
    { database: "demo", instance: "projects/-/instances/test" },
    { database: "demo", instance: "instances/test/roles/reader" },
  ])(
    "never guesses a Project for malformed or conflicting target %j",
    async (patch) => {
      expect(
        (await resolveLegacySqlHistoryProjects([transcript(patch)], () => true))
          .size
      ).toBe(0);
      expect(getDatabase).not.toHaveBeenCalled();
      expect(getProject).not.toHaveBeenCalled();
    }
  );

  it.each([
    "denied-database",
    "inactive-database",
    "wrong-database",
    "invalid-project",
    "denied-project",
    "inactive-project",
    "wrong-project",
    "wrong-project-owner",
  ])("leaves %s unscoped", async (failure) => {
    const input =
      failure === "wrong-project-owner"
        ? transcript({
            instance: "projects/other/instances/test",
            database: "projects/other/instances/test/databases/demo",
          })
        : transcript();
    if (failure === "denied-database")
      getDatabase.mockRejectedValue(new Error("Permission denied"));
    if (failure === "inactive-database")
      getDatabase.mockResolvedValue(
        create(DatabaseSchema, {
          name: database,
          project,
          state: State.DELETED,
        })
      );
    if (failure === "wrong-database")
      getDatabase.mockResolvedValue(
        create(DatabaseSchema, {
          name: "instances/other/databases/demo",
          project,
          state: State.ACTIVE,
        })
      );
    if (failure === "invalid-project")
      getDatabase.mockResolvedValue(
        create(DatabaseSchema, {
          name: database,
          project: "projects/-",
          state: State.ACTIVE,
        })
      );
    if (failure === "denied-project")
      getProject.mockRejectedValue(new Error("Permission denied"));
    if (failure === "inactive-project")
      getProject.mockResolvedValue(
        create(ProjectSchema, { name: project, state: State.DELETED })
      );
    if (failure === "wrong-project")
      getProject.mockResolvedValue(
        create(ProjectSchema, { name: "projects/other", state: State.ACTIVE })
      );
    expect(
      (await resolveLegacySqlHistoryProjects([input], () => true)).size
    ).toBe(0);
  });

  it.each([
    "projects/-",
    "projects/example\n",
    "projects/example/instances/test",
    "projects/..",
    "workspaces/example",
  ])(
    "rejects a malformed returned Project %j before fetching it",
    async (invalidProject) => {
      getDatabase.mockResolvedValue(
        create(DatabaseSchema, {
          name: database,
          project: invalidProject,
          state: State.ACTIVE,
        })
      );
      expect(
        (await resolveLegacySqlHistoryProjects([transcript()], () => true)).size
      ).toBe(0);
      expect(getProject).not.toHaveBeenCalled();
    }
  );

  it.each(["before-read", "database-read", "project-read"])(
    "stops when import identity changes at %s",
    async (boundary) => {
      let current = boundary !== "before-read";
      if (boundary === "database-read")
        getDatabase.mockImplementation(async ({ name }) => {
          current = false;
          return create(DatabaseSchema, { name, project, state: State.ACTIVE });
        });
      if (boundary === "project-read")
        getProject.mockImplementation(async ({ name }) => {
          current = false;
          return create(ProjectSchema, { name, state: State.ACTIVE });
        });
      expect(
        (await resolveLegacySqlHistoryProjects([transcript()], () => current))
          .size
      ).toBe(0);
      if (boundary === "before-read")
        expect(getDatabase).not.toHaveBeenCalled();
      if (boundary !== "project-read")
        expect(getProject).not.toHaveBeenCalled();
    }
  );
});
