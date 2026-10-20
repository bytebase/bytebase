import { useAppStore } from "@/stores/app";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";

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

    const { currentUser, listIssues } = useAppStore.getState();
    if (!currentUser?.name) {
      return JSON.stringify({ error: "The current user is unavailable." });
    }

    const results = await Promise.allSettled([
      listIssues({
        find: {
          project: projectName,
          query: "",
          currentApprover: currentUser.name,
          statusList: [IssueStatus.OPEN],
          approvalStatus: ApprovalStatus.PENDING,
          orderBy: "update_time desc",
        },
        pageSize: PAGE_SIZE,
      }),
      listIssues({
        find: {
          project: projectName,
          query: "",
          creator: currentUser.name,
          statusList: [IssueStatus.OPEN],
          orderBy: "update_time desc",
        },
        pageSize: PAGE_SIZE,
      }),
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
      partial: fulfilledCount !== results.length,
    });
  };
}
