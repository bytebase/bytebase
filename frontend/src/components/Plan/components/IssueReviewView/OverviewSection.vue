<template>
  <div class="w-full py-4 overflow-y-auto">
    <!-- Statistics Cards -->
    <div :class="'grid grid-cols-1 sm:grid-cols-3 gap-4'">
      <!-- Plan Statistics Column -->
      <!-- Specs Card -->
      <div class="bg-white rounded-md border px-3 py-2">
        <div class="flex items-center justify-between">
          <div>
            <p class="textlabel">
              {{ $t("plan.overview.total-specs") }}
            </p>
            <p class="text-xl font-medium mt-1">
              {{ statistics.totalSpecs }}
            </p>
          </div>
          <LayersIcon class="w-8 h-8 text-control-light" stroke-width="1" />
        </div>
      </div>

      <!-- Targets Card -->
      <div class="bg-white rounded-md border px-3 py-2">
        <div class="flex items-center justify-between">
          <div>
            <p class="textlabel">
              {{ $t("plan.overview.total-targets") }}
            </p>
            <p class="text-xl font-medium mt-1">
              {{ statistics.totalTargets }}
            </p>
          </div>
          <DatabaseIcon class="w-8 h-8 text-control-light" stroke-width="1" />
        </div>
      </div>

      <!-- Checks Card -->
      <div class="bg-white rounded-md border px-3 py-2">
        <div class="flex items-center justify-between">
          <div>
            <p class="textlabel">
              {{ $t("plan.navigator.checks") }}
            </p>
            <div class="flex items-center gap-3 mt-1">
              <div
                v-if="statistics.checkStatus.error > 0"
                class="flex items-center gap-1"
              >
                <XCircleIcon class="w-5 h-5 text-error" />
                <span class="text-xl font-medium text-error">{{
                  statistics.checkStatus.error
                }}</span>
              </div>
              <div
                v-if="statistics.checkStatus.warning > 0"
                class="flex items-center gap-1"
              >
                <AlertCircleIcon class="w-5 h-5 text-warning" />
                <span class="text-xl font-medium text-warning">{{
                  statistics.checkStatus.warning
                }}</span>
              </div>
              <div
                v-if="statistics.checkStatus.success > 0"
                class="flex items-center gap-1"
              >
                <CheckCircleIcon class="w-5 h-5 text-success" />
                <span class="text-xl font-medium text-success">{{
                  statistics.checkStatus.success
                }}</span>
              </div>
              <span
                v-if="statistics.checkStatus.total === 0"
                class="text-xl text-control"
              >
                {{ $t("plan.overview.no-checks") }}
              </span>
            </div>
          </div>
          <ActivityIcon class="w-8 h-8 text-control-light" stroke-width="1" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  LayersIcon,
  DatabaseIcon,
  ActivityIcon,
  CheckCircleIcon,
  AlertCircleIcon,
  XCircleIcon,
} from "lucide-vue-next";
import { computed, watch } from "vue";
import { useInstanceV1Store, useDBGroupStore } from "@/store";
import type { ComposedDatabaseGroup, ComposedInstance } from "@/types";
import { PlanCheckRun_Result_Status } from "@/types/proto-es/v1/plan_service_pb";
import { extractDatabaseResourceName } from "@/utils";
import { targetsForSpec, usePlanContext } from "../../logic";

const { plan } = usePlanContext();
const instanceStore = useInstanceV1Store();
const dbGroupStore = useDBGroupStore();

// Calculate statistics
const statistics = computed(() => {
  let totalTargets = 0;
  const checkStatus = {
    total: 0,
    success:
      plan.value.planCheckRunStatusCount[
        PlanCheckRun_Result_Status[PlanCheckRun_Result_Status.SUCCESS]
      ] || 0,
    warning:
      plan.value.planCheckRunStatusCount[
        PlanCheckRun_Result_Status[PlanCheckRun_Result_Status.WARNING]
      ] || 0,
    error:
      plan.value.planCheckRunStatusCount[
        PlanCheckRun_Result_Status[PlanCheckRun_Result_Status.ERROR]
      ] || 0,
  };
  checkStatus.total =
    checkStatus.success + checkStatus.warning + checkStatus.error;
  for (const spec of plan.value.specs) {
    totalTargets += targetsForSpec(spec).length;
  }
  return {
    totalSpecs: plan.value.specs.length,
    totalTargets,
    checkStatus,
  };
});

// Get affected resources
const affectedResources = computed(() => {
  const specs = plan.value?.specs || [];
  const resourceList: Array<
    | {
        type: "instance";
        name: string;
        instance: ComposedInstance;
        databases: string[];
      }
    | {
        type: "databaseGroup";
        name: string;
        databaseGroup?: ComposedDatabaseGroup;
        databases: string[];
      }
  > = [];

  const instanceMap = new Map<string, Set<string>>();
  const dbGroupSet = new Set<string>();

  for (const spec of specs) {
    const targets = targetsForSpec(spec);
    for (const target of targets) {
      // Check if it's a database group
      if (target.includes("/databaseGroups/")) {
        dbGroupSet.add(target);
      } else {
        // Parse instance/database format
        const { instance, databaseName } = extractDatabaseResourceName(target);
        if (instance && databaseName) {
          if (!instanceMap.has(instance)) {
            instanceMap.set(instance, new Set());
          }
          instanceMap.get(instance)!.add(databaseName);
        }
      }
    }
  }

  // Add instances to resource list
  for (const [instanceName, databases] of instanceMap.entries()) {
    const instance = instanceStore.getInstanceByName(instanceName);
    resourceList.push({
      type: "instance",
      name: instanceName,
      instance: instance,
      databases: Array.from(databases),
    });
  }

  // Add database groups to resource list
  for (const dbGroupName of dbGroupSet) {
    const dbGroup = dbGroupStore.getDBGroupByName(dbGroupName);
    resourceList.push({
      type: "databaseGroup",
      name: dbGroupName,
      databaseGroup: dbGroup,
      databases: dbGroup?.matchedDatabases.map((db) => db.name) || [],
    });
  }

  return resourceList;
});

watch(
  () => affectedResources.value,
  () => {
    for (const resource of affectedResources.value) {
      if (resource.type === "instance") {
        instanceStore.getOrFetchInstanceByName(resource.name);
      } else if (resource.type === "databaseGroup") {
        dbGroupStore.getOrFetchDBGroupByName(resource.name);
      }
    }
  },
  { immediate: true }
);
</script>
