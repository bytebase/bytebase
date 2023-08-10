<template>
  <CommonDrawer
    :show="action !== undefined"
    :title="title"
    :loading="state.loading"
    @close="$emit('close')"
  >
    <template #default>
      <div v-if="action" class="flex flex-col gap-y-4">
        <div class="text-sm">
          <label v-if="taskList.length > 1" class="textlabel">
            {{ $t("common.tasks") }}
          </label>
          <ul class="mt-1 max-h-[45vh] overflow-y-auto">
            <li
              v-for="item in distinctTaskList"
              :key="item.task.uid"
              class="text-sm textinfolabel"
            >
              <span class="textinfolabel">
                {{ item.task.title }}
              </span>
              <span v-if="item.similar.length > 0" class="ml-2 text-gray-400">
                {{
                  $t("task.n-similar-tasks", {
                    count: item.similar.length + 1,
                  })
                }}
              </span>
            </li>
          </ul>
          <PlanCheckBar
            v-if="taskList.length === 1 && action === 'ROLLOUT'"
            :allow-run-checks="false"
            :task="taskList[0]"
            class="pt-2"
          />
        </div>
        <div class="flex flex-col gap-y-1">
          <p class="textlabel">
            {{ $t("common.comment") }}
          </p>
          <NInput
            v-model:value="comment"
            type="textarea"
            :placeholder="$t('issue.leave-a-comment')"
            :autosize="{
              minRows: 3,
              maxRows: 10,
            }"
          />
        </div>
      </div>
    </template>
    <template #footer>
      <div v-if="action" class="flex justify-end gap-x-4">
        <NButton @click="$emit('close')">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          v-bind="taskRolloutActionButtonProps(action)"
          @click="handleConfirm(action, comment)"
        >
          {{ taskRolloutActionDialogButtonName(action, taskList) }}
        </NButton>
      </div>
    </template>
  </CommonDrawer>
</template>

<script setup lang="ts">
import { groupBy } from "lodash-es";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { PlanCheckBar } from "@/components/IssueV1/components/PlanCheckSection";
import {
  TaskRolloutAction,
  stageForTask,
  taskRolloutActionButtonProps,
  taskRolloutActionDialogButtonName,
  taskRolloutActionDisplayName,
  taskRunListForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { rolloutServiceClient } from "@/grpcweb";
import { pushNotification } from "@/store";
import {
  Task,
  TaskRun,
  TaskRun_Status,
} from "@/types/proto/v1/rollout_service";

type LocalState = {
  loading: boolean;
};

const props = defineProps<{
  action?: TaskRolloutAction;
  taskList: Task[];
}>();
const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({
  loading: false,
});
const { issue, events } = useIssueContext();
const comment = ref("");

const title = computed(() => {
  if (!props.action) return "";

  const action = taskRolloutActionDisplayName(props.action);
  if (props.taskList.length > 1) {
    return t("task.action-all-tasks-in-current-stage", { action });
  }
  return action;
});

const distinctTaskList = computed(() => {
  type DistinctTaskList = { task: Task; similar: Task[] };
  const groups = groupBy(props.taskList, (task) => task.title);

  return Object.keys(groups).map<DistinctTaskList>((taskName) => {
    const [task, ...similar] = groups[taskName];
    return { task, similar };
  });
});

const handleConfirm = async (
  action: TaskRolloutAction,
  comment: string | undefined
) => {
  state.loading = true;
  try {
    const stage = stageForTask(issue.value, props.taskList[0]);
    if (!stage) return;
    if (action === "ROLLOUT" || action === "RETRY" || action === "RESTART") {
      await rolloutServiceClient.batchRunTasks({
        parent: stage.name,
        tasks: props.taskList.map((task) => task.name),
      });
    } else if (action === "SKIP") {
      await rolloutServiceClient.batchSkipTasks({
        parent: stage.name,
        tasks: props.taskList.map((task) => task.name),
      });
    } else if (action === "CANCEL") {
      const taskRunListToCancel = props.taskList
        .map((task) => {
          const taskRunList = taskRunListForTask(issue.value, task);
          const currentRunningTaskRun = taskRunList.find(
            (taskRun) => taskRun.status === TaskRun_Status.RUNNING
          );
          return currentRunningTaskRun;
        })
        .filter((taskRun) => taskRun !== undefined) as TaskRun[];
      if (taskRunListToCancel.length > 0) {
        await rolloutServiceClient.batchCancelTaskRuns({
          parent: `${stage.name}/tasks/-`,
          taskRuns: taskRunListToCancel.map((taskRun) => taskRun.name),
        });
      }
    }
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: title.value,
    });

    // notify the issue logic to update issue status
    events.emit("status-changed", { eager: true });
  } finally {
    state.loading = false;
    emit("close");
  }

  // Trying to avoid some kind of concurrency and race condition, we fetch the
  // latest snapshot of issue from the server-side and check whether this
  // transition is applicable again.
  // const latestIssue = await useIssueStore().fetchIssueById(issue.value.id);

  // const { action: transition } = props;
  // const applicableList = getApplicableIssueStatusTransitionList(latestIssue);
  // if (!isApplicableTransition(transition, applicableList)) {
  //   return cleanup();
  // }

  // changeIssueStatus(transition.to, comment);
  // isTransiting.value = false;
  // emit("updated");
};
</script>
