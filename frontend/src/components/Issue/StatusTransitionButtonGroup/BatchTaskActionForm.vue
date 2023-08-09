<template>
  <div class="px-4 flex flex-col gap-y-6 divide-y divide-block-border">
    <div class="mt-2 grid grid-cols-1 gap-x-4 sm:grid-cols-4">
      <div class="sm:col-span-4 mb-4">
        <label for="about" class="textlabel">
          {{ $t("common.tasks") }}
        </label>
        <ul class="mt-1 max-h-[6rem] overflow-y-auto">
          <li
            v-for="item in distinctTaskList"
            :key="item.task.id"
            class="text-sm textinfolabel"
          >
            <span class="textinfolabel">
              {{ item.task.name }}
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
      </div>

      <div class="sm:col-span-4 w-112 min-w-full">
        <label for="about" class="textlabel">
          {{ $t("issue.status-transition.form.note") }}
        </label>
        <div class="mt-1">
          <textarea
            ref="commentTextArea"
            v-model="state.comment"
            rows="3"
            class="textarea block w-full resize-none mt-1 text-sm text-control rounded-md whitespace-pre-wrap"
            :placeholder="$t('issue.status-transition.form.placeholder')"
            @input="
              (e) => {
                sizeToFit(e.target as HTMLTextAreaElement);
              }
            "
            @focus="
              (e) => {
                sizeToFit(e.target as HTMLTextAreaElement);
              }
            "
          ></textarea>
        </div>
      </div>
    </div>

    <!-- Update button group -->
    <div class="flex justify-end items-center pt-5">
      <button
        type="button"
        class="btn-normal mt-3 px-4 py-2 sm:mt-0 sm:w-auto"
        @click.prevent="$emit('cancel')"
      >
        {{ cancelText }}
      </button>
      <button
        type="button"
        class="ml-3 px-4 py-2"
        :class="submitButtonStyle"
        @click.prevent="handleSubmit"
      >
        {{ displayingOkText }}
      </button>
    </div>

    <div
      v-if="state.loading"
      class="absolute inset-0 flex flex-col items-center justify-center bg-white/50 gap-y-1"
    >
      <BBSpin />
      <div
        class="flex items-center justify-center space-x-1 text-sm text-control"
      >
        <span>{{ progress.finished }}</span>
        <span>/</span>
        <span>{{ progress.total }}</span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { groupBy } from "lodash-es";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useTaskStore } from "@/store";
import { Issue, Task, TaskStatusPatch } from "@/types";
import { TaskStatusTransition } from "@/utils";
import { useIssueLogic } from "../logic";

interface LocalState {
  success: Set<Task>;
  failed: Set<Task>;
  comment: string;
  loading: boolean;
}
const props = withDefaults(
  defineProps<{
    issue: Issue;
    okText?: string;
    taskList: Task[];
    transition: TaskStatusTransition;
    title: string;
  }>(),
  {
    okText: "",
    loading: false,
  }
);

const emit = defineEmits<{
  (event: "cancel"): void;
  (event: "finish"): void;
}>();

const { t } = useI18n();
const commentTextArea = ref("");
const taskStore = useTaskStore();
const { onStatusChanged } = useIssueLogic();

const state = reactive<LocalState>({
  success: new Set(),
  failed: new Set(),
  comment: "",
  loading: false,
});

const progress = computed(() => {
  const finished = state.success.size + state.failed.size;
  const total = props.taskList.length;
  return { finished, total };
});

const cancelText = computed(() => t("common.cancel"));
const displayingOkText = computed(() => {
  if (props.okText === cancelText.value) {
    // We don't want to see [Cancel] [Cancel]
    // So fall back to [Cancel] [Confirm] if okText===cancelText
    return t("common.confirm");
  }
  return props.okText;
});

const submitButtonStyle = computed(() => {
  switch (props.transition.type) {
    case "RUN":
      return "btn-primary";
    case "ROLLOUT":
      return "btn-primary";
    case "RETRY":
      return "btn-primary";
    case "CANCEL":
      return "btn-danger";
    case "SKIP":
      return "btn-primary";
    case "RESTART":
      return "btn-normal";
  }
  return ""; // only to make eslint happy
});

const distinctTaskList = computed(() => {
  type DistinctTaskList = { task: Task; similar: Task[] };
  const groups = groupBy(props.taskList, (task) => task.name);

  return Object.keys(groups).map<DistinctTaskList>((taskName) => {
    const [task, ...similar] = groups[taskName];
    return { task, similar };
  });
});

const patchTaskStatus = async (task: Task) => {
  const status = props.transition.to;
  const comment = state.comment;

  const taskStatusPatch: TaskStatusPatch = {
    status,
    comment,
  };
  try {
    const { issue } = props;
    await taskStore.updateStatus({
      issueId: issue.id,
      pipelineId: issue.pipeline!.id,
      taskId: task.id,
      taskStatusPatch,
    });
    state.success.add(task);
  } catch {
    state.failed.add(task);
  }
};

const handleSubmit = async () => {
  state.loading = true;
  state.success.clear();
  state.failed.clear();
  try {
    const requests = props.taskList.map(patchTaskStatus);
    await Promise.allSettled(requests);
    const parts = [`${t("common.total")}: ${props.taskList.length}`];
    const successCount = state.success.size;
    const failedCount = state.failed.size;
    if (successCount > 0) {
      parts.push(`${t("common.success")}: ${successCount}`);
    }
    if (failedCount > 0) {
      parts.push(`${t("common.failed")}: ${failedCount}`);
    }
    pushNotification({
      module: "bytebase",
      style:
        failedCount > 0 ? (successCount > 0 ? "WARN" : "CRITICAL") : "SUCCESS",
      title: props.title,
      description: parts.join(", "),
    });
    emit("finish");
  } finally {
    onStatusChanged(true);
    state.loading = false;
  }
};
</script>
