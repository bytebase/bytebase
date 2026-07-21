import { create } from "@bufbuild/protobuf";
import { createPlanWithDraftReview } from "@/lib/plan/workflow";
import type {
  CreateIssueRequest,
  Issue,
} from "@/types/proto-es/v1/issue_service_pb";
import type {
  CreatePlanRequest,
  Plan,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";

export interface RollbackDraftPreview {
  statement: string;
  target: string;
}

export async function createRollbackDraftReview({
  createIssue,
  createPlan,
  createSheet,
  creator,
  newId,
  parent,
  previews,
  title,
  description,
}: {
  createIssue: (request: CreateIssueRequest) => Promise<Issue>;
  createPlan: (request: CreatePlanRequest) => Promise<Plan>;
  createSheet: (sheet: Sheet) => Promise<Sheet>;
  creator: string;
  newId: () => string;
  parent: string;
  previews: RollbackDraftPreview[];
  title: string;
  description: string;
}): Promise<{ issue: Issue; plan: Plan }> {
  const specs = [];
  for (const preview of previews) {
    const sheet = await createSheet(
      create(SheetSchema, {
        name: `${parent}/sheets/${newId()}`,
        content: new TextEncoder().encode(preview.statement),
      })
    );
    specs.push(
      create(Plan_SpecSchema, {
        id: newId(),
        config: {
          case: "changeDatabaseConfig",
          value: create(Plan_ChangeDatabaseConfigSchema, {
            targets: [preview.target],
            sheet: sheet.name,
          }),
        },
      })
    );
  }

  return createPlanWithDraftReview({
    createIssue,
    createPlan,
    creator,
    labels: [],
    parent,
    plan: create(PlanSchema, {
      creator,
      title,
      description,
      specs,
    }),
  });
}
