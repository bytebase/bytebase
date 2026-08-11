import { describe, expect, test } from "vitest";
import {
  PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL,
} from "@/app/router/handles";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  autoDatabaseChangelogRoute,
  autoDatabaseRevisionRoute,
  autoDatabaseRoute,
} from "./auto-route";

describe("autoDatabaseRoute", () => {
  test("preserves the canonical project instance parent", () => {
    const database = {
      name: "projects/proj1/instances/inst1/databases/db1",
      project: "projects/proj1",
    } as Database;

    expect(autoDatabaseRoute(database)).toEqual({
      name: PROJECT_V1_ROUTE_DATABASE_DETAIL,
      params: {
        projectId: "proj1",
        instanceId: "inst1",
        databaseName: "db1",
      },
      query: {
        parent: "projects/proj1/instances/inst1",
      },
    });
  });

  test("builds a changelog route from the canonical database route", () => {
    const database = {
      name: "projects/proj1/instances/inst1/databases/db1",
      project: "projects/proj1",
    } as Database;

    expect(autoDatabaseChangelogRoute(database, "42")).toEqual({
      name: PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
      params: {
        projectId: "proj1",
        instanceId: "inst1",
        databaseName: "db1",
        changelogId: "42",
      },
      query: {
        parent: "projects/proj1/instances/inst1",
      },
    });
  });

  test("builds a revision route from the canonical database route", () => {
    const database = {
      name: "projects/proj1/instances/inst1/databases/db1",
      project: "projects/proj1",
    } as Database;

    expect(autoDatabaseRevisionRoute(database, "7")).toEqual({
      name: PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL,
      params: {
        projectId: "proj1",
        instanceId: "inst1",
        databaseName: "db1",
        revisionId: "7",
      },
      query: {
        parent: "projects/proj1/instances/inst1",
      },
    });
  });
});
