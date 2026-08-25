import { describe, expect, test, vi } from "vitest";
import {
  findFirstPageItem,
  isSampleDatabaseName,
  isSetupProjectName,
  isUserProjectName,
} from "./WorkspaceSetupGuide.utils";

describe("WorkspaceSetupGuide utilities", () => {
  test.each([
    ["projects/app", "projects/default", true],
    ["projects/default", "projects/default", false],
    ["projects/project-sample", "projects/default", false],
  ])(
    "classifies project %s against default %s",
    (name, defaultProject, expected) => {
      expect(isUserProjectName(name, defaultProject)).toBe(expected);
    }
  );

  test("counts the legacy sample project as setup progress", () => {
    expect(
      isSetupProjectName("projects/project-sample", "projects/default")
    ).toBe(true);
    expect(isSetupProjectName("projects/default", "projects/default")).toBe(
      false
    );
  });

  test("classifies sample databases by exact instance resource name", () => {
    const samples = new Set([
      "instances/self-host-sample",
      "projects/app/instances/saas-sample",
    ]);

    expect(
      isSampleDatabaseName(
        "instances/self-host-sample/databases/employee",
        samples
      )
    ).toBe(true);
    expect(
      isSampleDatabaseName(
        "projects/app/instances/saas-sample/databases/employee",
        samples
      )
    ).toBe(true);
    expect(
      isSampleDatabaseName(
        "projects/app/instances/production/databases/main",
        samples
      )
    ).toBe(false);
    expect(
      isSampleDatabaseName(
        "projects/app/instances/sample-production/databases/main",
        samples
      )
    ).toBe(false);
  });

  test("finds a matching item on a later page", async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: ["sample"], nextPageToken: "page-2" })
      .mockResolvedValueOnce({ items: ["owned"], nextPageToken: "" });

    await expect(
      findFirstPageItem(fetchPage, (item) => item === "owned")
    ).resolves.toBe("owned");
    expect(fetchPage).toHaveBeenNthCalledWith(1, "");
    expect(fetchPage).toHaveBeenNthCalledWith(2, "page-2");
  });

  test("stops after exhausting all pages", async () => {
    const fetchPage = vi.fn().mockResolvedValue({
      items: ["sample"],
      nextPageToken: "",
    });

    await expect(
      findFirstPageItem(fetchPage, (item) => item === "owned")
    ).resolves.toBeUndefined();
    expect(fetchPage).toHaveBeenCalledTimes(1);
  });

  test("continues through an empty intermediate page", async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({ items: [], nextPageToken: "page-2" })
      .mockResolvedValueOnce({ items: ["owned"], nextPageToken: "" });

    await expect(findFirstPageItem(fetchPage, () => true)).resolves.toBe(
      "owned"
    );
  });
});
