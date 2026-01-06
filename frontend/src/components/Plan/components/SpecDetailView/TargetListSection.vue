<template>
  <div class="flex flex-col gap-y-1">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-1">
        <span class="text-base">{{ $t("plan.targets.title") }}</span>
        <span class="text-sm text-control-light" v-if="targets.length > 1"
          >({{ targets.length }})</span
        >
      </div>
      <NButton
        v-if="allowEdit"
        size="small"
        @click="showTargetsSelector = true"
      >
        {{ $t("common.edit") }}
      </NButton>
    </div>

    <div v-if="isLoadingTargets" class="flex items-center justify-center py-2">
      <BBSpin />
    </div>
    <div v-else-if="targets.length > 0" class="flex flex-wrap gap-1.5">
      <div
        v-for="target in visibleTargets"
        :key="target"
        class="inline-flex items-center px-3 py-2 border rounded text-sm"
      >
        <DatabaseDisplay v-if="isValidDatabaseName(target)" :database="target" show-environment />
        <DatabaseGroupTargetDisplay v-else-if="isValidDatabaseGroupName(target)" :target="target" />
        <span v-else>{{ target }}</span>
      </div>
      <NButton
        v-if="targets.length > DEFAULT_VISIBLE_TARGETS"
        size="small"
        quaternary
        @click="showAllTargetsDrawer = true"
      >
        {{ $t("plan.targets.view-all", { count: targets.length }) }}
      </NButton>
    </div>
    <div v-else class="text-sm text-control-light py-1">
      {{ $t("plan.targets.no-targets-found") }}
    </div>

    <TargetsSelectorDrawer
      v-if="project"
      v-model:show="showTargetsSelector"
      :current-targets="targets"
      @confirm="handleUpdateTargets"
    />

    <AllTargetsDrawer v-model:show="showAllTargetsDrawer" :targets="targets" />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import { NButton } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBSpin } from "@/bbkit";
import { planServiceClientConnect } from "@/connect";
import {
  projectNamePrefix,
  pushNotification,
  useDatabaseV1Store,
  useDBGroupStore,
  useProjectV1Store,
} from "@/store";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import { UpdatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { extractProjectResourceName } from "@/utils";
import { usePlanContext } from "../../logic/context";
import DatabaseDisplay from "../common/DatabaseDisplay.vue";
import AllTargetsDrawer from "./AllTargetsDrawer.vue";
import { DEFAULT_VISIBLE_TARGETS, useSelectedSpec } from "./context";
import DatabaseGroupTargetDisplay from "./DatabaseGroupTargetDisplay.vue";
import TargetsSelectorDrawer from "./TargetsSelectorDrawer.vue";

const { t } = useI18n();
const {
  plan,
  isCreating,
  readonly,
  allowEdit: hasPermission,
} = usePlanContext();
const { selectedSpec, targets, getDatabaseTargets } = useSelectedSpec();
const dbGroupStore = useDBGroupStore();
const projectStore = useProjectV1Store();
const dbStore = useDatabaseV1Store();

const isLoadingTargets = ref(false);
const showTargetsSelector = ref(false);
const showAllTargetsDrawer = ref(false);

const project = computed(() => {
  if (!plan.value?.name) return undefined;
  const projectName = `${projectNamePrefix}${extractProjectResourceName(plan.value.name)}`;
  return projectStore.getProjectByName(projectName);
});

// Only allow editing in creation mode or if the plan is editable and not readonly.
// An empty string for `plan.value.hasRollout` indicates that the plan is in a draft or uninitialized state,
// which allows edits to be made.
const allowEdit = computed(() => {
  if (readonly.value) {
    return false;
  }
  if (!hasPermission.value) {
    return false;
  }
  return (isCreating.value || !plan.value.hasRollout) && selectedSpec.value;
});

// Separate targets by type
const visibleTargets = computed(() => {
  return targets.value.slice(
    0,
    Math.min(DEFAULT_VISIBLE_TARGETS, targets.value.length)
  );
});

const handleUpdateTargets = async (targets: string[]) => {
  if (!selectedSpec.value) return;

  // Update the targets in the spec.
  if (selectedSpec.value.config?.case === "changeDatabaseConfig") {
    selectedSpec.value.config.value.targets = targets;
  } else if (selectedSpec.value.config?.case === "exportDataConfig") {
    selectedSpec.value.config.value.targets = targets;
  }

  if (!isCreating.value) {
    const request = create(UpdatePlanRequestSchema, {
      plan: plan.value,
      updateMask: { paths: ["specs"] },
    });
    const response = await planServiceClientConnect.updatePlan(request);
    Object.assign(plan.value, response);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  }
};

// Load target data when targets change
const loadTargetData = async () => {
  if (targets.value.length === 0) {
    return;
  }

  isLoadingTargets.value = true;

  try {
    // Fetch data for visible targets only
    const { databaseTargets, dbGroupTargets } = await getDatabaseTargets(
      visibleTargets.value
    );

    await dbStore.batchGetOrFetchDatabases(databaseTargets);
    await Promise.allSettled(
      dbGroupTargets.map((target) =>
        dbGroupStore.getOrFetchDBGroupByName(target, {
          view: DatabaseGroupView.FULL,
        })
      )
    );
  } catch {
    // Ignore errors
  } finally {
    isLoadingTargets.value = false;
  }
};

// Watch for target changes and load data
watch(
  targets,
  (newTargets, oldTargets) => {
    // Only reload if targets actually changed
    if (isEqual(newTargets, oldTargets)) {
      return;
    }
    loadTargetData();
  },
  { immediate: true }
);
</script>
