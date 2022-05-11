<template>
  <template v-if="create">
    <button
      type="button"
      class="btn-primary px-4 py-2"
      :disabled="!allowCreate"
      @click.prevent="doCreate"
    >
      {{ $t("common.create") }}
    </button>
  </template>
  <template v-else>
    <div
      v-if="applicableTaskStatusTransitionList.length > 0"
      class="flex space-x-2"
    >
      <template
        v-for="(transition, index) in applicableTaskStatusTransitionList"
        :key="index"
      >
        <button
          type="button"
          :class="transition.buttonClass"
          @click.prevent="tryStartTaskStatusTransition(transition)"
        >
          {{ $t(transition.buttonName) }}
        </button>
      </template>
      <template v-if="applicableIssueStatusTransitionList.length > 0">
        <button
          id="user-menu"
          type="button"
          class="text-control-light"
          aria-label="User menu"
          aria-haspopup="true"
          @click.prevent="menu?.toggle($event, {})"
          @contextmenu.capture.prevent="menu?.toggle($event, {})"
        >
          <heroicons-solid:dots-vertical class="w-6 h-6" />
        </button>
        <BBContextMenu ref="menu" class="origin-top-right mt-10 w-42">
          <template
            v-for="(transition, index) in applicableIssueStatusTransitionList"
            :key="index"
          >
            <div
              class="menu-item"
              role="menuitem"
              @click.prevent="tryStartIssueStatusTransition(transition)"
            >
              {{ $t(transition.buttonName) }}
            </div>
          </template>
        </BBContextMenu>
      </template>
    </div>
    <template v-else>
      <div
        if="applicableIssueStatusTransitionList.length > 0"
        class="flex space-x-2"
      >
        <template
          v-for="(transition, index) in applicableIssueStatusTransitionList"
          :key="index"
        >
          <button
            type="button"
            :class="transition.buttonClass"
            :disabled="!allowIssueStatusTransition(transition)"
            @click.prevent="tryStartIssueStatusTransition(transition)"
          >
            {{ $t(transition.buttonName) }}
          </button>
        </template>
      </div>
    </template>
  </template>
  <BBModal
    v-if="updateStatusModalState.show"
    :title="updateStatusModalState.title"
    @close="updateStatusModalState.show = false"
  >
    <StatusTransitionForm
      :mode="updateStatusModalState.mode"
      :ok-text="updateStatusModalState.okText"
      :issue="(issue as Issue)"
      :task="currentTask"
      :transition="updateStatusModalState.transition!"
      :output-field-list="issueTemplate.outputFieldList"
      @submit="
        (comment) => {
          updateStatusModalState.show = false;
          if (updateStatusModalState.mode == 'ISSUE') {
            doIssueStatusTransition(updateStatusModalState.transition as IssueStatusTransition, comment);
          } else if (updateStatusModalState.mode == 'TASK') {
            doTaskStatusTransition(
              updateStatusModalState.transition as TaskStatusTransition,
              updateStatusModalState.payload!,
              comment
            );
          }
        }
      "
      @cancel="
        () => {
          updateStatusModalState.show = false;
        }
      "
    />
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, reactive, ref } from "vue";
import { isEmpty } from "lodash-es";
import { useI18n } from "vue-i18n";
import type { TaskStatusTransition } from "@/utils";
import { allTaskList, applicableTaskTransition, isDBAOrOwner } from "@/utils";
import type {
  Issue,
  IssueCreate,
  IssueStatusTransition,
  IssueStatusTransitionType,
  Principal,
  Task,
  TaskCreate,
} from "@/types";
import {
  ASSIGNEE_APPLICABLE_ACTION_LIST,
  CREATOR_APPLICABLE_ACTION_LIST,
  SYSTEM_BOT_ID,
  UNKNOWN_ID,
  ISSUE_STATUS_TRANSITION_LIST,
  ONBOARDING_ISSUE_ID,
} from "@/types";
import { BBContextMenu } from "@/bbkit";
import { useCurrentUser } from "@/store";
import StatusTransitionForm from "./StatusTransitionForm.vue";
import {
  flattenTaskList,
  IssueTypeWithStatement,
  TaskTypeWithStatement,
  useIssueLogic,
} from "./logic";

export type IssueContext = {
  currentUser: Principal;
  create: boolean;
  issue: Issue | IssueCreate;
};

interface UpdateStatusModalState {
  mode: "ISSUE" | "TASK";
  show: boolean;
  style: string;
  okText: string;
  title: string;
  transition?: IssueStatusTransition | TaskStatusTransition;
  payload?: Task;
}

const emit = defineEmits(["change-issue-status", "change-task-status"]);

const { t } = useI18n();
const menu = ref<InstanceType<typeof BBContextMenu>>();

const {
  create,
  issue,
  template: issueTemplate,
  activeTaskOfPipeline,
  doCreate,
} = useIssueLogic();

const updateStatusModalState = reactive<UpdateStatusModalState>({
  mode: "ISSUE",
  show: false,
  style: "INFO",
  okText: "OK",
  title: "",
});

const currentUser = useCurrentUser();

const issueContext = computed((): IssueContext => {
  return {
    currentUser: currentUser.value,
    create: create.value,
    issue: issue.value,
  };
});

const applicableTaskStatusTransitionList = computed(
  (): TaskStatusTransition[] => {
    const issueEntity = issue.value as Issue;
    if (issueEntity.id == ONBOARDING_ISSUE_ID) {
      return [];
    }
    switch (issueEntity.status) {
      case "DONE":
      case "CANCELED":
        return [];
      case "OPEN": {
        let list: TaskStatusTransition[] = [];

        // Allow assignee, or assignee is the system bot and current user is DBA or owner
        if (
          currentUser.value.id === issueEntity.assignee?.id ||
          (issueEntity.assignee?.id == SYSTEM_BOT_ID &&
            isDBAOrOwner(currentUser.value.role))
        ) {
          list = applicableTaskTransition(issueEntity.pipeline);
        }

        return list;
      }
    }
    return []; // Only to make eslint happy. Should never reach this line.
  }
);

const tryStartTaskStatusTransition = (transition: TaskStatusTransition) => {
  updateStatusModalState.mode = "TASK";
  updateStatusModalState.okText = t(transition.buttonName);
  const task = currentTask.value;
  switch (transition.type) {
    case "RUN":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = `${t("common.run")} '${task.name}'?`;
      break;
    case "APPROVE":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = `${t("common.approve")} '${task.name}'?`;
      break;
    case "RETRY":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = `${t("common.retry")} '${task.name}'?`;
      break;
    case "CANCEL":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = `${t("common.cancel")} '${task.name}'?`;
      break;
    case "SKIP":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = `${t("common.skip")} '${task.name}'?`;
      break;
  }
  updateStatusModalState.transition = transition;
  updateStatusModalState.payload = task;
  updateStatusModalState.show = true;
};

const doTaskStatusTransition = (
  transition: TaskStatusTransition,
  task: Task,
  comment: string
) => {
  emit("change-task-status", task, transition.to, comment);
};

const applicableIssueStatusTransitionList = computed(
  (): IssueStatusTransition[] => {
    const issueEntity = issue.value as Issue;
    if (issueEntity.id == ONBOARDING_ISSUE_ID) {
      return [];
    }
    const list: IssueStatusTransitionType[] = [];
    // Allow assignee, or assignee is the system bot and current user is DBA or owner
    if (
      currentUser.value.id === issueEntity.assignee?.id ||
      (issueEntity.assignee?.id == SYSTEM_BOT_ID &&
        isDBAOrOwner(currentUser.value.role))
    ) {
      list.push(...ASSIGNEE_APPLICABLE_ACTION_LIST.get(issueEntity.status)!);
    }
    if (currentUser.value.id === issueEntity.creator.id) {
      CREATOR_APPLICABLE_ACTION_LIST.get(issueEntity.status)!.forEach(
        (item) => {
          if (list.indexOf(item) == -1) {
            list.push(item);
          }
        }
      );
    }

    return list
      .filter((item) => {
        const pipeline = issueEntity.pipeline;
        // Disallow any issue status transition if the active task is in RUNNING state.
        if (currentTask.value.status == "RUNNING") {
          return false;
        }

        const taskList = allTaskList(pipeline);
        // Don't display the Resolve action if the last task is NOT in DONE status.
        if (
          item == "RESOLVE" &&
          taskList.length > 0 &&
          (currentTask.value.id != taskList[taskList.length - 1].id ||
            currentTask.value.status != "DONE")
        ) {
          return false;
        }

        return true;
      })
      .map(
        (type: IssueStatusTransitionType) =>
          ISSUE_STATUS_TRANSITION_LIST.get(type)!
      )
      .reverse();
  }
);

const currentTask = computed(() => {
  return activeTaskOfPipeline((issue.value as Issue).pipeline);
});

const allowIssueStatusTransition = (
  transition: IssueStatusTransition
): boolean => {
  if (transition.type == "RESOLVE") {
    const template = issueTemplate.value;
    // Returns false if any of the required output fields is not provided.
    for (let i = 0; i < template.outputFieldList.length; i++) {
      const field = template.outputFieldList[i];
      if (!field.resolved(issueContext.value)) {
        return false;
      }
    }
    return true;
  }
  return true;
};

const tryStartIssueStatusTransition = (transition: IssueStatusTransition) => {
  updateStatusModalState.mode = "ISSUE";
  updateStatusModalState.okText = t(transition.buttonName);
  switch (transition.type) {
    case "RESOLVE":
      updateStatusModalState.style = "SUCCESS";
      updateStatusModalState.title = t("issue.status-transition.modal.resolve");
      break;
    case "CANCEL":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = t("issue.status-transition.modal.cancel");
      break;
    case "REOPEN":
      updateStatusModalState.style = "INFO";
      updateStatusModalState.title = t("issue.status-transition.modal.reopen");
      break;
  }
  updateStatusModalState.transition = transition;
  updateStatusModalState.show = true;
};

const doIssueStatusTransition = (
  transition: IssueStatusTransition,
  comment: string
) => {
  emit("change-issue-status", transition.to, comment);
};

const allowCreate = computed(() => {
  const newIssue = issue.value as IssueCreate;

  if (isEmpty(newIssue.name)) {
    return false;
  }

  if (newIssue.assigneeId == UNKNOWN_ID) {
    return false;
  }

  if (IssueTypeWithStatement.includes(newIssue.type)) {
    const allTaskList = flattenTaskList<TaskCreate>(newIssue);
    for (const task of allTaskList) {
      if (TaskTypeWithStatement.includes(task.type)) {
        if (isEmpty(task.statement)) {
          return false;
        }
      }
    }
  }

  const template = issueTemplate.value;
  for (const field of template.inputFieldList) {
    if (
      field.type !== "Boolean" && // Switch is boolean value which always is present
      !field.resolved(issueContext.value)
    ) {
      return false;
    }
  }
  return true;
});
</script>
