import { computed, Ref } from "vue";
import { useRouter } from "vue-router";
import { useIssueStore, useProjectV1Store } from "@/store";
import { Issue, IssueCreate, Task, TaskStatus } from "@/types";
import { issueSlug } from "@/utils";
import { maybeCreateBackTraceComments } from "../rollback/common";

export const useGrantRequestIssueLogic = (params: {
  create: Ref<boolean>;
  issue: Ref<Issue | IssueCreate>;
}) => {
  const { create, issue } = params;
  const router = useRouter();
  const issueStore = useIssueStore();
  const projectV1Store = useProjectV1Store();

  const project = computed(() => {
    const projectUID = create.value
      ? (issue.value as IssueCreate).projectId
      : (issue.value as Issue).project.id;
    return projectV1Store.getProjectByUID(String(projectUID));
  });

  const createIssue = async (issue: IssueCreate) => {
    const createdIssue = await issueStore.createIssue(issue);
    await maybeCreateBackTraceComments(createdIssue);

    // Use replace to omit the new issue url in the navigation history.
    router.replace(`/issue/${issueSlug(createdIssue.name, createdIssue.id)}`);
  };

  const allowApplyIssueStatusTransition = () => {
    // no extra logic by default
    return true;
  };

  const allowApplyTaskStatusTransition = (task: Task, to: TaskStatus) => {
    if (to === "CANCELED") {
      // All task types are not CANCELable by default.
      // Might be overwritten by other issue logic providers.
      return false;
    }

    // no extra logic by default
    return true;
  };

  return {
    project,
    createIssue,
    allowApplyIssueStatusTransition,
    allowApplyTaskStatusTransition,
  };
};
