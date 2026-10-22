// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  databaseServiceClientConnect,
  instanceServiceClientConnect,
  projectServiceClientConnect,
} from "@/api";
import { ListDatabasesRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import { ListInstancesRequestSchema } from "@/types/proto-es/v1/instance_service_pb";
import { SearchProjectsRequestSchema } from "@/types/proto-es/v1/project_service_pb";

const workspaceAttentionTool = vi.fn();
const mocks = vi.hoisted(() => ({
  listDatabases: vi.fn(),
  listInstances: vi.fn(),
  searchProjects: vi.fn(),
}));

vi.mock("@/api", () => ({
  databaseServiceClientConnect: { listDatabases: mocks.listDatabases },
  instanceServiceClientConnect: { listInstances: mocks.listInstances },
  projectServiceClientConnect: { searchProjects: mocks.searchProjects },
}));

vi.mock("./workspaceAttention", () => ({
  createWorkspaceAttentionTool: () => workspaceAttentionTool,
}));

import {
  collectWorkspaceOverview,
  createWorkspaceOverviewTool,
  extractWorkspaceOverviewEvidence,
} from "./workspaceOverview";

describe("workspaceOverview", () => {
  beforeEach(() => {
    workspaceAttentionTool.mockReset();
    mocks.searchProjects.mockReset();
    mocks.listInstances.mockReset();
    mocks.listDatabases.mockReset();
  });

  test("collects bounded setup readiness and member attention", async () => {
    workspaceAttentionTool.mockResolvedValue(
      JSON.stringify({
        issues: [
          {
            name: "projects/orders/issues/42",
            title: "Add customer table",
            relation: "review",
          },
        ],
        partial: false,
      })
    );
    mocks.searchProjects.mockResolvedValue({
      projects: [{ name: "projects/orders", title: "Orders" }],
      nextPageToken: "",
    } as never);
    mocks.listInstances.mockResolvedValue({
      instances: [{ name: "instances/prod" }],
      nextPageToken: "",
    } as never);
    mocks.listDatabases.mockResolvedValue({
      databases: [{ name: "instances/prod/databases/orders" }],
      nextPageToken: "",
    } as never);

    await expect(
      collectWorkspaceOverview("workspaces/default")
    ).resolves.toEqual({
      workspace: "workspaces/default",
      setup: {
        projects: { status: "available", partial: false, count: 1 },
        instances: { status: "available", partial: false, count: 1 },
        databases: { status: "available", partial: false, count: 1 },
      },
      attention: {
        status: "available",
        partial: false,
        issues: [
          {
            name: "projects/orders/issues/42",
            title: "Add customer table",
            relation: "review",
          },
        ],
      },
      evidence: {
        workspace: "workspaces/default",
        references: [
          {
            kind: "issue",
            name: "projects/orders/issues/42",
            title: "Add customer table",
          },
        ],
      },
    });

    expect(projectServiceClientConnect.searchProjects).toHaveBeenCalledWith(
      create(SearchProjectsRequestSchema, {
        filter: 'exclude_default == true && state == "ACTIVE"',
        pageSize: 3,
        orderBy: "title",
        showDeleted: false,
      })
    );
    expect(instanceServiceClientConnect.listInstances).toHaveBeenCalledWith(
      create(ListInstancesRequestSchema, { pageSize: 3, showDeleted: false })
    );
    expect(databaseServiceClientConnect.listDatabases).toHaveBeenCalledWith(
      create(ListDatabasesRequestSchema, {
        parent: "workspaces/default",
        pageSize: 3,
        showDeleted: false,
      })
    );
  });

  test("keeps successful facets when setup reads are partial or unavailable", async () => {
    workspaceAttentionTool.mockResolvedValue(
      JSON.stringify({ issues: [], partial: true })
    );
    mocks.searchProjects.mockRejectedValue(new Error("projects unavailable"));
    mocks.listInstances.mockResolvedValue({
      instances: [{ name: "instances/prod" }],
      nextPageToken: "more-instances",
    } as never);
    mocks.listDatabases.mockResolvedValue({
      databases: [],
      nextPageToken: "",
    } as never);

    const overview = await collectWorkspaceOverview("workspaces/default");

    expect(overview).toMatchObject({
      setup: {
        projects: { status: "unavailable" },
        instances: { status: "available", partial: true, count: 1 },
        databases: { status: "available", partial: false, count: 0 },
      },
      attention: { status: "available", partial: true, issues: [] },
    });
  });

  test("does not inspect an unbound chat", async () => {
    await expect(
      collectWorkspaceOverview("projects/orders")
    ).resolves.toBeNull();
    expect(workspaceAttentionTool).not.toHaveBeenCalled();
    expect(projectServiceClientConnect.searchProjects).not.toHaveBeenCalled();
    expect(instanceServiceClientConnect.listInstances).not.toHaveBeenCalled();
    expect(databaseServiceClientConnect.listDatabases).not.toHaveBeenCalled();
  });

  test("returns a workspace-bound JSON snapshot for the Agent", async () => {
    workspaceAttentionTool.mockResolvedValue(
      JSON.stringify({ issues: [], partial: false })
    );
    mocks.searchProjects.mockResolvedValue({
      projects: [],
      nextPageToken: "",
    } as never);
    mocks.listInstances.mockResolvedValue({
      instances: [],
      nextPageToken: "",
    } as never);
    mocks.listDatabases.mockResolvedValue({
      databases: [],
      nextPageToken: "",
    } as never);

    await expect(
      createWorkspaceOverviewTool({ workspaceName: "workspaces/default" })()
    ).resolves.toContain('"workspace":"workspaces/default"');
  });

  test("extracts valid cross-project issue evidence", () => {
    expect(
      extractWorkspaceOverviewEvidence(
        JSON.stringify({
          evidence: {
            workspace: "workspaces/default",
            references: [
              {
                kind: "issue",
                name: "projects/orders/issues/42",
                title: "Add customer table",
              },
              { kind: "database", name: "instances/prod/databases/orders" },
            ],
          },
        })
      )
    ).toEqual({
      workspace: "workspaces/default",
      references: [
        {
          kind: "issue",
          name: "projects/orders/issues/42",
          title: "Add customer table",
        },
      ],
    });
  });
});
