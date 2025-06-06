<template>
  <div class="w-full p-4 overflow-y-auto mt-8">
    <div class="max-w-4xl mx-auto space-y-4">
      <!-- Plan Header -->
      <div>
        <h1 class="text-2xl font-semibold mb-2">Plan Overview</h1>
        <p class="text-control-light">
          View and manage all specifications in this plan
        </p>
      </div>

      <!-- Statistics Cards -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div class="bg-white rounded border p-4">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-sm text-control-light">Total Specs</p>
              <p class="text-2xl font-semibold mt-1">
                {{ statistics.totalSpecs }}
              </p>
            </div>
            <LayersIcon class="w-8 h-8 text-control" />
          </div>
        </div>

        <div class="bg-white rounded border p-4">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-sm text-control-light">Total Targets</p>
              <p class="text-2xl font-semibold mt-1">
                {{ statistics.totalTargets }}
              </p>
            </div>
            <DatabaseIcon class="w-8 h-8 text-control" />
          </div>
        </div>

        <div class="bg-white rounded border p-4">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-sm text-control-light">Check Status</p>
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
                  No checks
                </span>
              </div>
            </div>
            <ActivityIcon class="w-8 h-8 text-control" />
          </div>
        </div>
      </div>

      <!-- Affected Resources -->
      <div v-if="affectedResources.length > 0">
        <h2 class="text-lg font-semibold mb-3">Affected Resources</h2>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div
            v-for="resource in affectedResources"
            :key="resource.name"
            class="flex items-start gap-3 p-3 bg-control-bg rounded"
          >
            <InstanceV1EngineIcon
              v-if="resource.instance"
              :instance="resource.instance"
              :tooltip="false"
              size="medium"
              class="mt-0.5"
            />
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <p class="font-medium truncate">
                  {{
                    resource.instance
                      ? instanceV1Name(resource.instance)
                      : resource.name
                  }}
                </p>
                <span
                  v-if="resource.instance?.environmentEntity"
                  class="text-xs px-1.5 py-0.5 rounded bg-gray-100 text-gray-600"
                >
                  {{ resource.instance.environmentEntity.title }}
                </span>
              </div>
              <div class="flex items-center gap-3 mt-1">
                <p class="text-sm text-control-light">
                  {{ resource.databases.length }} database{{
                    resource.databases.length === 1 ? "" : "s"
                  }}
                </p>
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
} from "lucide-vue-next";
import { computed, watch } from "vue";
import { InstanceV1EngineIcon } from "@/components/v2/Model/Instance";
import { useInstanceV1Store } from "@/store";
import { instanceV1Name } from "@/utils";
import { usePlanContext } from "../logic/context";
import { targetsForSpec } from "../logic/plan";
import DescriptionSection from "./DescriptionSection";

const { plan } = usePlanContext();
const instanceStore = useInstanceV1Store();

// Calculate statistics
const statistics = computed(() => {
  const specs = plan.value?.specs || [];
  let totalTargets = 0;
  const checkStatus = {
    total: 0,
    success: 0,
    warning: 0,
    error: 0,
  };

  // Count targets
  for (const spec of specs) {
    totalTargets += targetsForSpec(spec).length;
  }

  // Count check statuses
  const checkRuns = plan.value?.planCheckRunList || [];
  for (const checkRun of checkRuns) {
    checkStatus.total++;
    let hasError = false;
    let hasWarning = false;

    for (const result of checkRun.results) {
      if (result.status === "ERROR") {
        hasError = true;
      } else if (result.status === "WARNING") {
        hasWarning = true;
      }
    }

    if (hasError) {
      checkStatus.error++;
    } else if (hasWarning) {
      checkStatus.warning++;
    } else {
      checkStatus.success++;
    }
  }

  return {
    totalSpecs: specs.length,
    totalTargets,
    checkStatus,
  };
});

// Get affected resources
const affectedResources = computed(() => {
  const specs = plan.value?.specs || [];
  const resourceMap = new Map<
    string,
    { instanceName: string; databases: Set<string> }
  >();

  for (const spec of specs) {
    const targets = targetsForSpec(spec);
    for (const target of targets) {
      // Parse target format: instances/{instance}/databases/{database}
      const match = target.match(/instances\/([^/]+)\/databases\/([^/]+)/);
      if (match) {
        const instanceId = match[1];
        const database = match[2];
        const instanceName = `instances/${instanceId}`;

        if (!resourceMap.has(instanceName)) {
          resourceMap.set(instanceName, {
            instanceName,
            databases: new Set(),
          });
        }

        resourceMap.get(instanceName)!.databases.add(database);
      }
    }
  }

  return Array.from(resourceMap.values()).map((resource) => {
    const instance = instanceStore.getInstanceByName(resource.instanceName);
    return {
      name: resource.instanceName,
      instance: instance,
      databases: Array.from(resource.databases),
    };
  });
});

watch(
  () => affectedResources.value,
  () => {
    const instances = affectedResources.value.map((res) => res.name);
    for (const instanceName of instances) {
      instanceStore.getOrFetchInstanceByName(instanceName);
    }
  }
);
</script>
