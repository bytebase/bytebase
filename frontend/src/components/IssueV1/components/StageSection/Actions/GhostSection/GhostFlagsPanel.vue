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
          <FlagsForm v-model:flags="flags" :readonly="readonly" />
        </div>
      </template>
      <template #footer>
        <div class="flex flex-row justify-end gap-x-3">
          <NButton @click="close">{{ $t("common.cancel") }}</NButton>

          <NTooltip :disabled="errors.length === 0">
            <template #trigger>
              <NButton
                type="primary"
                :disabled="!isDirty"
                :loading="isUpdating"
                @click="trySave"
              >
                {{ $t("common.save") }}
              </NButton>
            </template>
            <template #default>
              <ErrorList :errors="errors" />
            </template>
          </NTooltip>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { cloneDeep, isEqual, uniqBy } from "lodash-es";
import { NButton, NTooltip } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  chooseUpdateTarget,
  databaseForTask,
  notifyNotEditableLegacyIssue,
  specForTask,
  stageForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import ErrorList from "@/components/misc/ErrorList.vue";
import { Drawer, DrawerContent, RichDatabaseName } from "@/components/v2";
import { rolloutServiceClient } from "@/grpcweb";
import { pushNotification } from "@/store";
import { Plan_Spec, Task_Type } from "@/types/proto/v1/rollout_service";
import FlagsForm from "./FlagsForm";
import { allowChangeTaskGhostFlags, useIssueGhostContext } from "./common";

const { t } = useI18n();
const { showFlagsPanel, denyEditGhostFlagsReasons } = useIssueGhostContext();
const {
  isCreating,
  issue,
  selectedTask: task,
  dialog,
  events,
} = useIssueContext();
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

const isDirty = computed(() => {
  return !isEqual(config.value?.ghostFlags ?? {}, flags.value);
});
const errors = computed(() => {
  if (denyEditGhostFlagsReasons.value.length > 0) {
    return denyEditGhostFlagsReasons.value;
  }
  const errors: string[] = [];
  if (!isDirty.value) {
    errors.push(t("task.online-migration.error.nothing-changed"));
  }
  return errors;
});

const readonly = computed(() => {
  if (isCreating.value) return false;
  return denyEditGhostFlagsReasons.value.length > 0;
});

const isDeploymentConfig = computed(() => {
  return !!spec.value?.changeDatabaseConfig?.target?.match(
    /\/deploymentConfigs\/[^/]+/
  );
});

const chooseUpdateSpecs = async () => {
  const { tasks } = await chooseUpdateTarget(
    issue.value,
    task.value,
    (task) =>
      task.type === Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC &&
      allowChangeTaskGhostFlags(issue.value, task),
    dialog,
    t("task.online-migration.ghost-parameters"),
    isDeploymentConfig.value
  );

  const specs = tasks
    .map((task) => specForTask(issue.value.planEntity, task))
    .filter((spec) => !!spec) as Plan_Spec[];

  return uniqBy(specs, (spec) => spec.id);
};

const close = () => {
  flags.value = cloneDeep(config.value?.ghostFlags ?? {});
  showFlagsPanel.value = false;
};

const trySave = async () => {
  const specs = await chooseUpdateSpecs();
  if (specs.length === 0) return;

  if (isCreating.value) {
    specs.forEach((spec) => {
      const config = spec.changeDatabaseConfig;
      if (!config) return;
      config.ghostFlags = cloneDeep(flags.value);
    });
    close();
  } else {
    isUpdating.value = true;
    try {
      const planPatch = cloneDeep(issue.value.planEntity);
      if (!planPatch) {
        notifyNotEditableLegacyIssue();
        return;
      }

      const distinctSpecIds = new Set(specs.map((spec) => spec.id));
      const specsToPatch = planPatch.steps
        .flatMap((step) => step.specs)
        .filter((spec) => distinctSpecIds.has(spec.id));

      for (let i = 0; i < specsToPatch.length; i++) {
        const spec = specsToPatch[i];
        const config = spec.changeDatabaseConfig;
        if (!config) continue;
        config.ghostFlags = cloneDeep(flags.value);
      }

      const updatedPlan = await rolloutServiceClient.updatePlan({
        plan: planPatch,
        updateMask: ["steps"],
      });

      issue.value.planEntity = updatedPlan;

      events.emit("status-changed", { eager: true });
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
      close();
    } finally {
      isUpdating.value = false;
    }
  }
};

watch(
  () => config.value?.ghostFlags,
  (newFlags, oldFlags) => {
    if (isEqual(newFlags, oldFlags)) {
      return;
    }
    flags.value = cloneDeep(newFlags ?? {});
  },
  { immediate: true, deep: true }
);
</script>
