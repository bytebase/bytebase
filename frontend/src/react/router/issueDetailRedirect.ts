import { create } from "@bufbuild/protobuf";
import { type LoaderFunctionArgs, redirect } from "react-router-dom";
import { issueServiceClientConnect, planServiceClientConnect } from "@/connect";
import { shouldStayOnPlanDetailPage } from "@/react/pages/project/plan-detail/utils/header";
import { issueNamePrefix, projectNamePrefix } from "@/store/modules/v1/common";
import {
  GetIssueRequestSchema,
  Issue_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import { GetPlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { extractPlanUID } from "@/utils/v1/issue/plan";

// BYT-9721: Plan Detail is the canonical review surface for schema/data change
// plans (3.20.0). This route loader redirects the issue-detail route to Plan
// Detail for those issues, preserving the query string. Create-database, export,
// and grant issues stay on Issue Detail.
//
// The discriminator is the plan's specs, not the issue type: schema-change and
// create-database share the DATABASE_CHANGE proto type, so we fetch the plan and
// redirect iff `shouldStayOnPlanDetailPage` — the same predicate Plan Detail's
// own `useRedirects` uses to decide staying. Sharing it guarantees the two can't
// ping-pong (issue -> plan -> issue -> ...).
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
    // Export and grant issues stay on Issue Detail; skip the plan fetch for them.
    // (DATABASE_CHANGE covers both schema-change and create-database.)
    if (issue.type !== Issue_Type.DATABASE_CHANGE || !issue.plan) {
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
