<template>
  <div class="flex flex-col gap-y-2 pt-2 px-4">
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-1">
        <h3 class="text-base font-medium">{{ $t("plan.targets.title") }}</h3>
        <span class="text-control-light" v-if="targets.length > 1"
          >({{ targets.length }})</span
        >
      </div>
      <div class="flex items-center gap-1">
        <NButton
          v-if="allowEdit"
          size="small"
          @click="showTargetsSelector = true"
        >
          <template #icon>
            <EditIcon class="w-4 h-4" />
          </template>
          {{ $t("common.edit") }}
        </NButton>
      </div>
    </div>

    <div class="relative w-full flex-1">
      <div
        v-if="isLoadingTargets"
        class="flex items-center justify-center py-8"
      >
        <BBSpin />
      </div>
      <div
        v-else-if="targets.length > 0"
        class="w-full flex flex-wrap gap-2 overflow-y-auto"
      >
        <div
          v-for="target in visibleTargets"
          :key="target"
          class="inline-flex items-center gap-x-1 px-2 py-1 border rounded-lg transition-all cursor-default"
        >
          <template v-if="isValidDatabaseName(target)">
            <DatabaseDisplay :database="target" show-environment />
          </template>
          <template v-else-if="isValidDatabaseGroupName(target)">
            <DatabaseGroupIcon
              class="w-4 h-4 text-control-light flex-shrink-0"
            />
            <DatabaseGroupName
              :database-group="
                dbGroupStore.getDBGroupByName(target) as DatabaseGroup
              "
              :link="false"
              :plain="true"
              class="text-sm"
            />
            <NTag size="tiny" round :bordered="false">
              {{ $t("plan.targets.type.database-group") }}
            </NTag>
            <div
              class="flex items-center justify-end cursor-pointer opacity-60 hover:opacity-100"
              @click="gotoDatabaseGroupDetailPage(target)"
            >
              <ExternalLinkIcon class="w-4 h-auto" />
            </div>
          </template>
          <template v-else>
            <!-- Unknown resource -->
            <span class="text-sm">{{ target }}</span>
          </template>
        </div>

        <NButton
          v-if="targets.length > DEFAULT_VISIBLE_TARGETS"
          size="small"
          quaternary
          @click="showAllTargetsDrawer = true"
        >
          {{
            $t("plan.targets.view-all", {
              count: targets.length,
            })
          }}
        </NButton>
      </div>
      <div v-else class="text-center text-control-light py-8">
        {{ $t("plan.targets.no-targets-found") }}
      </div>
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
import { EditIcon, ExternalLinkIcon } from "lucide-vue-next";
import { NButton, NTag } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import DatabaseGroupIcon from "@/components/DatabaseGroupIcon.vue";
import DatabaseGroupName from "@/components/v2/Model/DatabaseGroupName.vue";
import { planServiceClientConnect } from "@/grpcweb";
import { PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL } from "@/router/dashboard/projectV1";
import {
  useDBGroupStore,
  useProjectV1Store,
  pushNotification,
  batchGetOrFetchDatabases,
  getProjectNameAndDatabaseGroupName,
  projectNamePrefix,
} from "@/store";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import {
  DatabaseGroupView,
  type DatabaseGroup,
} from "@/types/proto-es/v1/database_group_service_pb";
import { UpdatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { extractProjectResourceName } from "@/utils";
import { usePlanContext } from "../../logic/context";
import DatabaseDisplay from "../common/DatabaseDisplay.vue";
import AllTargetsDrawer from "./AllTargetsDrawer.vue";
import TargetsSelectorDrawer from "./TargetsSelectorDrawer.vue";
import { useSelectedSpec } from "./context";

const DEFAULT_VISIBLE_TARGETS = 20;

const { t } = useI18n();
const router = useRouter();
const { plan, isCreating, readonly } = usePlanContext();
const selectedSpec = useSelectedSpec();
const dbGroupStore = useDBGroupStore();
const projectStore = useProjectV1Store();

const isLoadingTargets = ref(false);
const showTargetsSelector = ref(false);
const showAllTargetsDrawer = ref(false);

const targets = computed(() => {
  if (!selectedSpec.value) return [];
  if (
    selectedSpec.value.config.case === "changeDatabaseConfig" ||
    selectedSpec.value.config.case === "exportDataConfig"
  ) {
    return selectedSpec.value.config.value.targets;
  }
  return [];
});

const project = computed(() => {
  if (!plan.value?.name) return undefined;
  const projectName = `${projectNamePrefix}${extractProjectResourceName(plan.value.name)}`;
  return projectStore.getProjectByName(projectName);
});

// Only allow editing in creation mode or if the plan is editable and not readonly.
// An empty string for `plan.value.rollout` indicates that the plan is in a draft or uninitialized state,
// which allows edits to be made.
const allowEdit = computed(() => {
  if (readonly.value) return false;
  return (isCreating.value || plan.value.rollout === "") && selectedSpec.value;
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
    const visibleTargets = targets.value.slice(0, DEFAULT_VISIBLE_TARGETS);
    const databaseTargets: string[] = [];
    const dbGroupTargets: string[] = [];

    for (const target of visibleTargets) {
      if (isValidDatabaseGroupName(target)) {
        dbGroupTargets.push(target);
      } else {
        databaseTargets.push(target);
      }
    }

    if (databaseTargets.length > 0) {
      await batchGetOrFetchDatabases(databaseTargets);
    }

    const dbGroupPromises = dbGroupTargets.map((target) =>
      dbGroupStore.getOrFetchDBGroupByName(target, {
        view: DatabaseGroupView.FULL,
      })
    );

    await Promise.allSettled([...dbGroupPromises]);
  } catch {
    // Ignore errors
  } finally {
    isLoadingTargets.value = false;
  }
};

const gotoDatabaseGroupDetailPage = (dbGroup: string) => {
  const [projectId, databaseGroupName] =
    getProjectNameAndDatabaseGroupName(dbGroup);
  const url = router.resolve({
    name: PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL,
    params: {
      projectId,
      databaseGroupName,
    },
  }).fullPath;
  window.open(url, "_blank");
};

// Watch for target changes and load data
watch(
  targets,
  () => {
    loadTargetData();
  },
  { immediate: true }
);
</script>
