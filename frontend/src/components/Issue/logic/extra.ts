import { cloneDeep, isEqual } from "lodash-es";
import { computed, nextTick } from "vue";
import type { InputField, OutputField } from "@/plugins";
import {
  useCurrentUserV1,
  useIssueStore,
  useIssueSubscriberStore,
  useTaskStore,
} from "@/store";
import type {
  Issue,
  IssueCreate,
  IssueStatus,
  IssueStatusPatch,
  PrincipalId,
  Stage,
  StageAllTaskStatusPatch,
  Task,
  TaskCreate,
  TaskPatch,
  TaskStatus,
  TaskStatusPatch,
} from "@/types";
import { extractUserUID, hasWorkspacePermissionV1 } from "@/utils";
import { useIssueLogic } from "./index";

export const useExtraIssueLogic = () => {
  const {
    create,
    issue,
    isGhostMode,
    selectedStage,
    selectedTask,
    selectStageOrTask,
    selectTask,
    onStatusChanged,
    patchIssue,
    patchTask,
  } = useIssueLogic();
  const issueStore = useIssueStore();
  const issueSubscriberStore = useIssueSubscriberStore();
  const taskStore = useTaskStore();
  const currentUserV1 = useCurrentUserV1();
  const currentUserUID = computed(() =>
    extractUserUID(currentUserV1.value.name)
  );

  const allowEditOutput = computed(() => {
    if (create.value) {
      return true;
    }

    const issueEntity = issue.value as Issue;
    return (
      issueEntity.status === "OPEN" &&
      String(issueEntity.assignee?.id) === currentUserUID.value
    );
  });

  const allowEditNameAndDescription = computed(() => {
    if (create.value) {
      return true;
    }

    const issueEntity = issue.value as Issue;
    if (issueEntity.status === "OPEN") {
      if (
        String(issueEntity.assignee.id) === currentUserUID.value ||
        String(issueEntity.creator.id) === currentUserUID.value
      ) {
        // Allowed if current user is the assignee or creator.
        return true;
      }

      if (
        hasWorkspacePermissionV1(
          "bb.permission.workspace.manage-issue",
          currentUserV1.value.userRole
        )
      ) {
        // Allowed if RBAC is enabled and current is DBA or workspace owner.
        return true;
      }
    }

    return false;
  });

  const updateName = (
    newName: string,
    postUpdated?: (updatedIssue: Issue) => void
  ) => {
    if (create.value) {
      issue.value.name = newName;
    } else {
      patchIssue(
        {
          name: newName,
        },
        postUpdated
      );
    }
  };

  const updateDescription = (
    newDescription: string,
    postUpdated?: (updatedIssue: Issue) => void
  ) => {
    if (create.value) {
      issue.value.description = newDescription;
    } else {
      patchIssue(
        {
          description: newDescription,
        },
        postUpdated
      );
    }
  };

  const updateAssigneeId = (newAssigneeId: string) => {
    if (create.value) {
      (issue.value as IssueCreate).assigneeId = parseInt(newAssigneeId, 10);
    } else {
      patchIssue({
        assigneeId: parseInt(newAssigneeId, 10),
      });
    }
  };

  const updateEarliestAllowedTime = (newEarliestAllowedTsMs: number) => {
    if (create.value) {
      if (isGhostMode.value) {
        // In gh-ost mode, when creating an issue, all sub-tasks in a stage
        // share the same earliestAllowedTs.
        // So updates on any one of them will be applied to others.
        // (They can be updated independently after creation)
        const taskList = selectedStage.value.taskList as TaskCreate[];
        taskList.forEach((task) => {
          task.earliestAllowedTs = newEarliestAllowedTsMs;
        });
      } else {
        selectedTask.value.earliestAllowedTs = newEarliestAllowedTsMs;
      }
    } else {
      const task = selectedTask.value as Task;
      const taskPatch: TaskPatch = {
        earliestAllowedTs: newEarliestAllowedTsMs,
        updatedTs: task.updatedTs,
      };
      patchTask(task.id, taskPatch);
    }
  };

  const addSubscriberId = (subscriberId: PrincipalId) => {
    issueSubscriberStore.createSubscriber({
      issueId: (issue.value as Issue).id,
      subscriberId,
    });
  };

  const removeSubscriberId = (subscriberId: PrincipalId) => {
    issueSubscriberStore.deleteSubscriber({
      issueId: (issue.value as Issue).id,
      subscriberId,
    });
  };

  const updateCustomField = (field: InputField | OutputField, value: any) => {
    const payload = issue.value.payload as Record<string, any>;
    if (!isEqual(payload[field.id], value)) {
      if (create.value) {
        payload[field.id] = value;
      } else {
        const newPayload = cloneDeep(payload);
        newPayload[field.id] = value;
        patchIssue({
          payload: newPayload,
        });
      }
    }
  };

  const changeIssueStatus = (newStatus: IssueStatus, comment: string) => {
    const issueStatusPatch: IssueStatusPatch = {
      status: newStatus,
      comment: comment,
    };
    issueStore.updateIssueStatus({
      issueId: (issue.value as Issue).id,
      issueStatusPatch,
    });
  };

  const changeStageAllTaskStatus = (
    stage: Stage,
    newStatus: TaskStatus,
    comment: string
  ) => {
    // Switch to the last task in this stage
    const lastTask = stage.taskList[stage.taskList.length - 1];
    selectStageOrTask(Number(stage.id));
    nextTick(() => {
      selectTask(lastTask);
    });

    // Patch the stage
    const stageAllTaskStatusPatch: StageAllTaskStatusPatch = {
      id: stage.id,
      status: newStatus,
      comment,
      updatedTs: Math.floor(Date.now() / 1000),
    };
    taskStore
      .updateStageAllTaskStatus({
        issue: issue.value as Issue,
        stage,
        patch: stageAllTaskStatusPatch,
      })
      .then(() => {
        onStatusChanged(true);
      });
  };

  const changeTaskStatus = (
    task: Task,
    newStatus: TaskStatus,
    comment: string
  ) => {
    // Switch to the stage view containing this task
    selectStageOrTask(Number(task.stage.id));
    nextTick().then(() => {
      selectTask(task);
    });

    const taskStatusPatch: TaskStatusPatch = {
      status: newStatus,
      comment: comment,
      updatedTs: task.updatedTs,
    };
    taskStore
      .updateStatus({
        issueId: (issue.value as Issue).id,
        pipelineId: (issue.value as Issue).pipeline!.id,
        taskId: task.id,
        taskStatusPatch,
      })
      .then(() => {
        onStatusChanged(true);
      });
  };

  const runTaskChecks = (task: Task) => {
    taskStore
      .runChecks({
        issueId: (issue.value as Issue).id,
        pipelineId: (issue.value as Issue).pipeline!.id,
        taskId: task.id,
      })
      .then(() => {
        onStatusChanged(true);
      });
  };

  return {
    allowEditOutput,
    allowEditNameAndDescription,
    updateName,
    updateDescription,
    updateAssigneeId,
    updateEarliestAllowedTime,
    addSubscriberId,
    removeSubscriberId,
    updateCustomField,
    changeIssueStatus,
    changeStageAllTaskStatus,
    changeTaskStatus,
    runTaskChecks,
  };
};
