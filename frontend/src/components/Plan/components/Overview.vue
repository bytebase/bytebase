<template>
  <div class="w-full px-4 overflow-y-auto">
    <div class="max-w-4xl mx-auto space-y-4 mt-4">
      <!-- Plan Header -->
      <div>
        <h2 class="text-2xl font-semibold mb-2">
          {{ $t("common.overview") }}
        </h2>
        <p class="text-control-light">
          {{ $t("plan.overview.description") }}
        </p>
      </div>

      <!-- Statistics Cards -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div class="bg-white rounded-md border px-3 py-2">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-sm text-control-light">
                {{ $t("plan.overview.total-specs") }}
              </p>
              <p class="text-2xl font-semibold mt-1">
                {{ statistics.totalSpecs }}
              </p>
            </div>
            <LayersIcon class="w-8 h-8 text-control-light" stroke-width="1" />
          </div>
        </div>

        <div class="bg-white rounded-md border px-3 py-2">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-sm text-control-light">
                {{ $t("plan.overview.total-targets") }}
              </p>
              <p class="text-2xl font-semibold mt-1">
                {{ statistics.totalTargets }}
              </p>
            </div>
            <DatabaseIcon class="w-8 h-8 text-control-light" stroke-width="1" />
          </div>
        </div>

        <div class="bg-white rounded-md border px-3 py-2">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-sm text-control-light">
                {{ $t("plan.overview.check-status") }}
              </p>
              <div class="flex items-center gap-3 mt-1">
                <div
                  v-if="statistics.checkStatus.error > 0"
                  class="flex items-center gap-1"
                >
                  <XCircleIcon class="w-5 h-5 text-error" />
                  <span class="text-lg font-semibold text-error">{{
                    statistics.checkStatus.error
                  }}</span>
                </div>
                <div
                  v-if="statistics.checkStatus.warning > 0"
                  class="flex items-center gap-1"
                >
                  <AlertCircleIcon class="w-5 h-5 text-warning" />
                  <span class="text-lg font-semibold text-warning">{{
                    statistics.checkStatus.warning
                  }}</span>
                </div>
                <div
                  v-if="statistics.checkStatus.success > 0"
                  class="flex items-center gap-1"
                >
                  <CheckCircleIcon class="w-5 h-5 text-success" />
                  <span class="text-lg font-semibold text-success">{{
                    statistics.checkStatus.success
                  }}</span>
                </div>
                <span
                  v-if="statistics.checkStatus.total === 0"
                  class="text-lg text-control"
                >
                  {{ $t("plan.overview.no-checks") }}
                </span>
              </div>
            </div>
            <ActivityIcon class="w-8 h-8 text-control-light" stroke-width="1" />
          </div>
        </div>
      </div>

      <!-- Affected Resources -->
      <div v-if="affectedResources.length > 0">
        <h2 class="text-lg font-semibold mb-3">
          {{ $t("plan.overview.affected-resources") }}
        </h2>
        <div class="flex flex-row justify-start items-center gap-3">
          <div
            v-for="resource in affectedResources"
            :key="resource.name"
            class="flex items-center gap-1 px-3 py-2 max-w-[50%] rounded-md border"
          >
            <InstanceV1EngineIcon
              v-if="resource.type === 'instance' && resource.instance"
              :instance="resource.instance"
              :tooltip="false"
              size="medium"
            />
            <FolderIcon
              v-else-if="resource.type === 'databaseGroup'"
              class="w-5 h-5 text-control mt-0.5"
            />
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <!-- Instance info -->
                <template v-if="resource.type === 'instance'">
                  <span
                    v-if="resource.instance?.environmentEntity"
                    class="text-sm text-gray-500"
                  >
                    ({{ resource.instance.environmentEntity.title }})
                  </span>
                  <span class="font-medium truncate">
                    {{
                      resource.instance
                        ? instanceV1Name(resource.instance)
                        : resource.name
                    }}
                  </span>
                  <span class="text-sm text-control-light shrink-0">
                    {{
                      resource.databases.length === 1
                        ? t("plan.targets.one-database")
                        : t("plan.targets.multiple-databases", {
                            count: resource.databases.length,
                          })
                    }}
                  </span>
                </template>

                <!-- Database Group info -->
                <template v-else-if="resource.type === 'databaseGroup'">
                  <span class="font-medium truncate">
                    {{ resource.databaseGroup?.title || resource.name }}
                  </span>
                  <span
                    class="text-xs px-1.5 py-0.5 rounded-md bg-blue-100 text-blue-600"
                  >
                    {{ t("plan.targets.database-group") }}
                  </span>
                </template>
              </div>
            </div>
          </div>
        </div>
      </div>

      <DescriptionSection />
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
  FolderIcon,
} from "lucide-vue-next";
import { computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import { InstanceV1EngineIcon } from "@/components/v2/Model/Instance";
import { useInstanceV1Store, useDBGroupStore } from "@/store";
import { PlanCheckRun_Result_Status } from "@/types/proto/v1/plan_service";
import { instanceV1Name, extractDatabaseResourceName } from "@/utils";
import { usePlanContext } from "../logic/context";
import { targetsForSpec } from "../logic/plan";
import DescriptionSection from "./DescriptionSection.vue";

const { t } = useI18n();
const { plan } = usePlanContext();
const instanceStore = useInstanceV1Store();
const dbGroupStore = useDBGroupStore();

// Calculate statistics
const statistics = computed(() => {
  let totalTargets = 0;
  const checkStatus = {
    total: 0,
    success:
      plan.value.planCheckRunStatusCount[PlanCheckRun_Result_Status.SUCCESS] ||
      0,
    warning:
      plan.value.planCheckRunStatusCount[PlanCheckRun_Result_Status.WARNING] ||
      0,
    error:
      plan.value.planCheckRunStatusCount[PlanCheckRun_Result_Status.ERROR] || 0,
  };
  checkStatus.total =
    checkStatus.success + checkStatus.warning + checkStatus.error || 0;
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
  const resourceList: Array<{
    type: "instance" | "databaseGroup";
    name: string;
    instance?: any;
    databaseGroup?: any;
    databases: string[];
  }> = [];

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
