import { cloneDeep, isEqual } from "lodash-es";
import { InputField, OutputField } from "@/plugins";
import {
  useCurrentUser,
  useIssueStore,
  useIssueSubscriberStore,
  useTaskStore,
} from "@/store";
import {
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
import { useIssueLogic } from "./index";
import { computed, nextTick } from "vue";

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
  const currentUser = useCurrentUser();

  const allowEditOutput = computed(() => {
    if (create.value) {
      return true;
    }

    const issueEntity = issue.value as Issue;
    return (
      issueEntity.status == "OPEN" &&
      issueEntity.assignee?.id == currentUser.value.id
    );
  });

  const allowEditNameAndDescription = computed(() => {
    if (create.value) {
      return true;
    }

    const issueEntity = issue.value as Issue;
    return (
      issueEntity.status == "OPEN" &&
      (issueEntity.assignee.id == currentUser.value.id ||
        issueEntity.creator.id == currentUser.value.id)
    );
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

  const updateAssigneeId = (newAssigneeId: PrincipalId) => {
    if (create.value) {
      (issue.value as IssueCreate).assigneeId = newAssigneeId;
    } else {
      patchIssue({
        assigneeId: newAssigneeId,
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
      const taskPatch: TaskPatch = {
        earliestAllowedTs: newEarliestAllowedTsMs,
      };
      patchTask((selectedTask.value as Task).id, taskPatch);
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
    if (!isEqual(issue.value.payload[field.id], value)) {
      if (create.value) {
        issue.value.payload[field.id] = value;
      } else {
        const newPayload = cloneDeep(issue.value.payload);
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
    issueStore
      .updateIssueStatus({
        issueId: (issue.value as Issue).id,
        issueStatusPatch,
      })
      .then(() => {
        // pollIssue(POST_CHANGE_POLL_INTERVAL);
      });
  };

  const changeStageAllTaskStatus = (
    stage: Stage,
    newStatus: TaskStatus,
    comment: string
  ) => {
    // Switch to the last task in this stage
    const lastTask = stage.taskList[stage.taskList.length - 1];
    selectStageOrTask(stage.id);
    nextTick(() => {
      selectTask(lastTask);
    });

    // Patch the stage
    const stageAllTaskStatusPatch: StageAllTaskStatusPatch = {
      id: stage.id,
      status: newStatus,
      comment,
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
    selectStageOrTask(task.stage.id);
    nextTick().then(() => {
      selectTask(task);
    });

    const taskStatusPatch: TaskStatusPatch = {
      status: newStatus,
      comment: comment,
    };
    taskStore
      .updateStatus({
        issueId: (issue.value as Issue).id,
        pipelineId: (issue.value as Issue).pipeline.id,
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
        pipelineId: (issue.value as Issue).pipeline.id,
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
