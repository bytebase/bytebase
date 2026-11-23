import { create, toJson } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { describe, expect, test } from "vitest";
import type { AccessGrantDraft } from "@/modules/sql-editor/model/accessGrantDraft";
import {
  AccessGrant_Status,
  AccessGrantSchema,
} from "@/types/proto-es/v1/access_grant_service_pb";
import {
  createSubmittedAccessGrant,
  decodeSubmittedAccessGrant,
  normalizeSubmittedAccessGrants,
} from "./submittedAccessGrant";

const draft: AccessGrantDraft = {
  project: "projects/example",
  sourceDatabase: "instances/example/databases/source",
  targets: ["instances/example/databases/source"],
  statement: "SELECT original FROM example",
};
const grant = () =>
  create(AccessGrantSchema, {
    name: "projects/example/accessGrants/request-1",
    creator: "users/member@example.com",
    issue: "projects/example/issues/101",
    query: "SELECT edited FROM example",
    targets: ["projects/example/instances/example/databases/edited"],
    status: AccessGrant_Status.ACTIVE,
    unmask: true,
    export: true,
    schema: "public",
    container: "example",
    expiration: {
      case: "expireTime",
      value: create(TimestampSchema, { seconds: 1800000000n, nanos: 123 }),
    },
  });

describe("submitted Access Grant receipts", () => {
  test("round trips the actual submitted scope and protobuf expiration as JSON", () => {
    const submitted = grant();
    const receipt = createSubmittedAccessGrant(submitted, 1234);

    expect(receipt).toEqual({
      grant: toJson(AccessGrantSchema, submitted),
      submittedAt: 1234,
    });
    expect(
      decodeSubmittedAccessGrant(JSON.parse(JSON.stringify(receipt)))
    ).toEqual(submitted);
  });

  test("preserves canonical database names containing spaces", () => {
    const submitted = create(AccessGrantSchema, {
      ...grant(),
      targets: ["instances/example/databases/example data"],
    });
    expect(
      decodeSubmittedAccessGrant(createSubmittedAccessGrant(submitted))
    ).toEqual(submitted);
  });

  test.each([
    { name: "projects/-/accessGrants/request-1" },
    { name: "projects/example/accessGrants/-" },
    { creator: "users/-" },
    { issue: "projects/other/issues/101" },
    { targets: ["projects/other/instances/example/databases/edited"] },
    { targets: ["instances/example/databases/-"] },
    { targets: [] },
    { query: " " },
  ])("does not link malformed or cross-project scope: %o", (patch) => {
    expect(
      createSubmittedAccessGrant(
        create(AccessGrantSchema, {
          ...grant(),
          ...patch,
        }),
        1234
      )
    ).toBeUndefined();
  });

  test("rejects corrupt JSON and invalid submission timestamps", () => {
    expect(
      decodeSubmittedAccessGrant({ grant: { name: 123 }, submittedAt: 1 })
    ).toBeUndefined();
    expect(
      decodeSubmittedAccessGrant({ grant: {}, submittedAt: NaN })
    ).toBeUndefined();
    expect(
      decodeSubmittedAccessGrant({ grant: {}, submittedAt: -1 })
    ).toBeUndefined();
  });

  test("normalizes only the owner's receipts for a matching source project", () => {
    const receipt = createSubmittedAccessGrant(grant(), 1234)!;
    const foreign = createSubmittedAccessGrant(
      create(AccessGrantSchema, {
        ...grant(),
        name: "projects/example/accessGrants/foreign",
        creator: "users/other@example.com",
      }),
      1235
    )!;

    expect(
      normalizeSubmittedAccessGrants(
        [receipt, foreign, receipt],
        "member@example.com",
        draft
      )
    ).toEqual([receipt]);
    expect(
      normalizeSubmittedAccessGrants([receipt], "member@example.com", {
        ...draft,
        project: "projects/other",
      })
    ).toBeUndefined();
    expect(
      normalizeSubmittedAccessGrants([receipt], "member@example.com", undefined)
    ).toBeUndefined();
    expect(
      normalizeSubmittedAccessGrants([receipt], "member@example.com", {
        ...draft,
        kind: "requestRole",
      })
    ).toBeUndefined();
  });
});
