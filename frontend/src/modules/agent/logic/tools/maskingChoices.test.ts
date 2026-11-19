// @vitest-environment node

import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { beforeEach, expect, test, vi } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  SemanticTypeSetting_SemanticTypeSchema,
  SettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  getMaskingScopeChoices,
  getMaskingSemanticTypeChoices,
  getUnmaskMethodChoices,
} from "./maskingChoices";

const mocks = vi.hoisted(() => ({
  setup: vi.fn(),
  setting: vi.fn(),
  database: vi.fn(),
  user: "users/member@example.com",
}));
vi.mock("./maskingSetup", () => ({ getMaskingSetup: mocks.setup }));
vi.mock("./resourceChoices", () => ({
  validateResourceChoices: mocks.database,
}));
vi.mock("@/api", () => ({
  settingServiceClientConnect: { getSetting: mocks.setting },
}));
vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      currentUser: { name: mocks.user },
      workspaceResourceName: () => "workspaces/example",
    }),
  },
}));
const allowed = {
  status: "allowed",
  requiredPermissions: [],
  missingPermissions: [],
  scope: "workspaces/example",
};
const setup = () => ({
  project: { name: "projects/sample", state: State.ACTIVE },
  justInTime: {
    eligible: true,
    enabled: true,
    feature: "available",
    create: { ...allowed },
  },
  features: { masking: "available" },
  scopeAccess: {
    project: { ...allowed },
    databasePicker: { ...allowed },
    projects: { ...allowed },
  },
  steps: ["semantic-types", "global-masking", "masking-exemptions"].map(
    (id) => ({ id, readAccess: { ...allowed }, configure: { ...allowed } })
  ),
});
const context = {
  workspaceName: "workspaces/example",
  userName: "users/member@example.com",
};
const methodContext = { ...context, projectName: "projects/sample" };
beforeEach(() => {
  vi.clearAllMocks();
  mocks.user = context.userName;
  mocks.setup.mockResolvedValue(setup());
  mocks.setting.mockResolvedValue(
    create(SettingSchema, {
      value: { value: { case: "semanticType", value: { types: [] } } },
    })
  );
  mocks.database.mockResolvedValue([
    { value: "instances/prod/databases/app", label: "app (Production)" },
  ]);
});
test("offers both eligible unmask methods with no selection or default", async () => {
  const result = await getUnmaskMethodChoices(methodContext);
  expect(result.options.map(({ value }) => value)).toEqual([
    "masking-exemption",
    "just-in-time",
  ]);
  expect(result).not.toHaveProperty("defaultValue");
  expect(result.project).toBe("projects/sample");
  expect(result.needsScope).toBe(false);
  expect(result).not.toHaveProperty("methodExplanation");
});
test("explains the use case, scope, recipients, and expiration of each method", async () => {
  const { options } = await getUnmaskMethodChoices(methodContext);
  expect(options).toEqual([
    {
      value: "masking-exemption",
      label: "Masking exemption",
      description:
        "For access to selected data across different queries, for yourself or selected members. Choose an expiration or no expiration. Existing query access is still required.",
    },
    {
      value: "just-in-time",
      label: "Access Grant",
      description:
        "For a short task or investigation. Request access for yourself to one exact read-only query until expiration, under the project's approval policy. Unmask covers its entire result.",
    },
  ]);
});
test.each([
  {
    blocked: "query access",
    reason:
      "A masking exemption cannot grant the query access you are missing.",
  },
  {
    blocked: "permission",
    reason:
      "You do not have all permissions required to prepare a masking exemption.",
  },
  {
    blocked: "read permission",
    reason:
      "You do not have all permissions required to prepare a masking exemption.",
  },
  {
    blocked: "feature",
    reason: "Data masking is not available under the current subscription.",
  },
  {
    blocked: "unknown permission",
    reason: "Masking exemption availability could not be verified.",
  },
  {
    blocked: "unknown feature",
    reason: "Masking exemption availability could not be verified.",
  },
])(
  "explains why an exemption is excluded by $blocked",
  async ({ blocked, reason }) => {
    const state = setup();
    if (blocked === "permission") state.steps[2].configure.status = "denied";
    if (blocked === "read permission")
      state.steps[2].readAccess.status = "denied";
    if (blocked === "feature") state.features.masking = "unavailable";
    if (blocked === "unknown permission")
      state.steps[2].configure.status = "unknown";
    if (blocked === "unknown feature") state.features.masking = "unknown";
    mocks.setup.mockResolvedValue(state);

    const result = await getUnmaskMethodChoices({
      ...methodContext,
      queryAccessMissing: blocked === "query access",
    });

    expect(result.options.map(({ value }) => value)).toEqual(["just-in-time"]);
    expect(result).toHaveProperty("methodExplanation", {
      label: result.options[0].label,
      description: result.options[0].description,
      alternativeReason: reason,
    });
  }
);
test.each([
  {
    blocked: "another member",
    reason: "Access Grants can only be requested for yourself.",
  },
  {
    blocked: "project setting",
    reason: "Access Grants are disabled in this project.",
  },
  {
    blocked: "permission",
    reason:
      "You do not have permission to request an Access Grant in this project.",
  },
  {
    blocked: "feature",
    reason: "Access Grants are not available under the current subscription.",
  },
  {
    blocked: "unknown permission",
    reason: "Access Grant availability could not be verified.",
  },
  {
    blocked: "unknown feature",
    reason: "Access Grant availability could not be verified.",
  },
])(
  "explains why an Access Grant is excluded by $blocked",
  async ({ blocked, reason }) => {
    const state = setup();
    state.justInTime.eligible = false;
    if (blocked === "project setting") state.justInTime.enabled = false;
    if (blocked === "permission") state.justInTime.create.status = "denied";
    if (blocked === "feature") state.justInTime.feature = "unavailable";
    if (blocked === "unknown permission")
      state.justInTime.create.status = "unknown";
    if (blocked === "unknown feature") state.justInTime.feature = "unknown";
    mocks.setup.mockResolvedValue(state);

    const result = await getUnmaskMethodChoices({
      ...methodContext,
      forCurrentUser: blocked !== "another member",
    });

    expect(result.options.map(({ value }) => value)).toEqual([
      "masking-exemption",
    ]);
    expect(result).toHaveProperty("methodExplanation", {
      label: result.options[0].label,
      description: result.options[0].description,
      alternativeReason: reason,
    });
  }
);
test.each(["denied", "unknown"])(
  "excludes exemption with %s configure access",
  async (status) => {
    const state = setup();
    state.steps[2].configure.status = status;
    mocks.setup.mockResolvedValue(state);
    expect(
      (await getUnmaskMethodChoices(methodContext)).options.map(
        ({ value }) => value
      )
    ).toEqual(["just-in-time"]);
  }
);
test("excludes an unreadable exemption without hiding independently eligible JIT", async () => {
  const state = setup();
  state.steps[2].readAccess.status = "unknown";
  mocks.setup.mockResolvedValue(state);
  expect(
    (await getUnmaskMethodChoices(methodContext)).options.map(
      ({ value }) => value
    )
  ).toEqual(["just-in-time"]);
});
test("does not offer JIT when its feature, setting, or create permission is ineligible", async () => {
  const state = setup();
  state.justInTime.eligible = false;
  mocks.setup.mockResolvedValue(state);
  expect(
    (await getUnmaskMethodChoices(methodContext)).options.map(
      ({ value }) => value
    )
  ).toEqual(["masking-exemption"]);
});
test("known missing query access excludes an exemption even when it can be configured", async () => {
  expect(
    (
      await getUnmaskMethodChoices({
        ...methodContext,
        queryAccessMissing: true,
      })
    ).options.map(({ value }) => value)
  ).toEqual(["just-in-time"]);
});
test("requests for other members never offer a self-only Access Grant", async () => {
  expect(
    (
      await getUnmaskMethodChoices({ ...methodContext, forCurrentUser: false })
    ).options.map(({ value }) => value)
  ).toEqual(["masking-exemption"]);
});
test("returns blockers instead of unusable choices when neither method is eligible", async () => {
  const state = setup();
  state.justInTime.eligible = false;
  mocks.setup.mockResolvedValue(state);
  const result = await getUnmaskMethodChoices({
    ...methodContext,
    queryAccessMissing: true,
  });
  expect(result.options).toEqual([]);
  expect(result.setup.justInTime.eligible).toBe(false);
  expect(result).not.toHaveProperty("methodExplanation");
});
test("requires project scope before choosing a method", async () => {
  const result = await getUnmaskMethodChoices(context);
  expect(result.needsScope).toBe(true);
  expect(result.options).toEqual([]);
  expect(result).not.toHaveProperty("methodExplanation");
});
test("does not offer either method for a deleted project", async () => {
  const state = setup();
  state.project.state = State.DELETED;
  state.justInTime.eligible = false;
  mocks.setup.mockResolvedValue(state);
  expect((await getUnmaskMethodChoices(methodContext)).options).toEqual([]);
});
test("preserves an independently verified database in the selected project", async () => {
  const result = await getUnmaskMethodChoices({
    ...methodContext,
    database: "instances/prod/databases/app",
  });
  expect(result.database).toBe("instances/prod/databases/app");
  expect(mocks.database).toHaveBeenCalledWith(
    "database",
    ["instances/prod/databases/app"],
    "projects/sample",
    "workspaces/example"
  );
});
test("rejects a changed or inaccessible database before showing methods", async () => {
  mocks.database.mockResolvedValue(undefined);
  await expect(
    getUnmaskMethodChoices({
      ...methodContext,
      database: "instances/prod/databases/app",
    })
  ).rejects.toThrow(/database/i);
});
test("rejects account drift while computing method availability", async () => {
  mocks.setup.mockImplementation(async () => {
    mocks.user = "users/other@example.com";
    return setup();
  });
  await expect(getUnmaskMethodChoices(methodContext)).rejects.toThrow(
    /context/
  );
});
test("offers built-in full and partial types before the semantic types setting exists", async () => {
  mocks.setting.mockRejectedValue(new ConnectError("missing", Code.NotFound));

  const choices = await getMaskingSemanticTypeChoices(context);

  expect(choices.map((option) => option.value)).toEqual([
    "semantic-type:bb.default",
    "semantic-type:bb.default-partial",
    "action:create-semantic-type",
  ]);
});
test("collects custom masking behavior with the creation choice", async () => {
  const choices = await getMaskingSemanticTypeChoices(context);

  expect(choices.at(-1)).toMatchObject({
    value: "action:create-semantic-type",
    kind: "custom-input",
    placeholder: "Describe how values should be masked",
  });
  expect(choices.slice(0, -1).every((option) => !option.kind)).toBe(true);
});
test("lists all types including duplicate titles/configurations, with creation last and no default", async () => {
  mocks.setting.mockResolvedValue(
    create(SettingSchema, {
      value: {
        value: {
          case: "semanticType",
          value: {
            types: Array.from({ length: 110 }, (_, i) =>
              create(SemanticTypeSetting_SemanticTypeSchema, {
                id: `custom-${i}`,
                title: "Same title",
                description: "Same configuration",
                algorithm: {
                  mask: { case: "md5Mask", value: { salt: "must-not-leak" } },
                },
              })
            ),
          },
        },
      },
    })
  );
  const choices = await getMaskingSemanticTypeChoices(context);
  expect(choices).toHaveLength(113);
  expect(choices.slice(0, 2).map((option) => option.value)).toEqual([
    "semantic-type:bb.default",
    "semantic-type:bb.default-partial",
  ]);
  expect(choices[111].value).toBe("semantic-type:custom-109");
  expect(choices.at(-1)?.value).toBe("action:create-semantic-type");
  expect(choices[2].label).not.toBe(choices[3].label);
  expect(JSON.stringify(choices)).not.toContain("must-not-leak");
});
test.each(["allowed", "denied", "unknown"])(
  "keeps action-like type IDs selectable with %s configure access",
  async (status) => {
    const state = setup();
    state.steps[0].configure.status = status;
    mocks.setup.mockResolvedValue(state);
    mocks.setting.mockResolvedValue(
      create(SettingSchema, {
        value: {
          value: {
            case: "semanticType",
            value: {
              types: [
                { id: "create-semantic-type", title: "Email" },
                { id: "action:create-semantic-type", title: "Name" },
                { id: "semantic-type:custom-email", title: "Phone" },
              ],
            },
          },
        },
      })
    );

    const choices = await getMaskingSemanticTypeChoices(context);
    const values = choices.map((option) => option.value);
    expect(values).toEqual([
      "semantic-type:bb.default",
      "semantic-type:bb.default-partial",
      "semantic-type:create-semantic-type",
      "semantic-type:action:create-semantic-type",
      "semantic-type:semantic-type:custom-email",
      ...(status === "allowed" ? ["action:create-semantic-type"] : []),
    ]);
    expect(new Set(values).size).toBe(choices.length);
    expect(choices[2].label).toBe("Email (create-semantic-type)");
  }
);
test.each(["denied", "unknown"])(
  "does not offer creation with %s configure access",
  async (status) => {
    const state = setup();
    state.steps[0].configure.status = status;
    mocks.setup.mockResolvedValue(state);
    expect(
      (await getMaskingSemanticTypeChoices(context)).map(
        (option) => option.value
      )
    ).toEqual(["semantic-type:bb.default", "semantic-type:bb.default-partial"]);
  }
);
test.each(["denied", "unknown"])(
  "blocks unreadable %s type catalogs without calling the settings API",
  async (status) => {
    const state = setup();
    state.steps[0].readAccess.status = status;
    mocks.setup.mockResolvedValue(state);
    await expect(getMaskingSemanticTypeChoices(context)).rejects.toThrow(
      /semantic-types/
    );
    expect(mocks.setting).not.toHaveBeenCalled();
  }
);
test("rejects account drift during a settings read", async () => {
  mocks.setting.mockImplementation(async () => {
    mocks.user = "users/other@example.com";
    return create(SettingSchema, {
      value: { value: { case: "semanticType", value: { types: [] } } },
    });
  });
  await expect(getMaskingSemanticTypeChoices(context)).rejects.toThrow(
    /context/
  );
});
test.each(["rule", "exemption"] as const)(
  "blocks %s scope choices without configure permission",
  async (kind) => {
    const state = setup();
    state.steps.find(
      (step) =>
        step.id === (kind === "rule" ? "global-masking" : "masking-exemptions")
    )!.configure.status = "denied";
    mocks.setup.mockResolvedValue(state);
    await expect(
      getMaskingScopeChoices({
        ...context,
        kind,
        projectName: "projects/sample",
      })
    ).rejects.toThrow(/permission|access/);
    expect(mocks.database).not.toHaveBeenCalled();
  }
);
test("keeps a verified database and project scope but omits an inaccessible picker", async () => {
  const state = setup();
  state.scopeAccess.databasePicker.status = "denied";
  mocks.setup.mockResolvedValue(state);
  const choices = await getMaskingScopeChoices({
    ...context,
    kind: "rule",
    projectName: "projects/sample",
    database: "instances/prod/databases/app",
  });
  expect(choices.map((option) => option.value)).toEqual([
    "instances/prod/databases/app",
    "projects/sample",
    "workspace",
  ]);
});
test("keeps exemption scope project-bound", async () => {
  expect(
    (
      await getMaskingScopeChoices({
        ...context,
        kind: "exemption",
        projectName: "projects/sample",
      })
    ).map((option) => option.value)
  ).toEqual(["projects/sample", "another-database"]);
});
test("collects a project inline when choosing a new rule's project scope", async () => {
  const choices = await getMaskingScopeChoices({ ...context, kind: "rule" });

  expect(choices).toEqual([
    { label: "All workspace databases", value: "workspace" },
    {
      label: "Choose a project",
      value: "choose-project",
      kind: "custom-input",
      placeholder: "Search projects",
    },
  ]);
  expect(
    choices.every((choice) => !Object.hasOwn(choice, "defaultValue"))
  ).toBe(true);
  expect(mocks.database).not.toHaveBeenCalled();
});
test.each(["denied", "unknown"])(
  "does not collect a project with %s project-list access",
  async (status) => {
    const state = setup();
    state.scopeAccess.projects.status = status;
    mocks.setup.mockResolvedValue(state);

    const choices = await getMaskingScopeChoices({ ...context, kind: "rule" });

    expect(choices.map((choice) => choice.value)).toEqual(["workspace"]);
    expect(choices.every((choice) => !choice.kind)).toBe(true);
  }
);
test.each(["rule", "exemption"] as const)(
  "collects the database with the %s another-database choice",
  async (kind) => {
    const choices = await getMaskingScopeChoices({
      ...context,
      kind,
      projectName: "projects/sample",
    });

    expect(
      choices.find((option) => option.value === "another-database")
    ).toMatchObject({
      kind: "custom-input",
      placeholder: "Enter a database name",
    });
    expect(
      choices
        .filter((option) => option.value !== "another-database")
        .every((option) => !option.kind)
    ).toBe(true);
  }
);
