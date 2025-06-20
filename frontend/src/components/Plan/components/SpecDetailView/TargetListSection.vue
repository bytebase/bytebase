<template>
  <div class="px-4 flex flex-col gap-y-2">
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-1">
        <h3 class="text-base font-medium">{{ $t("plan.targets.title") }}</h3>
        <span class="text-control-light" v-if="targets.length > 1"
          >({{ targets.length }})</span
        >
      </div>
      <div class="flex items-center gap-1">
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

    <div class="relative flex-1">
      <div
        v-if="isLoadingTargets"
        class="flex items-center justify-center py-8"
      >
        <BBSpin />
      </div>
      <div
        v-else-if="targets.length > 0"
        class="flex flex-wrap gap-2 overflow-y-auto"
      >
        <div
          v-for="item in tableData"
          :key="item.name"
          class="inline-flex items-center gap-x-1.5 px-3 py-1.5 border rounded-lg transition-all cursor-default max-w-[20rem]"
        >
          <EngineIcon
            v-if="item.type === 'database' && item.engine"
            :engine="item.engine"
            custom-class="w-4 h-4 text-control-light flex-shrink-0"
          />
          <component
            v-else
            :is="item.icon"
            class="w-4 h-4 text-control-light flex-shrink-0"
          />
          <span class="text-sm text-gray-500" v-if="item.environment">
            ({{ item.environment }})
          </span>
          <div class="flex items-center gap-x-1 min-w-0 text-sm">
            <NEllipsis :line-clamp="1">
              <span class="font-medium">{{ item.name }}</span>
            </NEllipsis>
          </div>
          <div
            class="text-xs px-2 py-0.5 rounded-md bg-control-bg text-control-light flex-shrink-0"
          >
            {{ getTypeLabel(item.type) }}
          </div>
          <div
            v-if="item.type === 'databaseGroup'"
            class="flex items-center justify-end cursor-pointer opacity-60 hover:opacity-100"
            @click="gotoDatabaseGroupDetailPage(item.target)"
          >
            <ExternalLinkIcon class="w-4 h-auto" />
          </div>
        </div>
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
import {
  ServerIcon,
  DatabaseIcon,
  FolderIcon,
  EditIcon,
  ExternalLinkIcon,
} from "lucide-vue-next";
import { NEllipsis, NButton } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBSpin } from "@/bbkit";
import EngineIcon from "@/components/Icon/EngineIcon.vue";
import { planServiceClient } from "@/grpcweb";
import { PROJECT_V1_ROUTE_DATABASE_GROUP_DETAIL } from "@/router/dashboard/projectV1";
import {
  useInstanceV1Store,
  useDatabaseV1Store,
  useDBGroupStore,
  useProjectV1Store,
  pushNotification,
  batchGetOrFetchDatabases,
  getProjectNameAndDatabaseGroupName,
} from "@/store";
import type { Engine } from "@/types/proto/v1/common";
import {
  extractInstanceResourceName,
  instanceV1Name,
  extractDatabaseResourceName,
  extractDatabaseGroupName,
  extractProjectResourceName,
} from "@/utils";
import { usePlanContext } from "../../logic/context";
import { targetsForSpec } from "../../logic/plan";
import AllTargetsDrawer from "./AllTargetsDrawer.vue";
import TargetsSelectorDrawer from "./TargetsSelectorDrawer.vue";
import { usePlanSpecContext } from "./context";

interface TargetRow {
  target: string;
  type: "instance" | "database" | "databaseGroup";
  icon: any;
  name: string;
  instance?: string;
  environment?: string;
  description?: string;
  engine?: Engine;
}

const DEFAULT_VISIBLE_TARGETS = 20;

const { t } = useI18n();
const router = useRouter();
const { plan, isCreating, events } = usePlanContext();
const { selectedSpec } = usePlanSpecContext();
const instanceStore = useInstanceV1Store();
const databaseStore = useDatabaseV1Store();
const dbGroupStore = useDBGroupStore();
const projectStore = useProjectV1Store();

const isLoadingTargets = ref(false);
const showTargetsSelector = ref(false);
const showAllTargetsDrawer = ref(false);

const targets = computed(() => {
  if (!selectedSpec.value) return [];
  return targetsForSpec(selectedSpec.value);
});

const isCreateDatabaseSpec = computed(() => {
  return !!selectedSpec.value?.createDatabaseConfig;
});

const project = computed(() => {
  if (!plan.value?.name) return undefined;
  const projectName = `projects/${extractProjectResourceName(plan.value.name)}`;
  return projectStore.getProjectByName(projectName);
});

// Only allow editing in creation mode or if the plan is editable.
// An empty string for `plan.value.rollout` indicates that the plan is in a draft or uninitialized state,
// which allows edits to be made.
const allowEdit = computed(() => {
  return (isCreating.value || plan.value.rollout === "") && selectedSpec.value;
});

const tableData = computed((): TargetRow[] => {
  if (!selectedSpec.value) return [];

  // Show only the first DEFAULT_VISIBLE_TARGETS targets
  const visibleTargets = targets.value.slice(
    0,
    Math.min(DEFAULT_VISIBLE_TARGETS, targets.value.length)
  );

  return visibleTargets.map((target): TargetRow => {
    // For create database spec, target is instance
    if (isCreateDatabaseSpec.value) {
      const instanceResourceName = extractInstanceResourceName(target);
      const instance = instanceStore.getInstanceByName(instanceResourceName);

      return {
        target,
        type: "instance",
        icon: ServerIcon,
        name: instance ? instanceV1Name(instance) : instanceResourceName,
        environment: instance?.environmentEntity?.title || "Unknown",
        description: instance?.title || "",
      };
    }

    // For database group targets
    if (target.includes("/databaseGroups/")) {
      const groupName = extractDatabaseGroupName(target);
      const dbGroup = dbGroupStore.getDBGroupByName(target);

      return {
        target,
        type: "databaseGroup",
        icon: FolderIcon,
        name: dbGroup?.title || groupName,
      };
    }

    // For regular database targets
    const database = databaseStore.getDatabaseByName(target);

    if (!database) {
      // Fallback when database is not found
      const { instance: instanceId, databaseName } =
        extractDatabaseResourceName(target);
      if (instanceId && databaseName) {
        return {
          target,
          type: "database",
          icon: DatabaseIcon,
          name: databaseName,
          instance: instanceId,
          description: t("plan.targets.database-not-found"),
        };
      }
    }

    const instance = database?.instanceResource;

    return {
      target,
      type: "database",
      icon: DatabaseIcon,
      name: database?.databaseName || target,
      instance: instance ? instanceV1Name(instance) : "",
      environment: database?.effectiveEnvironmentEntity?.title || "",
      engine: instance?.engine,
    };
  });
});

const getTypeLabel = (type: TargetRow["type"]) => {
  const typeLabels = {
    instance: t("plan.targets.type.instance"),
    database: t("plan.targets.type.database"),
    databaseGroup: t("plan.targets.type.database-group"),
  };
  return typeLabels[type];
};

const handleUpdateTargets = async (targets: string[]) => {
  if (!selectedSpec.value) return;

  // Update the targets in the spec.
  const config =
    selectedSpec.value.changeDatabaseConfig ||
    selectedSpec.value.exportDataConfig;
  if (config) {
    config.targets = targets;
  }

  if (!isCreating.value) {
    await planServiceClient.updatePlan({
      plan: plan.value,
      updateMask: ["specs"],
    });
    events.emit("status-changed", {
      eager: true,
    });
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
    const instanceTargets: string[] = [];
    const dbGroupTargets: string[] = [];

    for (const target of visibleTargets) {
      if (isCreateDatabaseSpec.value) {
        instanceTargets.push(target);
      } else if (target.includes("/databaseGroups/")) {
        dbGroupTargets.push(target);
      } else {
        databaseTargets.push(target);
      }
    }

    if (databaseTargets.length > 0) {
      await batchGetOrFetchDatabases(databaseTargets);
    }

    const instancePromises = instanceTargets.map((target) => {
      const instanceResourceName = extractInstanceResourceName(target);
      return instanceStore.getOrFetchInstanceByName(instanceResourceName);
    });

    const dbGroupPromises = dbGroupTargets.map((target) =>
      dbGroupStore.getOrFetchDBGroupByName(target)
    );

    await Promise.allSettled([...instancePromises, ...dbGroupPromises]);
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
