import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { issueServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import { useAppStore } from "@/stores/app";
import { buildIssueFilter } from "@/stores/app/issue";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import {
  type Issue,
  SearchIssuesRequestSchema,
} from "@/types/proto-es/v1/issue_service_pb";

const PAGE_SIZE = 10;

type ProjectOpenIssue = {
  name: string;
  title: string;
  relation: "review" | "created";
  approvalStatus: string;
};

function isProjectName(value: string | undefined): value is string {
  return !!value && /^projects\/[^/]+$/.test(value);
}

function summarizeIssue(
  issue: Issue,
  relation: ProjectOpenIssue["relation"]
): ProjectOpenIssue {
  return {
    name: issue.name,
    title: issue.title,
    relation,
    approvalStatus: ApprovalStatus[issue.approvalStatus],
  };
}

export function createProjectOpenIssuesTool({
  projectName,
}: {
  projectName?: string;
}) {
  return async (): Promise<string> => {
    if (!isProjectName(projectName)) {
      return JSON.stringify({
        error: "This starter is not bound to a Project.",
      });
    }

    const { currentUser } = useAppStore.getState();
    if (!currentUser?.name) {
      return JSON.stringify({ error: "The current user is unavailable." });
    }

    const options = {
      contextValues: createContextValues().set(silentContextKey, true),
    };
    const results = await Promise.allSettled([
      issueServiceClientConnect.searchIssues(
        create(SearchIssuesRequestSchema, {
          parent: projectName,
          filter: buildIssueFilter({
            project: projectName,
            query: "",
            currentApprover: currentUser.name,
            statusList: [IssueStatus.OPEN],
            approvalStatus: ApprovalStatus.PENDING,
          }),
          orderBy: "update_time desc",
          pageSize: PAGE_SIZE,
        }),
        options
      ),
      issueServiceClientConnect.searchIssues(
        create(SearchIssuesRequestSchema, {
          parent: projectName,
          filter: buildIssueFilter({
            project: projectName,
            query: "",
            creator: currentUser.name,
            statusList: [IssueStatus.OPEN],
          }),
          orderBy: "update_time desc",
          pageSize: PAGE_SIZE,
        }),
        options
      ),
    ]);

    const fulfilledCount = results.filter(
      (result) => result.status === "fulfilled"
    ).length;
    if (!fulfilledCount) {
      return JSON.stringify({
        error: "Unable to load open issues for this Project.",
      });
    }

    const issues = new Map<string, ProjectOpenIssue>();
    const pendingReviews =
      results[0].status === "fulfilled" ? results[0].value.issues : [];
    const createdIssues =
      results[1].status === "fulfilled" ? results[1].value.issues : [];
    for (const issue of pendingReviews) {
      issues.set(issue.name, summarizeIssue(issue, "review"));
    }
    for (const issue of createdIssues) {
      if (!issues.has(issue.name)) {
        issues.set(issue.name, summarizeIssue(issue, "created"));
      }
    }

    return JSON.stringify({
      project: projectName,
      issues: [...issues.values()],
      partial:
        fulfilledCount !== results.length ||
        results.some(
          (result) =>
            result.status === "fulfilled" && Boolean(result.value.nextPageToken)
        ),
    });
  };
}
