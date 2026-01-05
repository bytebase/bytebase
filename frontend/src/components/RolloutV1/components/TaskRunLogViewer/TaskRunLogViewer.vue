<template>
  <div
    v-if="sections.length > 0"
    class="w-full font-mono text-xs bg-gray-50 border border-gray-200 overflow-hidden rounded"
  >
    <!-- Toolbar with summary stats and expand/collapse buttons -->
    <div
      class="flex items-center justify-between px-2 py-1 bg-gray-100 border-b border-gray-200"
    >
      <!-- Left: Summary stats -->
      <div class="flex items-center gap-x-2 text-gray-500">
        <ListIcon class="w-3.5 h-3.5" />
        <span>
          {{
            $t("task-run.log-viewer.summary", {
              sections: totalSections,
              entries: totalEntries,
            })
          }}
        </span>
      </div>

      <!-- Right: Expand/Collapse toggle button -->
      <button
        class="flex items-center gap-x-1 px-1.5 py-0.5 text-gray-600 hover:text-gray-900 hover:bg-gray-200 rounded transition-colors"
        @click="toggleExpandAll"
      >
        <component
          :is="areAllExpanded ? ChevronsDownUpIcon : ChevronsUpDownIcon"
          class="w-3.5 h-3.5"
        />
        <span>{{
          areAllExpanded
            ? $t("task-run.log-viewer.collapse-all")
            : $t("task-run.log-viewer.expand-all")
        }}</span>
      </button>
    </div>

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
          <!-- Orphan sections (before any release file marker) -->
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

          <!-- Release file groups -->
          <div
            v-for="(fileGroup, fileIdx) in deployGroup.releaseFileGroups"
            :key="`${deployGroup.deployId}-file-${fileIdx}`"
            class="border-b border-gray-200 last:border-b-0"
          >
            <!-- Release file header -->
            <div
              class="flex items-center gap-x-2 px-3 py-1.5 bg-blue-50 hover:bg-blue-100 cursor-pointer select-none ml-4"
              @click="toggleReleaseFile(`${deployGroup.deployId}-file-${fileIdx}`)"
            >
              <component
                :is="isReleaseFileExpanded(`${deployGroup.deployId}-file-${fileIdx}`) ? ChevronDownIcon : ChevronRightIcon"
                class="w-3.5 h-3.5 text-blue-500 shrink-0"
              />
              <FileCodeIcon class="w-3.5 h-3.5 text-blue-500 shrink-0" />
              <span class="text-blue-700 font-medium">
                {{ getReleaseFileLabel(fileGroup.version, fileGroup.filePath) }}
              </span>
            </div>

            <!-- Sections within release file group -->
            <div v-if="isReleaseFileExpanded(`${deployGroup.deployId}-file-${fileIdx}`)" class="ml-4">
              <div
                v-for="section in fileGroup.sections"
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
        </div>
      </div>
    </template>

    <!-- Single-deploy view with release files -->
    <template v-else-if="hasReleaseFiles">
      <div
        v-for="(fileGroup, fileIdx) in releaseFileGroups"
        :key="`file-${fileIdx}`"
        class="border-b border-gray-200 last:border-b-0"
      >
        <!-- Release file header -->
        <div
          class="flex items-center gap-x-2 px-3 py-1.5 bg-blue-50 hover:bg-blue-100 cursor-pointer select-none"
          @click="toggleReleaseFile(`file-${fileIdx}`)"
        >
          <component
            :is="isReleaseFileExpanded(`file-${fileIdx}`) ? ChevronDownIcon : ChevronRightIcon"
            class="w-3.5 h-3.5 text-blue-500 shrink-0"
          />
          <FileCodeIcon class="w-3.5 h-3.5 text-blue-500 shrink-0" />
          <span class="text-blue-700 font-medium">
            {{ getReleaseFileLabel(fileGroup.version, fileGroup.filePath) }}
          </span>
        </div>

        <!-- Sections within release file group -->
        <div v-if="isReleaseFileExpanded(`file-${fileIdx}`)">
          <div
            v-for="section in fileGroup.sections"
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

    <!-- Single-deploy view: standard flat sections (no release files) -->
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
  ChevronsDownUpIcon,
  ChevronsUpDownIcon,
  FileCodeIcon,
  ListIcon,
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
  hasReleaseFiles,
  releaseFileGroups,
  deployGroups,
  toggleSection,
  toggleDeploy,
  toggleReleaseFile,
  isSectionExpanded,
  isDeployExpanded,
  isReleaseFileExpanded,
  expandAll,
  collapseAll,
  areAllExpanded,
  totalSections,
  totalEntries,
} = useTaskRunLogSections(
  () => props.entries,
  () => props.sheet
);

const toggleExpandAll = () => {
  if (areAllExpanded.value) {
    collapseAll();
  } else {
    expandAll();
  }
};

const getReleaseFileLabel = (version: string, filePath: string): string => {
  if (filePath) {
    return `${version}: ${filePath}`;
  }
  return version;
};
</script>
