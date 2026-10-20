import { create } from "@bufbuild/protobuf";
import { issueServiceClientConnect } from "@/api";
import { useAppStore } from "@/stores/app";
import { buildIssueFilter } from "@/stores/app/issue";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import { SearchIssuesRequestSchema } from "@/types/proto-es/v1/issue_service_pb";

const PAGE_SIZE = 5;

type WorkspaceAttentionIssue = {
  name: string;
  title: string;
  relation: "review" | "created";
};

function isWorkspaceName(value: string | undefined): value is string {
  return !!value && /^workspaces\/[^/]+$/.test(value);
}

export function createWorkspaceAttentionTool({
  workspaceName,
}: {
  workspaceName?: string;
}) {
  return async (): Promise<string> => {
    if (!isWorkspaceName(workspaceName)) {
      return JSON.stringify({
        error: "This starter is not bound to a workspace.",
      });
    }

    const currentUser = useAppStore.getState().currentUser;
    if (!currentUser?.name) {
      return JSON.stringify({ error: "The current user is unavailable." });
    }

    const results = await Promise.allSettled([
      issueServiceClientConnect.searchIssues(
        create(SearchIssuesRequestSchema, {
          parent: "projects/-",
          filter: buildIssueFilter({
            project: "projects/-",
            query: "",
            currentApprover: currentUser.name,
            statusList: [IssueStatus.OPEN],
            approvalStatus: ApprovalStatus.PENDING,
          }),
          orderBy: "update_time desc",
          pageSize: PAGE_SIZE,
        })
      ),
      issueServiceClientConnect.searchIssues(
        create(SearchIssuesRequestSchema, {
          parent: "projects/-",
          filter: buildIssueFilter({
            project: "projects/-",
            query: "",
            creator: currentUser.name,
            statusList: [IssueStatus.OPEN],
          }),
          orderBy: "update_time desc",
          pageSize: PAGE_SIZE,
        })
      ),
    ]);
    const fulfilled = results.filter((result) => result.status === "fulfilled");
    if (!fulfilled.length) {
      return JSON.stringify({
        error: "Unable to load items needing attention.",
      });
    }

    const issues = new Map<string, WorkspaceAttentionIssue>();
    const pendingReviews =
      results[0].status === "fulfilled" ? results[0].value.issues : [];
    const createdIssues =
      results[1].status === "fulfilled" ? results[1].value.issues : [];
    for (const issue of pendingReviews) {
      issues.set(issue.name, {
        name: issue.name,
        title: issue.title,
        relation: "review",
      });
    }
    for (const issue of createdIssues) {
      if (!issues.has(issue.name)) {
        issues.set(issue.name, {
          name: issue.name,
          title: issue.title,
          relation: "created",
        });
      }
    }

    return JSON.stringify({
      issues: [...issues.values()],
      partial:
        fulfilled.length !== results.length ||
        results.some(
          (result) =>
            result.status === "fulfilled" && Boolean(result.value.nextPageToken)
        ),
    });
  };
}
