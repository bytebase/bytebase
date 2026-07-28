import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import {
  Plan_CreateDatabaseConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { shouldRedirectToIssueDetail } from "./useRedirects";

const createDatabasePlan = () =>
  create(PlanSchema, {
    specs: [
      create(Plan_SpecSchema, {
        config: {
          case: "createDatabaseConfig",
          value: create(Plan_CreateDatabaseConfigSchema),
        },
      }),
    ],
  });

describe("shouldRedirectToIssueDetail", () => {
  test("keeps a linked draft create-database issue on Plan Detail", () => {
    const issue = create(IssueSchema, {
      draft: true,
      name: "projects/p1/issues/1",
    });

    expect(shouldRedirectToIssueDetail(createDatabasePlan(), issue)).toBe(
      false
    );
  });

  test("redirects a submitted create-database issue to Issue Detail", () => {
    const issue = create(IssueSchema, {
      name: "projects/p1/issues/1",
    });

    expect(shouldRedirectToIssueDetail(createDatabasePlan(), issue)).toBe(true);
  });
});
