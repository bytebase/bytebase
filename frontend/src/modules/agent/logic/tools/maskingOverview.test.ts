// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { silentContextKey } from "@/api/context-key";
import { State } from "@/types/proto-es/v1/common_pb";
import { Algorithm_InnerOuterMask_MaskType } from "@/types/proto-es/v1/setting_service_pb";
import type { AgentPageResource } from "../chatContext";
import { createMaskingOverviewTool } from "./maskingOverview";

const mocks = vi.hoisted(() => ({
  getDatabase: vi.fn(),
  getDatabaseCatalog: vi.fn(),
  getPolicy: vi.fn(),
  getSetting: vi.fn(),
  getProject: vi.fn(),
  getMaskingSetup: vi.fn(),
}));

vi.mock("./maskingSetup", () => ({ getMaskingSetup: mocks.getMaskingSetup }));

vi.mock("@/api", () => ({
  databaseCatalogServiceClientConnect: {
    getDatabaseCatalog: mocks.getDatabaseCatalog,
  },
  databaseServiceClientConnect: { getDatabase: mocks.getDatabase },
  orgPolicyServiceClientConnect: { getPolicy: mocks.getPolicy },
  settingServiceClientConnect: { getSetting: mocks.getSetting },
  projectServiceClientConnect: { getProject: mocks.getProject },
}));

import {
  DatabaseCatalogSchema,
  GetDatabaseCatalogRequestSchema,
  ObjectSchema_Type,
  ObjectSchemaSchema,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import { GetDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";

describe("masking overview", () => {
  beforeEach(() => {
    mocks.getDatabase.mockReset();
    mocks.getDatabaseCatalog.mockReset();
    mocks.getPolicy.mockReset();
    mocks.getSetting.mockReset();
    mocks.getProject
      .mockReset()
      .mockImplementation(async ({ name }) => ({ name, state: State.ACTIVE }));
    mocks.getMaskingSetup.mockReset().mockResolvedValue({ steps: [] });
  });

  test("reports fresh UTC time when each overview read completes", async () => {
    vi.useFakeTimers({ toFake: ["Date"] });
    try {
      vi.setSystemTime("2030-03-12T08:00:00.000Z");
      const overview = createMaskingOverviewTool({
        context: "workspaces/default",
      });
      vi.setSystemTime("2030-03-12T08:05:00.000Z");
      mocks.getMaskingSetup.mockImplementationOnce(async () => {
        vi.setSystemTime("2030-03-12T08:05:30.000Z");
        return { steps: [] };
      });

      expect(JSON.parse(await overview()).currentTime).toBe(
        "2030-03-12T08:05:30.000Z"
      );

      vi.setSystemTime("2030-03-12T09:06:00.000Z");
      expect(JSON.parse(await overview()).currentTime).toBe(
        "2030-03-12T09:06:00.000Z"
      );
    } finally {
      vi.useRealTimers();
    }
  });

  test.each(["Asia/Shanghai", "America/New_York"])(
    "reports the browser IANA time zone %s without changing UTC time",
    async (timeZone) => {
      const options = Intl.DateTimeFormat().resolvedOptions();
      const resolvedOptions = vi
        .spyOn(Intl.DateTimeFormat.prototype, "resolvedOptions")
        .mockReturnValue({ ...options, timeZone });
      vi.useFakeTimers({ toFake: ["Date"] });
      try {
        vi.setSystemTime("2030-03-12T08:05:30.000Z");
        const result = JSON.parse(
          await createMaskingOverviewTool({
            context: "workspaces/default",
          })()
        );

        expect(result.timeZone).toBe(timeZone);
        expect(result.currentTime).toBe("2030-03-12T08:05:30.000Z");
      } finally {
        resolvedOptions.mockRestore();
        vi.useRealTimers();
      }
    }
  );

  test("inspects masking setup from workspace context without a masking page", async () => {
    const result = JSON.parse(
      await createMaskingOverviewTool({
        context: "workspaces/default",
      })()
    );

    expect(result.error).toBeUndefined();
    expect(result.page).toBe("workspace");
    expect(result.setup).toEqual({ steps: [] });
    expect(mocks.getSetting).toHaveBeenCalledTimes(2);
    expect(mocks.getMaskingSetup).toHaveBeenCalledWith({
      workspaceName: "workspaces/default",
      projectName: undefined,
      databaseName: undefined,
    });
  });

  test("includes project exemption guidance from a normal project chat", async () => {
    const result = JSON.parse(
      await createMaskingOverviewTool({
        context: "projects/sample",
        workspaceName: "workspaces/default",
      })()
    );

    expect(result.error).toBeUndefined();
    expect(result.page).toBe("project");
    expect(result.project).toBe("projects/sample");
    expect(mocks.getPolicy).toHaveBeenCalledWith(
      expect.objectContaining({
        name: "projects/sample/policies/MASKING_EXEMPTION",
      }),
      expect.anything()
    );
  });

  test.each(["bound database", "catalog page"])(
    "an explicit project selection ignores the database from the %s",
    async (source) => {
      const project = "projects/sample";
      const database = "instances/primary/databases/app";
      const result = JSON.parse(
        await createMaskingOverviewTool({
          context: project,
          workspaceName: "workspaces/default",
          ...(source === "bound database"
            ? { databaseName: database }
            : {
                page: {
                  kind: "database-catalog" as const,
                  project,
                  names: [database],
                },
              }),
        })({ project })
      );

      expect(result.project).toBe(project);
      expect(result.catalog).toEqual({ status: "not-applicable" });
      expect(mocks.getDatabase).not.toHaveBeenCalled();
      expect(mocks.getDatabaseCatalog).not.toHaveBeenCalled();
      expect(mocks.getMaskingSetup).toHaveBeenCalledWith({
        workspaceName: "workspaces/default",
        projectName: project,
        databaseName: undefined,
      });
    }
  );

  test.each([
    [{}, "instances/primary/databases/app"],
    [
      {
        project: "projects/sample",
        database: "instances/secondary/databases/app",
      },
      "instances/secondary/databases/app",
    ],
  ])(
    "preserves default and explicit database selection for %j",
    async (args, database) => {
      const project = "projects/sample";
      mocks.getDatabase.mockResolvedValue({ name: database, project });
      mocks.getDatabaseCatalog.mockResolvedValue({ schemas: [] });
      const result = JSON.parse(
        await createMaskingOverviewTool({
          context: project,
          workspaceName: "workspaces/default",
          databaseName: "instances/primary/databases/app",
        })(args)
      );

      expect(result.catalog).toMatchObject({ status: "available", database });
      expect(mocks.getMaskingSetup).toHaveBeenCalledWith({
        workspaceName: "workspaces/default",
        projectName: project,
        databaseName: database,
      });
    }
  );

  test("does not substitute a captured page's project for workspace chat scope", async () => {
    const result = JSON.parse(
      await createMaskingOverviewTool({
        context: "workspaces/default",
        page: { kind: "masking-exemption", project: "projects/different" },
      })()
    );
    expect(result.project).toBeUndefined();
    expect(result.page).toBe("workspace");
    expect(mocks.getProject).not.toHaveBeenCalled();
    expect(mocks.getMaskingSetup).toHaveBeenCalledWith(
      expect.objectContaining({ projectName: undefined })
    );
  });

  test("requires accessible exact project evidence for a workspace selection", async () => {
    mocks.getProject.mockResolvedValue({
      name: "projects/different",
      state: State.ACTIVE,
    });
    const result = JSON.parse(
      await createMaskingOverviewTool({
        context: "workspaces/default",
      })({ project: "projects/sample" })
    );

    expect(result.error).toContain("project");
    expect(mocks.getPolicy).not.toHaveBeenCalled();
    expect(mocks.getMaskingSetup).not.toHaveBeenCalled();
  });

  test("does not inspect a project outside the chat scope", async () => {
    const result = JSON.parse(
      await createMaskingOverviewTool({
        context: "projects/sample",
        workspaceName: "workspaces/default",
      })({ project: "projects/different" })
    );

    expect(result.error).toContain("scope");
    expect(mocks.getProject).not.toHaveBeenCalled();
    expect(mocks.getPolicy).not.toHaveBeenCalled();
  });

  test("reports known read blockers without calling settings APIs or offering inaccessible links", async () => {
    const readAccess = {
      status: "denied",
      scope: "workspaces/default",
      requiredPermissions: ["bb.settings.get"],
      missingPermissions: ["bb.settings.get"],
    };
    const setup = {
      steps: [
        { id: "semantic-types", readAccess },
        { id: "data-classification", readAccess },
      ],
    };
    mocks.getMaskingSetup.mockResolvedValue(setup);
    mocks.getSetting.mockResolvedValue({
      value: { value: { case: "semanticType", value: { types: [] } } },
    });
    const result = JSON.parse(
      await createMaskingOverviewTool({
        context: "workspaces/default",
      })()
    );

    expect(result.setup).toEqual(setup);
    expect(result.semanticTypes).toMatchObject({
      status: "unavailable",
      reason: "permission-denied",
      missingPermissions: ["bb.settings.get"],
    });
    expect(result.classification.reason).toBe("permission-denied");
    expect(result.openTargets).not.toContainEqual({
      kind: "open-semantic-types",
    });
    expect(result.openTargets).not.toContainEqual({
      kind: "open-data-classification",
    });
    expect(mocks.getSetting).not.toHaveBeenCalled();
  });

  test("does not read a selected catalog belonging to another project", async () => {
    const database = "instances/primary/databases/app";
    mocks.getDatabase.mockResolvedValue({
      name: database,
      project: "projects/different",
    });
    const result = JSON.parse(
      await createMaskingOverviewTool({
        context: "workspaces/default",
      })({ project: "projects/sample", database })
    );

    expect(result.catalog.status).toBe("unavailable");
    expect(result.openTargets).not.toContainEqual(
      expect.objectContaining({ kind: "open-database-catalog" })
    );
    expect(mocks.getDatabaseCatalog).not.toHaveBeenCalled();
    expect(mocks.getMaskingSetup).toHaveBeenCalledWith(
      expect.objectContaining({ databaseName: undefined })
    );
  });

  test("guides first-time setup when no masking policy exists and read access is confirmed", async () => {
    mocks.getPolicy.mockRejectedValue(
      new ConnectError("missing", Code.NotFound)
    );
    mocks.getMaskingSetup.mockResolvedValue({
      steps: [
        {
          id: "global-masking",
          readAccess: {
            status: "allowed",
            scope: "workspaces/default",
            requiredPermissions: ["bb.policies.getMaskingRulePolicy"],
            missingPermissions: [],
          },
        },
      ],
    });

    const result = JSON.parse(
      await createMaskingOverviewTool({ context: "workspaces/default" })()
    );

    expect(result.maskingPolicy).toMatchObject({
      status: "available",
      configured: false,
      rules: [],
    });
    expect(result.openTargets).toContainEqual({ kind: "open-global-masking" });
  });

  test("blocks initial policy setup on a missing create permission without blocking existing-policy updates", async () => {
    mocks.getPolicy.mockRejectedValue(
      new ConnectError("missing", Code.NotFound)
    );
    mocks.getMaskingSetup.mockResolvedValue({
      steps: [
        {
          id: "global-masking",
          readAccess: { status: "allowed" },
          configure: {
            status: "allowed",
            scope: "workspaces/default",
            requiredPermissions: ["bb.policies.updateMaskingRulePolicy"],
            missingPermissions: [],
          },
          additionalRequirements: [
            {
              source: "backend",
              condition: "when-policy-missing",
              permission: "bb.policies.createMaskingRulePolicy",
              status: "denied",
              scope: "workspaces/default",
              requiredPermissions: ["bb.policies.createMaskingRulePolicy"],
              missingPermissions: ["bb.policies.createMaskingRulePolicy"],
            },
          ],
        },
      ],
    });
    const result = JSON.parse(
      await createMaskingOverviewTool({ context: "workspaces/default" })()
    );
    expect(result.setup.steps[0].configure).toMatchObject({
      status: "denied",
      missingPermissions: ["bb.policies.createMaskingRulePolicy"],
    });
    expect(result.setup.steps[0].configure.requiredPermissions).toEqual(
      expect.arrayContaining([
        "bb.policies.createMaskingRulePolicy",
        "bb.policies.updateMaskingRulePolicy",
      ])
    );
  });

  test("does not turn an absent policy into a settings link without confirmed read access", async () => {
    mocks.getPolicy.mockRejectedValue(
      new ConnectError("missing", Code.NotFound)
    );
    const result = JSON.parse(
      await createMaskingOverviewTool({ context: "workspaces/default" })()
    );
    expect(result.openTargets).not.toContainEqual({
      kind: "open-global-masking",
    });
    expect(result.maskingPolicy).toMatchObject({
      status: "unavailable",
      reason: "permission-unverified",
    });
  });

  test("does not offer a settings-page link when route read eligibility is unknown", async () => {
    mocks.getSetting.mockResolvedValue({
      value: { value: { case: "semanticType", value: { types: [] } } },
    });
    const result = JSON.parse(
      await createMaskingOverviewTool({ context: "workspaces/default" })()
    );
    expect(result.semanticTypes.status).toBe("available");
    expect(result.openTargets).not.toContainEqual({
      kind: "open-semantic-types",
    });
  });

  test("returns policy and catalog metadata without exposing exemption identities or row values", async () => {
    mocks.getMaskingSetup.mockResolvedValue({
      steps: [
        "global-masking",
        "semantic-types",
        "data-classification",
        "database-catalog",
        "masking-exemptions",
      ].map((id) => ({ id, readAccess: { status: "allowed" } })),
    });
    const project = "projects/sample";
    const database = `${project}/instances/primary/databases/app`;
    const page: AgentPageResource = {
      kind: "database-catalog",
      project,
      names: [database],
    };
    mocks.getPolicy.mockImplementation(async ({ name }: { name: string }) => {
      if (name.endsWith("MASKING_RULE")) {
        return {
          enforce: true,
          inheritFromParent: false,
          policy: {
            case: "maskingRulePolicy",
            value: {
              rules: [
                {
                  semanticType: "Email",
                  condition: { expression: 'resource.column_name == "email"' },
                },
              ],
            },
          },
        };
      }
      return {
        policy: {
          case: "maskingExemptionPolicy",
          value: {
            exemptions: [
              {
                members: ["user:private@example.com"],
                condition: { expression: 'resource.column_name == "email"' },
              },
            ],
          },
        },
      };
    });
    mocks.getSetting.mockImplementation(async ({ name }: { name: string }) =>
      name.includes("SEMANTIC_TYPES")
        ? {
            value: {
              value: {
                case: "semanticType",
                value: {
                  types: [
                    {
                      id: "email-id",
                      title: "Email",
                      algorithm: {
                        mask: {
                          case: "md5Mask",
                          value: { salt: "never expose this salt" },
                        },
                      },
                    },
                    {
                      id: "phone-id",
                      title: "Phone",
                      algorithm: {
                        mask: {
                          case: "innerOuterMask",
                          value: {
                            type: Algorithm_InnerOuterMask_MaskType.INNER,
                            prefixLen: 1,
                            suffixLen: 2,
                            substitution: "*",
                          },
                        },
                      },
                    },
                  ],
                },
              },
            },
          }
        : {
            value: {
              value: {
                case: "dataClassification",
                value: {
                  configs: [
                    { title: "Default", levels: [], classification: {} },
                  ],
                },
              },
            },
          }
    );
    mocks.getDatabase.mockResolvedValue({ name: database, project });
    mocks.getDatabaseCatalog.mockResolvedValue({
      name: `${database}/catalog`,
      schemas: [
        {
          name: "app",
          tables: [
            {
              name: "users",
              kind: {
                case: "columns",
                value: {
                  columns: [
                    {
                      name: "email",
                      semanticType: "email-id",
                      classification: "confidential",
                    },
                  ],
                },
              },
            },
          ],
        },
      ],
    });

    const result = await createMaskingOverviewTool({
      page,
      workspaceName: "workspaces/default",
    })();
    const overview = JSON.parse(result);

    expect(overview.maskingPolicy.rules).toEqual([
      { semanticType: "Email", condition: 'resource.column_name == "email"' },
    ]);

    expect(overview.catalog.columns).toEqual([
      {
        schema: "app",
        table: "users",
        column: "email",
        semanticType: "email-id",
        classification: "confidential",
      },
    ]);
    expect(overview.semanticTypes.types).toContainEqual(
      expect.objectContaining({
        id: "email-id",
        algorithm: { kind: "md5-mask", saltConfigured: true },
      })
    );
    expect(overview.semanticTypes.types).toContainEqual(
      expect.objectContaining({
        id: "phone-id",
        algorithm: {
          kind: "inner-outer-mask",
          type: "inner",
          prefixLen: 1,
          suffixLen: 2,
          substitution: "*",
        },
      })
    );
    expect(overview.exemptions.exemptions).toEqual([
      {
        memberCount: 1,
        condition: 'resource.column_name == "email"',
      },
    ]);
    expect(overview.openTargets).toEqual(
      expect.arrayContaining([
        { kind: "open-global-masking" },
        { kind: "open-semantic-types" },
        { kind: "open-data-classification" },
        { kind: "open-project-masking-exemptions", project },
        { kind: "open-database-catalog", project, database },
      ])
    );
    expect(result).not.toContain("private@example.com");
    expect(result).not.toContain("never expose this salt");
    expect(result).not.toContain("actual-row-value");
    expect(
      mocks.getDatabase.mock.calls[0][1].contextValues.get(silentContextKey)
    ).toBe(true);
    expect(
      mocks.getDatabaseCatalog.mock.calls[0][1].contextValues.get(
        silentContextKey
      )
    ).toBe(true);
    expect(mocks.getDatabaseCatalog).toHaveBeenCalledWith(
      create(GetDatabaseCatalogRequestSchema, {
        name: `${database}/catalog`,
      }),
      expect.objectContaining({ contextValues: expect.anything() })
    );
  });

  test("refuses project exemption context without an exact project page", async () => {
    const page = {
      kind: "masking-exemption",
      project: "projects/sample",
    } as AgentPageResource;
    const result = await createMaskingOverviewTool({ page })();

    expect(JSON.parse(result)).toEqual({
      error: "A workspace context is required to inspect masking settings.",
    });
    expect(mocks.getPolicy).not.toHaveBeenCalled();
    expect(mocks.getSetting).not.toHaveBeenCalled();
  });

  test("includes the effective built-in algorithms when no semantic types are configured", async () => {
    mocks.getSetting.mockResolvedValue({
      value: { value: { case: "semanticType", value: { types: [] } } },
    });

    const result = JSON.parse(
      await createMaskingOverviewTool({
        page: { kind: "semantic-types" },
        workspaceName: "workspaces/default",
      })()
    );

    expect(result.semanticTypes).toEqual({
      status: "available",
      types: [
        expect.objectContaining({
          id: "bb.default",
          algorithm: { kind: "default-full-mask", substitution: "******" },
        }),
        expect.objectContaining({
          id: "bb.default-partial",
          algorithm: {
            kind: "default-partial-mask",
            stringAndBytes: {
              visiblePortion: "length-dependent middle portion",
              prefix: "**",
              suffix: "**",
            },
          },
        }),
      ],
      totalTypes: 2,
      truncated: false,
    });
  });

  test("ignores stored built-in overrides and counts effective types before truncation", async () => {
    mocks.getSetting.mockResolvedValue({
      value: {
        value: {
          case: "semanticType",
          value: {
            types: [
              {
                id: "bb.default",
                title: "Stored default override",
                algorithm: {
                  mask: {
                    case: "md5Mask",
                    value: { salt: "never expose this salt" },
                  },
                },
              },
              {
                id: "bb.default-partial",
                title: "Stored partial override",
                algorithm: {
                  mask: { case: "fullMask", value: { substitution: "X" } },
                },
              },
              ...Array.from({ length: 100 }, (_, index) => ({
                id: `custom-${index}`,
                title: `Custom ${index}`,
                algorithm: {
                  mask: { case: "fullMask", value: { substitution: "*" } },
                },
              })),
            ],
          },
        },
      },
    });

    const response = await createMaskingOverviewTool({
      page: { kind: "semantic-types" },
      workspaceName: "workspaces/default",
    })();
    const result = JSON.parse(response);

    expect(result.semanticTypes.types).toHaveLength(100);
    expect(result.semanticTypes.totalTypes).toBe(102);
    expect(result.semanticTypes.truncated).toBe(true);
    expect(result.semanticTypes.types[0]).toEqual(
      expect.objectContaining({
        id: "bb.default",
        algorithm: { kind: "default-full-mask", substitution: "******" },
      })
    );
    expect(result.semanticTypes.types[1]).toEqual(
      expect.objectContaining({
        id: "bb.default-partial",
        algorithm: {
          kind: "default-partial-mask",
          stringAndBytes: {
            visiblePortion: "length-dependent middle portion",
            prefix: "**",
            suffix: "**",
          },
        },
      })
    );
    expect(result.semanticTypes.types[99]).toEqual(
      expect.objectContaining({
        id: "custom-97",
        algorithm: { kind: "full-mask", substitution: "*" },
      })
    );
    expect(response).not.toContain("Stored default override");
    expect(response).not.toContain("Stored partial override");
    expect(response).not.toContain("never expose this salt");
  });

  test("looks up an exact saved type beyond the bounded catalog without exposing other types or salts", async () => {
    const types = Array.from({ length: 150 }, (_, index) => ({
      id: `custom-${index}`,
      title: `Custom ${index}`,
      description: "Configured type",
      algorithm: {
        mask: { case: "md5Mask", value: { salt: `private-salt-${index}` } },
      },
    }));
    mocks.getSetting.mockResolvedValue({
      value: { value: { case: "semanticType", value: { types } } },
    });
    const overview = createMaskingOverviewTool({
      context: "workspaces/default",
    });
    const bounded = JSON.parse(await overview());
    expect(bounded.semanticTypes.types).toHaveLength(100);
    expect(bounded.semanticTypes.truncated).toBe(true);

    const response = await overview({ semanticTypeId: "custom-149" });
    const result = JSON.parse(response);
    expect(result.semanticTypes).toEqual({
      status: "available",
      requestedTypeId: "custom-149",
      requestedTypeFound: true,
      types: [
        {
          id: "custom-149",
          title: "Custom 149",
          description: "Configured type",
          algorithm: { kind: "md5-mask", saltConfigured: true },
        },
      ],
      totalTypes: 152,
      truncated: false,
    });
    expect(response).not.toContain("private-salt");
    expect(response).not.toContain('"custom-148"');
  });

  test("refreshes a newly saved exact type instead of substituting the first available type", async () => {
    const type = {
      id: "newly-created",
      title: "Custom type",
      description: "",
      algorithm: {
        mask: { case: "fullMask", value: { substitution: "hidden" } },
      },
    };
    mocks.getSetting.mockImplementation(async () => ({
      value: { value: { case: "semanticType", value: { types: [] } } },
    }));
    const overview = createMaskingOverviewTool({
      context: "workspaces/default",
    });
    expect(
      JSON.parse(await overview({ semanticTypeId: type.id })).semanticTypes
    ).toMatchObject({
      requestedTypeId: type.id,
      requestedTypeFound: false,
      types: [],
      truncated: false,
    });
    mocks.getSetting.mockImplementation(async () => ({
      value: { value: { case: "semanticType", value: { types: [type] } } },
    }));
    expect(
      JSON.parse(await overview({ semanticTypeId: type.id })).semanticTypes
    ).toMatchObject({
      requestedTypeId: type.id,
      requestedTypeFound: true,
      types: [
        {
          id: type.id,
          algorithm: { kind: "full-mask", substitution: "hidden" },
        },
      ],
    });
  });

  test("exact type lookup retains effective builtin behavior and bounded algorithm configuration", async () => {
    mocks.getSetting.mockResolvedValue({
      value: {
        value: {
          case: "semanticType",
          value: {
            types: [
              {
                id: "bb.default",
                title: "Invalid override",
                algorithm: {
                  mask: { case: "md5Mask", value: { salt: "secret" } },
                },
              },
              {
                id: "many-slices",
                title: "Large range",
                algorithm: {
                  mask: {
                    case: "rangeMask",
                    value: {
                      slices: Array.from({ length: 21 }, (_, start) => ({
                        start,
                        end: start + 1,
                        substitution: "*",
                      })),
                    },
                  },
                },
              },
            ],
          },
        },
      },
    });
    const overview = createMaskingOverviewTool({
      context: "workspaces/default",
    });
    const builtin = JSON.parse(
      await overview({ semanticTypeId: "bb.default" })
    );
    expect(builtin.semanticTypes.types).toEqual([
      expect.objectContaining({
        id: "bb.default",
        algorithm: { kind: "default-full-mask", substitution: "******" },
      }),
    ]);
    const ranged = JSON.parse(
      await overview({ semanticTypeId: "many-slices" })
    );
    expect(ranged.semanticTypes.types[0].algorithm).toMatchObject({
      kind: "range-mask",
      totalSlices: 21,
      truncated: true,
    });
    expect(ranged.semanticTypes.types[0].algorithm.slices).toHaveLength(20);
  });

  test("does not bypass semantic-type read authorization for exact lookup", async () => {
    mocks.getMaskingSetup.mockResolvedValue({
      steps: [
        {
          id: "semantic-types",
          readAccess: {
            status: "denied",
            scope: "workspaces/default",
            requiredPermissions: ["bb.settings.get"],
            missingPermissions: ["bb.settings.get"],
          },
        },
        { id: "data-classification", readAccess: { status: "denied" } },
      ],
    });
    const result = JSON.parse(
      await createMaskingOverviewTool({ context: "workspaces/default" })({
        semanticTypeId: "custom-149",
      })
    );
    expect(result.semanticTypes).toMatchObject({
      status: "unavailable",
      reason: "permission-denied",
    });
    expect(mocks.getSetting).not.toHaveBeenCalled();
    expect(result.semanticTypes.requestedTypeFound).toBeUndefined();
  });

  test.each([null, 12, "", " ", "bad\0id", "x".repeat(501)])(
    "rejects invalid exact type ID %j before reading resources",
    async (semanticTypeId) => {
      const result = JSON.parse(
        await createMaskingOverviewTool({ context: "workspaces/default" })({
          semanticTypeId,
        } as never)
      );
      expect(result.error).toMatch(/semantic type/i);
      expect(mocks.getMaskingSetup).not.toHaveBeenCalled();
      expect(mocks.getSetting).not.toHaveBeenCalled();
    }
  );

  test("distinguishes built-in masking from empty configurable full and range algorithms", async () => {
    mocks.getSetting.mockResolvedValue({
      value: {
        value: {
          case: "semanticType",
          value: {
            types: [
              {
                id: "custom-full",
                title: "Custom full",
                algorithm: {
                  mask: { case: "fullMask", value: { substitution: "" } },
                },
              },
              {
                id: "custom-range",
                title: "Custom range",
                algorithm: {
                  mask: { case: "rangeMask", value: { slices: [] } },
                },
              },
            ],
          },
        },
      },
    });

    const result = JSON.parse(
      await createMaskingOverviewTool({
        page: { kind: "semantic-types" },
        workspaceName: "workspaces/default",
      })()
    );

    expect(result.semanticTypes.types).toEqual([
      expect.objectContaining({
        id: "bb.default",
        algorithm: { kind: "default-full-mask", substitution: "******" },
      }),
      expect.objectContaining({
        id: "bb.default-partial",
        algorithm: {
          kind: "default-partial-mask",
          stringAndBytes: {
            visiblePortion: "length-dependent middle portion",
            prefix: "**",
            suffix: "**",
          },
        },
      }),
      expect.objectContaining({
        id: "custom-full",
        algorithm: { kind: "full-mask", substitution: "" },
      }),
      expect.objectContaining({
        id: "custom-range",
        algorithm: {
          kind: "range-mask",
          slices: [],
          totalSlices: 0,
          truncated: false,
        },
      }),
    ]);
  });

  test("includes nested document and array fields with their semantic types", async () => {
    const project = "projects/sample";
    const database = `${project}/instances/primary/databases/app`;
    mocks.getDatabase.mockResolvedValue({ name: database, project });
    mocks.getDatabaseCatalog.mockResolvedValue(
      create(DatabaseCatalogSchema, {
        name: `${database}/catalog`,
        schemas: [
          {
            name: "app",
            tables: [
              {
                name: "users",
                kind: {
                  case: "objectSchema",
                  value: {
                    type: ObjectSchema_Type.OBJECT,
                    kind: {
                      case: "structKind",
                      value: {
                        properties: {
                          contact: {
                            type: ObjectSchema_Type.OBJECT,
                            kind: {
                              case: "structKind",
                              value: {
                                properties: {
                                  email: {
                                    type: ObjectSchema_Type.STRING,
                                    semanticType: "email-id",
                                  },
                                },
                              },
                            },
                          },
                          phones: {
                            type: ObjectSchema_Type.ARRAY,
                            kind: {
                              case: "arrayKind",
                              value: {
                                kind: {
                                  type: ObjectSchema_Type.OBJECT,
                                  kind: {
                                    case: "structKind",
                                    value: {
                                      properties: {
                                        number: {
                                          type: ObjectSchema_Type.STRING,
                                          semanticType: "phone-id",
                                        },
                                      },
                                    },
                                  },
                                },
                              },
                            },
                          },
                          tags: {
                            type: ObjectSchema_Type.ARRAY,
                            kind: {
                              case: "arrayKind",
                              value: {
                                kind: { type: ObjectSchema_Type.STRING },
                              },
                            },
                          },
                        },
                      },
                    },
                  },
                },
              },
            ],
          },
        ],
      })
    );

    const result = JSON.parse(
      await createMaskingOverviewTool({
        page: { kind: "database-catalog", project, names: [database] },
        workspaceName: "workspaces/default",
      })()
    );

    expect(result.catalog).toEqual({
      status: "available",
      database,
      columns: [
        {
          schema: "app",
          table: "users",
          column: "contact.email",
          semanticType: "email-id",
          classification: "",
        },
        {
          schema: "app",
          table: "users",
          column: "phones.number",
          semanticType: "phone-id",
          classification: "",
        },
        {
          schema: "app",
          table: "users",
          column: "tags",
          semanticType: "",
          classification: "",
        },
      ],
      totalColumns: 3,
      truncated: false,
    });
  });

  test("bounds combined relational columns and document fields without losing the total count", async () => {
    const project = "projects/sample";
    const database = `${project}/instances/primary/databases/app`;
    mocks.getDatabase.mockResolvedValue({ name: database, project });
    mocks.getDatabaseCatalog.mockResolvedValue(
      create(DatabaseCatalogSchema, {
        name: `${database}/catalog`,
        schemas: [
          {
            name: "app",
            tables: [
              {
                name: "users",
                kind: {
                  case: "columns",
                  value: { columns: [{ name: "email" }] },
                },
              },
              {
                name: "documents",
                kind: {
                  case: "objectSchema",
                  value: {
                    type: ObjectSchema_Type.OBJECT,
                    kind: {
                      case: "structKind",
                      value: {
                        properties: Object.fromEntries(
                          Array.from({ length: 200 }, (_, index) => [
                            `field_${index}`,
                            create(ObjectSchemaSchema, {
                              type: ObjectSchema_Type.STRING,
                              semanticType: "email-id",
                            }),
                          ])
                        ),
                      },
                    },
                  },
                },
              },
            ],
          },
        ],
      })
    );

    const result = JSON.parse(
      await createMaskingOverviewTool({
        page: { kind: "database-catalog", project, names: [database] },
        workspaceName: "workspaces/default",
      })()
    );

    expect(result.catalog.columns).toHaveLength(200);
    expect(result.catalog.totalColumns).toBe(201);
    expect(result.catalog.truncated).toBe(true);
    expect(result.catalog.columns[199]).toEqual({
      schema: "app",
      table: "documents",
      column: "field_198",
      semanticType: "email-id",
      classification: "",
    });
  });

  test("verifies a catalog database belongs to the page project before reading its catalog", async () => {
    const page: AgentPageResource = {
      kind: "database-catalog",
      project: "projects/sample",
      names: ["projects/sample/instances/primary/databases/app"],
    };
    mocks.getPolicy.mockRejectedValue(new Error("unavailable"));
    mocks.getSetting.mockRejectedValue(new Error("unavailable"));
    mocks.getDatabase.mockResolvedValue({
      name: "projects/other/instances/primary/databases/app",
      project: "projects/other",
    });

    const result = JSON.parse(
      await createMaskingOverviewTool({
        page,
        workspaceName: "workspaces/default",
      })()
    );

    expect(result.catalog.status).toBe("unavailable");
    expect(mocks.getDatabaseCatalog).not.toHaveBeenCalled();
    expect(mocks.getDatabase).toHaveBeenCalledWith(
      create(GetDatabaseRequestSchema, {
        name: "projects/sample/instances/primary/databases/app",
      }),
      expect.objectContaining({ contextValues: expect.anything() })
    );
  });
});
