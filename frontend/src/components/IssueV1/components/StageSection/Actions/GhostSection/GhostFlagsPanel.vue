<template>
  <Drawer v-model:show="showFlagsPanel">
    <DrawerContent
      :title="title"
      class="w-[100vw] md:max-w-[calc(100vw-8rem)] md:w-[40vw]"
    >
      <template #default>
        <div class="flex flex-col gap-y-4">
          <div
            class="grid gap-y-4 gap-x-4 items-center text-sm"
            style="grid-template-columns: auto 1fr"
          >
            <div v-if="stage" class="contents">
              <label class="font-medium text-control">
                {{ $t("common.stage") }}
              </label>
              <div class="textinfolabel break-all">
                {{ stage.title }}
              </div>
            </div>
            <div class="contents">
              <label class="font-medium text-control">
                {{ $t("common.task") }}
              </label>
              <div class="textinfolabel break-all">
                {{ task.title }}
              </div>
            </div>
            <div class="contents">
              <label class="font-medium text-control">
                {{ $t("common.database") }}
              </label>
              <div class="textinfolabel break-all">
                <RichDatabaseName :database="database" />
              </div>
            </div>
          </div>

          <p class="font-medium text-control">
            {{ $t("task.online-migration.ghost-parameters") }}
          </p>
          <FlagsForm v-model:flags="flags" />

          <div>
            <div>affectedTasks: {{ affectedTasks.length }}</div>
            <div>isDirty: {{ isDirty }}</div>
          </div>
        </div>
      </template>
      <template #footer>
        <div class="flex flex-row justify-end gap-x-3">
          <NButton @click="close">{{ $t("common.cancel") }}</NButton>
          <NButton
            type="primary"
            :disabled="!isDirty"
            :loading="isUpdating"
            @click="handleSave"
          >
            {{ $t("common.save") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { cloneDeep, isEqual } from "lodash-es";
import { NButton } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  databaseForTask,
  specForTask,
  stageForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { Drawer, DrawerContent, RichDatabaseName } from "@/components/v2";
import { pushNotification } from "@/store";
import { Task_Type } from "@/types/proto/v1/rollout_service";
import { flattenTaskV1List } from "@/utils";
import { allowChangeTaskGhostFlags, useIssueGhostContext } from "./common";

const { t } = useI18n();
const { showFlagsPanel } = useIssueGhostContext();
const { isCreating, issue, selectedTask: task } = useIssueContext();
const isUpdating = ref(false);

const stage = computed(() => {
  return stageForTask(issue.value, task.value);
});
const database = computed(() => {
  return databaseForTask(issue.value, task.value);
});
const title = computed(() => {
  return t("task.online-migration.configure-ghost-parameters");
});
const spec = computed(() => {
  return specForTask(issue.value.planEntity, task.value);
});
const config = computed(() => {
  return spec.value?.changeDatabaseConfig;
});
const flags = ref<Record<string, string>>({});

const affectedTasks = computed(() => {
  const spec = specForTask(issue.value.planEntity, task.value);
  if (!spec) return [];
  const tasks = flattenTaskV1List(issue.value.rolloutEntity);
  return tasks
    .filter((task) => task.specId === spec.id)
    .filter((task) => allowChangeTaskGhostFlags(issue.value, task))
    .filter(
      (task) => task.type === Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC
    );
});

const isDirty = computed(() => {
  return !isEqual(config.value?.ghostFlags ?? {}, flags.value);
});

const close = () => {
  flags.value = cloneDeep(config.value?.ghostFlags ?? {});
  showFlagsPanel.value = false;
};

const handleSave = async () => {
  if (isCreating.value) {
    if (!config.value) return;
    config.value.ghostFlags = cloneDeep(flags.value);
    close();
  } else {
    isUpdating.value = true;
    try {
      await new Promise((r) => setTimeout(r, 500));
    } finally {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
      isUpdating.value = false;
      close();
    }
  }
};

watch(
  () => config.value?.ghostFlags,
  (updatedFlags) => {
    flags.value = cloneDeep(updatedFlags ?? {});
  },
  { immediate: true }
);
</script>
