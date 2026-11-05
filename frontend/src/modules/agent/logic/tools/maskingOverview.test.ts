// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { silentContextKey } from "@/api/context-key";
import { Algorithm_InnerOuterMask_MaskType } from "@/types/proto-es/v1/setting_service_pb";
import type { AgentPageResource } from "../chatContext";
import { createMaskingOverviewTool } from "./maskingOverview";

const mocks = vi.hoisted(() => ({
  getDatabase: vi.fn(),
  getDatabaseCatalog: vi.fn(),
  getPolicy: vi.fn(),
  getSetting: vi.fn(),
}));

vi.mock("@/api", () => ({
  databaseCatalogServiceClientConnect: {
    getDatabaseCatalog: mocks.getDatabaseCatalog,
  },
  databaseServiceClientConnect: { getDatabase: mocks.getDatabase },
  orgPolicyServiceClientConnect: { getPolicy: mocks.getPolicy },
  settingServiceClientConnect: { getSetting: mocks.getSetting },
}));

import { GetDatabaseCatalogRequestSchema } from "@/types/proto-es/v1/database_catalog_service_pb";
import { GetDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";

describe("masking overview", () => {
  beforeEach(() => {
    mocks.getDatabase.mockReset();
    mocks.getDatabaseCatalog.mockReset();
    mocks.getPolicy.mockReset();
    mocks.getSetting.mockReset();
  });

  test("returns policy and catalog metadata without exposing exemption identities or row values", async () => {
    const project = "projects/sample";
    const database = `${project}/instances/primary/databases/app`;
    const page: AgentPageResource = {
      kind: "database-catalog",
      project,
      names: [database],
    };
    mocks.getPolicy.mockImplementation(async ({ name }: { name: string }) => {
      if (name.endsWith("masking_rule")) {
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

    expect(overview.catalog.columns).toEqual([
      {
        schema: "app",
        table: "users",
        column: "email",
        semanticType: "email-id",
        classification: "confidential",
      },
    ]);
    expect(overview.semanticTypes.types[0].algorithm).toEqual({
      kind: "md5-mask",
      saltConfigured: true,
    });
    expect(overview.semanticTypes.types[1].algorithm).toEqual({
      kind: "inner-outer-mask",
      type: "inner",
      prefixLen: 1,
      suffixLen: 2,
      substitution: "*",
    });
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
