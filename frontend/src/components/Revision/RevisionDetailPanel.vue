<template>
  <div v-if="revision" class="w-full">
    <h1 class="text-xl font-bold text-main truncate">
      {{ revision.file }}
    </h1>
    <div
      class="mt-2 text-control text-base space-x-4 flex flex-row items-center flex-wrap"
    >
      <span>{{ "Version" }}: {{ revision.version }}</span>
      <span>{{ "Hash" }}: {{ revision.sheetSha256.slice(0, 8) }}</span>
      <div
        v-if="creator"
        class="flex flex-row items-center overflow-hidden gap-x-1"
      >
        <BBAvatar size="SMALL" :username="creator.title" />
        <span class="truncate">{{ creator.title }}</span>
      </div>
    </div>
  </div>

  <NDivider />

  <div class="flex flex-col">
    <p class="w-auto flex items-center text-base text-main mb-2">
      <span>{{ $t("common.statement") }}</span>
      <ClipboardIcon
        class="ml-1 w-4 h-4 cursor-pointer hover:opacity-80"
        @click.prevent="copyStatement"
      />
    </p>
    <MonacoEditor
      class="h-auto max-h-[480px] min-h-[120px] border rounded-[3px] text-sm overflow-clip relative"
      :content="statement"
      :readonly="true"
      :auto-height="{ min: 120, max: 480 }"
    />
  </div>

  <NDivider />

  <div v-if="taskRun">
    <p class="w-auto flex items-center text-base text-main mb-2">
      Task run logs
    </p>
    <TaskRunLogTable :task-run="taskRun" :sheet="sheet" />
  </div>
</template>

<script lang="ts" setup>
import { ClipboardIcon } from "lucide-vue-next";
import { NDivider } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import BBAvatar from "@/bbkit/BBAvatar.vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import { rolloutServiceClient } from "@/grpcweb";
import {
  pushNotification,
  useUserStore,
  useRevisionStore,
  useSheetV1Store,
} from "@/store";
import { type ComposedDatabase } from "@/types";
import type { TaskRun } from "@/types/proto/v1/rollout_service";
import {
  extractUserResourceName,
  getSheetStatement,
  toClipboard,
} from "@/utils";
import TaskRunLogTable from "../IssueV1/components/TaskRunSection/TaskRunLogTable/TaskRunLogTable.vue";

interface LocalState {
  loading: boolean;
}

const props = defineProps<{
  database: ComposedDatabase;
  revisionName: string;
}>();

const state = reactive<LocalState>({
  loading: false,
});

const revisionStore = useRevisionStore();
const sheetStore = useSheetV1Store();
const taskRun = ref<TaskRun | undefined>(undefined);

watch(
  () => props.revisionName,
  async (revisionName) => {
    if (!revisionName) {
      return;
    }

    state.loading = true;
    const revision = await revisionStore.getOrFetchRevisionByName(revisionName);
    if (revision) {
      await sheetStore.getOrFetchSheetByName(revision.sheet, "FULL");
      const taskRunData = await rolloutServiceClient.getTaskRun({
        name: revision.taskRun,
      });
      taskRun.value = taskRunData;
    }
    state.loading = false;
  },
  { immediate: true }
);

const revision = computed(() =>
  revisionStore.getRevisionByName(props.revisionName)
);
const sheet = computed(() =>
  revision.value ? sheetStore.getSheetByName(revision.value.sheet) : undefined
);

const statement = computed(() =>
  sheet.value ? getSheetStatement(sheet.value) : ""
);

const creator = computed(() => {
  if (!revision.value) {
    return undefined;
  }
  const email = extractUserResourceName(revision.value.creator);
  return useUserStore().getUserByEmail(email);
});

const copyStatement = async () => {
  toClipboard(statement.value).then(() => {
    pushNotification({
      module: "bytebase",
      style: "INFO",
      title: `Statement copied to clipboard.`,
    });
  });
};
</script>
