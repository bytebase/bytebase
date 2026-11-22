import { describe, expect, test } from "vitest";
import { revisionsLinkOf } from "./revisionsLink";

describe("revisionsLinkOf", () => {
  test("links a project instance's database to its Revisions tab", () => {
    expect(
      revisionsLinkOf("projects/shop/instances/prod/databases/sales", "shop")
    ).toEqual({
      href: "/projects/shop/instances/prod/databases/sales?parent=projects%2Fshop%2Finstances%2Fprod#revision",
      database: "sales",
    });
  });

  test("encodes each route segment, so a reserved character stays in its name", () => {
    expect(
      revisionsLinkOf("instances/prod db/databases/sales#archive?v=2", "shop")
        ?.href
    ).toBe(
      "/projects/shop/instances/prod%20db/databases/sales%23archive%3Fv%3D2?parent=instances%2Fprod%20db#revision"
    );
  });

  test("a target that is not a database gets no link", () => {
    expect(
      revisionsLinkOf("projects/shop/databaseGroups/all", "shop")
    ).toBeUndefined();
  });
});
