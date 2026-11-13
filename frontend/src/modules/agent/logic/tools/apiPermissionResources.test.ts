import { describe, expect, it } from "vitest";
import { getApiPermissionResources } from "./apiPermissionResources";

describe("getApiPermissionResources", () => {
  it("does not interpret arbitrary connection parameters as permission targets", () => {
    expect(
      getApiPermissionResources("InstanceService/UpdateInstance", {
        instance: {
          name: "projects/a/instances/example",
          dataSources: [
            {
              extraConnectionParameters: { project: "projects/b" },
            },
          ],
        },
        updateMask: "dataSources",
      })
    ).toEqual(["projects/a/instances/example"]);
  });

  it("ignores unmasked resource data in a full update payload", () => {
    expect(
      getApiPermissionResources("InstanceService/UpdateInstance", {
        instance: {
          name: "projects/a/instances/example",
          title: "Renamed instance",
          dataSources: [
            {
              extraConnectionParameters: { project: "projects/b" },
            },
          ],
        },
        updateMask: "title",
      })
    ).toEqual(["projects/a/instances/example"]);
  });

  it("uses an annotated request parent instead of a nested create payload", () => {
    expect(
      getApiPermissionResources("InstanceService/CreateInstance", {
        parent: "projects/a",
        instance: { name: "projects/b/instances/example" },
      })
    ).toEqual(["projects/a"]);
  });

  it("does not use a parentless create payload name as its permission target", () => {
    expect(
      getApiPermissionResources("ProjectService/CreateProject", {
        project: { name: "projects/a" },
      })
    ).toEqual([""]);
  });

  it("uses the catalog name for UpdateDatabaseCatalog", () => {
    expect(
      getApiPermissionResources(
        "DatabaseCatalogService/UpdateDatabaseCatalog",
        {
          catalog: {
            name: "instances/example/databases/example/catalog",
            schemas: [{ name: "projects/not-a-target" }],
          },
        }
      )
    ).toEqual(["instances/example/databases/example/catalog"]);
  });

  it("uses annotated batch names without scanning nested data", () => {
    expect(
      getApiPermissionResources("ProjectService/BatchDeleteProjects", {
        names: ["projects/a", "projects/b"],
      })
    ).toEqual(["projects/a", "projects/b"]);
  });

  it("uses the batch parent and each actual nested request target", () => {
    expect(
      getApiPermissionResources("InstanceService/BatchSyncInstances", {
        parent: "projects/a",
        requests: [
          { name: "projects/a/instances/one" },
          { name: "projects/a/instances/two" },
        ],
      })
    ).toEqual([
      "projects/a",
      "projects/a/instances/one",
      "projects/a/instances/two",
    ]);
  });

  it("defers batch implicit creations to backend authorization", () => {
    expect(
      getApiPermissionResources("InstanceService/BatchUpdateInstances", {
        parent: "projects/a",
        requests: [
          {
            instance: { name: "projects/a/instances/example" },
            allowMissing: true,
          },
        ],
      })
    ).toBeUndefined();
  });

  it("uses an annotated resource field for IAM policy calls", () => {
    expect(
      getApiPermissionResources("ProjectService/SetIamPolicy", {
        resource: "projects/a",
        policy: { bindings: [{ project: "projects/b" }] },
      })
    ).toEqual(["projects/a"]);
  });

  it("resolves known workspace lists", () => {
    expect(
      getApiPermissionResources("ProjectService/ListProjects", {})
    ).toEqual([""]);
  });

  it("defers unknown operations and malformed target payloads", () => {
    expect(
      getApiPermissionResources("ProjectService/DoesNotExist", {
        name: "projects/a",
      })
    ).toBeUndefined();
    expect(
      getApiPermissionResources("InstanceService/UpdateInstance", {
        instance: { title: "Missing name" },
      })
    ).toBeUndefined();
  });
});
