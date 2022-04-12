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
      :issue="issue"
      :task="currentTask"
      :transition="updateStatusModalState.transition"
      :output-field-list="issueTemplate.outputFieldList"
      @submit="
        (comment) => {
          updateStatusModalState.show = false;
          if (updateStatusModalState.mode == 'ISSUE') {
            doIssueStatusTransition(updateStatusModalState.transition, comment);
          } else if (updateStatusModalState.mode == 'TASK') {
            doTaskStatusTransition(
              updateStatusModalState.transition,
              updateStatusModalState.payload,
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

<script lang="ts">
import { PropType, computed, reactive, defineComponent, ref } from "vue";
import { Store, useStore } from "vuex";
import StatusTransitionForm from "./StatusTransitionForm.vue";
import {
  activeTask,
  allTaskList,
  applicableTaskTransition,
  isDBAOrOwner,
  TaskStatusTransition,
} from "../../utils";
import {
  ASSIGNEE_APPLICABLE_ACTION_LIST,
  CREATOR_APPLICABLE_ACTION_LIST,
  Issue,
  IssueCreate,
  IssueStatusTransition,
  IssueStatusTransitionType,
  ISSUE_STATUS_TRANSITION_LIST,
  ONBOARDING_ISSUE_ID,
  Principal,
  SYSTEM_BOT_ID,
  Task,
  UNKNOWN_ID,
} from "../../types";
import { IssueTemplate } from "../../plugins";
import isEmpty from "lodash-es/isEmpty";
import { useI18n } from "vue-i18n";
import { BBContextMenu } from "../../bbkit";
import { useCurrentUser } from "@/store";

interface UpdateStatusModalState {
  mode: "ISSUE" | "TASK";
  show: boolean;
  style: string;
  okText: string;
  title: string;
  transition?: IssueStatusTransition | TaskStatusTransition;
  payload?: Task;
}

export type IssueContext = {
  store: Store<any>;
  currentUser: Principal;
  create: boolean;
  issue: Issue | IssueCreate;
};

export default defineComponent({
  name: "IssueStatusTransitionButtonGroup",
  components: {
    StatusTransitionForm,
  },
  props: {
    create: {
      required: true,
      type: Boolean,
    },
    issue: {
      required: true,
      type: Object as PropType<Issue | IssueCreate>,
    },
    issueTemplate: {
      required: true,
      type: Object as PropType<IssueTemplate>,
    },
  },
  emits: ["create", "change-issue-status", "change-task-status"],

  setup(props, { emit }) {
    const { t } = useI18n();
    const store = useStore();
    const menu = ref<InstanceType<typeof BBContextMenu>>();

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
        store,
        currentUser: currentUser.value,
        create: props.create,
        issue: props.issue,
      };
    });

    const applicableTaskStatusTransitionList = computed(
      (): TaskStatusTransition[] => {
        if ((props.issue as Issue).id == ONBOARDING_ISSUE_ID) {
          return [];
        }
        switch ((props.issue as Issue).status) {
          case "DONE":
          case "CANCELED":
            return [];
          case "OPEN": {
            let list: TaskStatusTransition[] = [];

            // Allow assignee, or assignee is the system bot and current user is DBA or owner
            if (
              currentUser.value.id === (props.issue as Issue).assignee?.id ||
              ((props.issue as Issue).assignee?.id == SYSTEM_BOT_ID &&
                isDBAOrOwner(currentUser.value.role))
            ) {
              list = applicableTaskTransition((props.issue as Issue).pipeline);
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
          updateStatusModalState.title = `${t("common.approve")} '${
            task.name
          }'?`;
          break;
        case "RETRY":
          updateStatusModalState.style = "INFO";
          updateStatusModalState.title = `${t("common.retry")} '${task.name}'?`;
          break;
        case "CANCEL":
          updateStatusModalState.style = "INFO";
          updateStatusModalState.title = `${t("common.cancel")} '${
            task.name
          }'?`;
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
        if ((props.issue as Issue).id == ONBOARDING_ISSUE_ID) {
          return [];
        }
        const list: IssueStatusTransitionType[] = [];
        // Allow assignee, or assignee is the system bot and current user is DBA or owner
        if (
          currentUser.value.id === (props.issue as Issue).assignee?.id ||
          ((props.issue as Issue).assignee?.id == SYSTEM_BOT_ID &&
            isDBAOrOwner(currentUser.value.role))
        ) {
          list.push(
            ...ASSIGNEE_APPLICABLE_ACTION_LIST.get(
              (props.issue as Issue).status
            )!
          );
        }
        if (currentUser.value.id === (props.issue as Issue).creator.id) {
          CREATOR_APPLICABLE_ACTION_LIST.get(
            (props.issue as Issue).status
          )!.forEach((item) => {
            if (list.indexOf(item) == -1) {
              list.push(item);
            }
          });
        }

        return list
          .filter((item) => {
            const pipeline = (props.issue as Issue).pipeline;
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
      return activeTask((props.issue as Issue).pipeline);
    });

    const allowIssueStatusTransition = (
      transition: IssueStatusTransition
    ): boolean => {
      if (transition.type == "RESOLVE") {
        // Returns false if any of the required output fields is not provided.
        for (let i = 0; i < props.issueTemplate.outputFieldList.length; i++) {
          const field = props.issueTemplate.outputFieldList[i];
          if (!field.resolved(issueContext.value)) {
            return false;
          }
        }
        return true;
      }
      return true;
    };

    const tryStartIssueStatusTransition = (
      transition: IssueStatusTransition
    ) => {
      updateStatusModalState.mode = "ISSUE";
      updateStatusModalState.okText = t(transition.buttonName);
      switch (transition.type) {
        case "RESOLVE":
          updateStatusModalState.style = "SUCCESS";
          updateStatusModalState.title = t(
            "issue.status-transition.modal.resolve"
          );
          break;
        case "CANCEL":
          updateStatusModalState.style = "INFO";
          updateStatusModalState.title = t(
            "issue.status-transition.modal.cancel"
          );
          break;
        case "REOPEN":
          updateStatusModalState.style = "INFO";
          updateStatusModalState.title = t(
            "issue.status-transition.modal.reopen"
          );
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
      const newIssue = props.issue as IssueCreate;
      if (isEmpty(newIssue.name)) {
        return false;
      }

      if (newIssue.assigneeId == UNKNOWN_ID) {
        return false;
      }

      if (
        newIssue.type == "bb.issue.database.create" ||
        newIssue.type == "bb.issue.database.schema.update" ||
        newIssue.type == "bb.issue.database.data.update"
      ) {
        for (const stage of newIssue.pipeline.stageList) {
          for (const task of stage.taskList) {
            if (
              task.type == "bb.task.database.create" ||
              task.type == "bb.task.database.schema.update" ||
              task.type == "bb.task.database.data.update"
            ) {
              if (isEmpty(task.statement)) {
                return false;
              }
            }
          }
        }
      }

      for (const field of props.issueTemplate.inputFieldList) {
        if (
          field.type != "Boolean" && // Switch is boolean value which always is present
          !field.resolved(issueContext.value)
        ) {
          return false;
        }
      }
      return true;
    });

    const doCreate = () => {
      emit("create");
    };

    return {
      menu,
      updateStatusModalState,
      applicableTaskStatusTransitionList,
      tryStartTaskStatusTransition,
      doTaskStatusTransition,
      applicableIssueStatusTransitionList,
      currentTask,
      allowIssueStatusTransition,
      tryStartIssueStatusTransition,
      doIssueStatusTransition,
      allowCreate,
      doCreate,
    };
  },
});
</script>
