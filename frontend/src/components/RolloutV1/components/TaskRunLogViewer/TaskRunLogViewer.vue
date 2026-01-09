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
        <span v-if="totalDuration" class="text-blue-500 tabular-nums">
          {{ totalDuration }}
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

    <!-- Multi-replica view: show replica groups when server restarts detected -->
    <template v-if="hasMultipleReplicas">
      <!-- Server restart notice -->
      <div
        class="flex items-center gap-x-2 px-3 py-2 bg-amber-50 border-b border-amber-200 text-amber-800"
      >
        <AlertTriangleIcon class="w-4 h-4 shrink-0" />
        <span>{{ $t("task-run.log-viewer.multiple-replicas-notice") }}</span>
      </div>

      <!-- Replica groups -->
      <div
        v-for="(replicaGroup, replicaIdx) in replicaGroups"
        :key="replicaGroup.replicaId"
        class="border-b border-gray-300 last:border-b-0"
      >
        <!-- Replica Header -->
        <div
          class="flex items-center gap-x-2 px-3 py-1.5 bg-gray-100 hover:bg-gray-200 cursor-pointer select-none"
          @click="toggleReplica(replicaGroup.replicaId)"
        >
          <component
            :is="isReplicaExpanded(replicaGroup.replicaId) ? ChevronDownIcon : ChevronRightIcon"
            class="w-3.5 h-3.5 text-gray-500 shrink-0"
          />
          <ServerIcon class="w-3.5 h-3.5 text-gray-500 shrink-0" />
          <span class="text-gray-700 font-medium">
            {{ $t("task-run.log-viewer.replica-n", { n: replicaIdx + 1 }) }}
          </span>
          <span class="text-gray-400 text-[10px] font-normal">
            {{ replicaGroup.replicaId.substring(0, 8) }}
          </span>
        </div>

        <!-- Sections within replica group -->
        <div v-if="isReplicaExpanded(replicaGroup.replicaId)">
          <!-- Orphan sections (before any release file marker) -->
          <div
            v-for="section in replicaGroup.sections"
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
            v-for="(fileGroup, fileIdx) in replicaGroup.releaseFileGroups"
            :key="`${replicaGroup.replicaId}-file-${fileIdx}`"
            class="border-b border-gray-200 last:border-b-0"
          >
            <!-- Release file header -->
            <div
              class="flex items-center gap-x-2 px-3 py-1.5 bg-blue-50 hover:bg-blue-100 cursor-pointer select-none ml-4"
              @click="toggleReleaseFile(`${replicaGroup.replicaId}-file-${fileIdx}`)"
            >
              <component
                :is="isReleaseFileExpanded(`${replicaGroup.replicaId}-file-${fileIdx}`) ? ChevronDownIcon : ChevronRightIcon"
                class="w-3.5 h-3.5 text-blue-500 shrink-0"
              />
              <FileCodeIcon class="w-3.5 h-3.5 text-blue-500 shrink-0" />
              <span class="text-blue-700 font-medium">
                {{ getReleaseFileLabel(fileGroup.version, fileGroup.filePath) }}
              </span>
            </div>

            <!-- Sections within release file group -->
            <div v-if="isReleaseFileExpanded(`${replicaGroup.replicaId}-file-${fileIdx}`)" class="ml-4">
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

    <!-- Single-replica view with release files -->
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

    <!-- Single-replica view: standard flat sections (no release files) -->
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
import SectionContent from "./SectionContent.vue";
import SectionHeader from "./SectionHeader.vue";
import { useTaskRunLogData } from "./useTaskRunLogData";
import { useTaskRunLogSections } from "./useTaskRunLogSections";

const props = defineProps<{
  taskRunName: string;
}>();

// Fetch task run log entries and sheets internally
const { entries, sheet, sheetsMap } = useTaskRunLogData(
  () => props.taskRunName
);

const {
  sections,
  hasMultipleReplicas,
  hasReleaseFiles,
  releaseFileGroups,
  replicaGroups,
  toggleSection,
  toggleReplica,
  toggleReleaseFile,
  isSectionExpanded,
  isReplicaExpanded,
  isReleaseFileExpanded,
  expandAll,
  collapseAll,
  areAllExpanded,
  totalSections,
  totalEntries,
  totalDuration,
} = useTaskRunLogSections(
  () => entries.value,
  () => sheet.value,
  () => sheetsMap.value
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
