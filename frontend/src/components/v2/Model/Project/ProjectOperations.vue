<template>
  <div
    v-bind="$attrs"
    class="text-sm flex flex-col lg:flex-row items-start lg:items-center bg-blue-100 py-3 px-4 text-main gap-y-2 gap-x-4 overflow-x-auto"
  >
    <span class="whitespace-nowrap">
      {{ $t("project.batch.selected", { count: projectList.length }) }}
    </span>
    <div class="flex items-center">
      <template v-for="action in actions" :key="action.text">
        <component :is="action.render()" v-if="action.render" />
        <NButton
          v-else
          quaternary
          size="small"
          type="primary"
          :disabled="action.disabled"
          @click="action.click"
        >
          <template v-if="action.icon" #icon>
            <component :is="action.icon" class="h-4 w-4" />
          </template>
          <span class="text-sm">{{ action.text }}</span>
        </NButton>
      </template>
    </div>
  </div>

  <BBAlert
    v-model:show="state.showArchiveConfirm"
    type="warning"
    :title="$t('project.batch.archive.title', { count: projectList.length })"
    :description="$t('project.batch.archive.description')"
    :ok-text="$t('common.archive')"
    @ok="handleBatchArchive"
    @cancel="state.showArchiveConfirm = false"
  >
    <!-- Force Archive Option -->
    <div class="flex flex-col gap-y-3">
      <NCheckbox v-model:checked="state.force">
        <div class="text-sm font-normal text-control-light">
          {{ $t("project.batch.force-archive-description") }}
        </div>
      </NCheckbox>

      <!-- Project List -->
      <div class="max-h-40 overflow-y-auto border rounded-sm p-2 bg-gray-50">
        <div class="text-xs font-medium text-gray-600 mb-2">
          {{ $t("project.batch.projects-to-archive") }}:
        </div>
        <div class="flex flex-col gap-y-1">
          <div
            v-for="project in projectList"
            :key="project.name"
            class="text-sm flex items-center gap-x-2"
          >
            <CheckIcon class="w-3 h-3 text-green-600" />
            <span>{{ project.title }}</span>
            <span class="text-gray-500"
              >({{ extractProjectResourceName(project.name) }})</span
            >
          </div>
        </div>
      </div>

      <!-- Warning for projects with resources -->
      <NAlert
        v-if="projectsWithResources.length > 0"
        type="warning"
        size="small"
      >
        <template #header>
          {{ $t("project.batch.resource-warning.title") }}
        </template>
        <div class="text-sm">
          {{ $t("project.batch.resource-warning.description") }}
          <ul class="mt-1 list-disc list-inside">
            <li v-for="project in projectsWithResources" :key="project.name">
              {{ project.title }} ({{
                extractProjectResourceName(project.name)
              }})
            </li>
          </ul>
        </div>
      </NAlert>
    </div>
  </BBAlert>
</template>

<script setup lang="tsx">
import { ArchiveIcon, CheckIcon } from "lucide-vue-next";
import { NAlert, NButton, NCheckbox } from "naive-ui";
import type { VNode } from "vue";
import { computed, h, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { BBAlert } from "@/bbkit";
import { pushNotification, useProjectV1Store } from "@/store";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { extractProjectResourceName, hasWorkspacePermissionV2 } from "@/utils";

interface Action {
  icon?: VNode;
  render?: () => VNode;
  text: string;
  disabled?: boolean;
  click?: () => void;
}

interface LocalState {
  loading: boolean;
  showArchiveConfirm: boolean;
  force: boolean;
}

const props = defineProps<{
  projectList: Project[];
}>();

const emit = defineEmits<{
  (event: "update"): void;
}>();

const { t } = useI18n();
const projectStore = useProjectV1Store();
const state = reactive<LocalState>({
  loading: false,
  showArchiveConfirm: false,
  force: false,
});

// For now, we'll assume no projects have resources - this would need to be enhanced
// to actually check for databases and open issues
const projectsWithResources = computed((): Project[] => {
  // TODO: Implement actual resource checking
  return [];
});

const actions = computed((): Action[] => {
  const list: Action[] = [];

  if (hasWorkspacePermissionV2("bb.projects.delete")) {
    list.push({
      icon: h(ArchiveIcon),
      text: t("common.archive"),
      disabled: props.projectList.length === 0,
      click: () => (state.showArchiveConfirm = true),
    });
  }

  // Add more batch operations here in the future
  // For example: batch update settings, batch export, etc.

  return list;
});

const handleBatchArchive = async () => {
  try {
    state.loading = true;

    // Use the batch delete API
    await projectStore.batchDeleteProjects(
      props.projectList.map((p) => p.name),
      state.force
    );

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("project.batch.archive.success", {
        count: props.projectList.length,
      }),
    });

    state.showArchiveConfirm = false;
    state.force = false;
    emit("update");
  } catch (error: unknown) {
    const err = error as { message?: string };
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("project.batch.archive.error"),
      description: err.message,
    });
  } finally {
    state.loading = false;
  }
};
</script>
