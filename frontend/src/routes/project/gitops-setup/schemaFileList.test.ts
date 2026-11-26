import { describe, expect, test } from "vitest";
import { filterSchemaFiles, groupSchemaFiles } from "./schemaFileList";

describe("groupSchemaFiles", () => {
  test("groups files by folder, in path order", () => {
    expect(
      groupSchemaFiles([
        "schemas/public/views/active_users.sql",
        "schemas/public/tables/users.sql",
        "schemas/public/tables/orders.sql",
        "schemas/public/views/order_totals.sql",
      ])
    ).toEqual([
      {
        kind: "folder",
        dir: "schemas/public/tables/",
        paths: [
          "schemas/public/tables/orders.sql",
          "schemas/public/tables/users.sql",
        ],
      },
      {
        kind: "folder",
        dir: "schemas/public/views/",
        paths: [
          "schemas/public/views/active_users.sql",
          "schemas/public/views/order_totals.sql",
        ],
      },
    ]);
  });

  test("a folder with one file is listed as the file", () => {
    expect(
      groupSchemaFiles([
        "schemas/public/sequences.sql",
        "schemas/public/tables/a.sql",
        "schemas/public/tables/b.sql",
      ])
    ).toEqual([
      { kind: "file", path: "schemas/public/sequences.sql" },
      {
        kind: "folder",
        dir: "schemas/public/tables/",
        paths: ["schemas/public/tables/a.sql", "schemas/public/tables/b.sql"],
      },
    ]);
  });
});

describe("filterSchemaFiles", () => {
  const paths = [
    "schemas/billing/tables/invoice.sql",
    "schemas/public/tables/Invoice_draft.sql",
    "schemas/public/tables/users.sql",
  ];

  test("keeps the paths that contain the query, ignoring case", () => {
    expect(filterSchemaFiles(paths, " invoice ")).toEqual([
      "schemas/billing/tables/invoice.sql",
      "schemas/public/tables/Invoice_draft.sql",
    ]);
  });

  test("an empty query keeps every path", () => {
    expect(filterSchemaFiles(paths, "")).toBe(paths);
  });
});
