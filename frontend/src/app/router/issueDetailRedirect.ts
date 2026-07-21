import { create } from "@bufbuild/protobuf";
import { type LoaderFunctionArgs, redirect } from "react-router";
import { issueServiceClientConnect, planServiceClientConnect } from "@/api";
import { shouldStayOnPlanDetailPage } from "@/lib/plan/workflow";
import { issueNamePrefix, projectNamePrefix } from "@/stores/modules/v1/common";
import {
  GetIssueRequestSchema,
  Issue_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import { GetPlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { extractPlanUID } from "@/utils/v1/issue/plan";

// Plan Detail is the canonical review surface for Draft Review Issues and
// schema/data change plans. Drafts redirect directly from their linked plan;
// submitted create-database, export, and grant issues stay on Issue Detail.
//
// For submitted DATABASE_CHANGE issues, the plan specs distinguish schema
// changes from create-database plans. Drafts do not need that fetch: their
// lifecycle and metadata always belong on the linked Plan Detail surface.
//
// Every issue-detail entry point navigates to this route by name, so this single
// guard covers deep links, inbox/notifications, and review CTAs without patching
// call sites. Old issue-detail URLs resolve via this redirect rather than 404.
export async function issueDetailRedirectLoader({
  params,
  request,
}: LoaderFunctionArgs): Promise<Response | null> {
  const { projectId, issueId } = params;
  if (!projectId || !issueId || issueId.toLowerCase() === "create") {
    return null;
  }
  try {
    const issue = await issueServiceClientConnect.getIssue(
      create(GetIssueRequestSchema, {
        name: `${projectNamePrefix}${projectId}/${issueNamePrefix}${issueId}`,
      })
    );
    if (!issue.plan) {
      return null;
    }
    // A Draft Review Issue is edited and submitted from Plan Detail regardless
    // of plan spec type. Its plan name is already sufficient, so avoid an
    // unnecessary GetPlan request.
    if (issue.draft) {
      const planId = extractPlanUID(issue.plan);
      if (!planId) return null;
      const { search } = new URL(request.url);
      return redirect(`/projects/${projectId}/plans/${planId}${search}`);
    }
    // Submitted export and grant issues stay on Issue Detail; skip the plan
    // fetch for them. DATABASE_CHANGE remains ambiguous until the plan loads.
    if (issue.type !== Issue_Type.DATABASE_CHANGE) {
      return null;
    }
    // DATABASE_CHANGE is ambiguous (schema-change vs create-database); the plan's
    // specs decide. Reuse Plan Detail's own predicate so they can't loop.
    const plan = await planServiceClientConnect.getPlan(
      create(GetPlanRequestSchema, { name: issue.plan })
    );
    if (!shouldStayOnPlanDetailPage(plan)) {
      // create-database (and any non-change plan) → Issue Detail.
      return null;
    }
    const planId = extractPlanUID(issue.plan);
    if (!planId) {
      return null;
    }
    const { search } = new URL(request.url);
    return redirect(`/projects/${projectId}/plans/${planId}${search}`);
  } catch {
    // 404/403/network → fall through to Issue Detail, which has its own
    // not-found / permission-denied handling. Avoids turning a transient error
    // into a broken redirect.
    return null;
  }
}
