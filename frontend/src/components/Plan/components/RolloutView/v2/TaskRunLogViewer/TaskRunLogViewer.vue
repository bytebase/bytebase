<template>
  <div
    v-if="sections.length > 0"
    class="w-full font-mono text-xs bg-gray-50 border border-gray-200 overflow-hidden rounded"
  >
    <!-- Multi-deploy view: show deploy groups when server restarts detected -->
    <template v-if="hasMultipleDeploys">
      <!-- Server restart notice -->
      <div
        class="flex items-center gap-x-2 px-3 py-2 bg-amber-50 border-b border-amber-200 text-amber-800"
      >
        <AlertTriangleIcon class="w-4 h-4 shrink-0" />
        <span>{{ $t("task-run.log-viewer.multiple-deploys-notice") }}</span>
      </div>

      <!-- Deploy groups -->
      <div
        v-for="(deployGroup, deployIdx) in deployGroups"
        :key="deployGroup.deployId"
        class="border-b border-gray-300 last:border-b-0"
      >
        <!-- Deploy Header -->
        <div
          class="flex items-center gap-x-2 px-3 py-1.5 bg-gray-100 hover:bg-gray-200 cursor-pointer select-none"
          @click="toggleDeploy(deployGroup.deployId)"
        >
          <component
            :is="isDeployExpanded(deployGroup.deployId) ? ChevronDownIcon : ChevronRightIcon"
            class="w-3.5 h-3.5 text-gray-500 shrink-0"
          />
          <ServerIcon class="w-3.5 h-3.5 text-gray-500 shrink-0" />
          <span class="text-gray-700 font-medium">
            {{ $t("task-run.log-viewer.deployment-n", { n: deployIdx + 1 }) }}
          </span>
          <span class="text-gray-400 text-[10px] font-normal">
            {{ deployGroup.deployId.substring(0, 8) }}
          </span>
        </div>

        <!-- Sections within deploy group -->
        <div v-if="isDeployExpanded(deployGroup.deployId)">
          <div
            v-for="section in deployGroup.sections"
            :key="section.id"
            class="border-b border-gray-200 last:border-b-0"
          >
            <SectionHeader
              :section="section"
              :is-expanded="isSectionExpanded(section.id)"
              :indent="true"
              @toggle="toggleSection(section.id)"
            />
            <SectionContent
              v-if="isSectionExpanded(section.id)"
              :section="section"
              :indent="true"
            />
          </div>
        </div>
      </div>
    </template>

    <!-- Single-deploy view: standard flat sections -->
    <template v-else>
      <div
        v-for="section in sections"
        :key="section.id"
        class="border-b border-gray-200 last:border-b-0"
      >
        <SectionHeader
          :section="section"
          :is-expanded="isSectionExpanded(section.id)"
          @toggle="toggleSection(section.id)"
        />
        <SectionContent
          v-if="isSectionExpanded(section.id)"
          :section="section"
        />
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import {
  AlertTriangleIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  ServerIcon,
} from "lucide-vue-next";
import type { TaskRunLogEntry } from "@/types/proto-es/v1/rollout_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import SectionContent from "./SectionContent.vue";
import SectionHeader from "./SectionHeader.vue";
import { useTaskRunLogSections } from "./useTaskRunLogSections";

const props = defineProps<{
  entries: TaskRunLogEntry[];
  sheet?: Sheet;
}>();

const {
  sections,
  hasMultipleDeploys,
  deployGroups,
  toggleSection,
  toggleDeploy,
  isSectionExpanded,
  isDeployExpanded,
} = useTaskRunLogSections(
  () => props.entries,
  () => props.sheet
);
</script>
