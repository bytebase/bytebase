import { computed, Ref } from "vue";
import { Issue, IssueCreate, Project, Task, TaskStatus } from "@/types";
import { useRouter } from "vue-router";
import { useIssueStore, useProjectStore } from "@/store";
import { issueSlug } from "@/utils";
import { maybeCreateBackTraceComments } from "../rollback/common";

export const useGrantRequestIssueLogic = (params: {
  create: Ref<boolean>;
  issue: Ref<Issue | IssueCreate>;
}) => {
  const { create, issue } = params;
  const router = useRouter();
  const issueStore = useIssueStore();
  const projectStore = useProjectStore();

  const project = computed((): Project => {
    if (create.value) {
      return projectStore.getProjectById(
        (issue.value as IssueCreate).projectId
      );
    }
    return (issue.value as Issue).project;
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
