<template>
  <div
    v-bind="$attrs"
    class="text-sm flex flex-col lg:flex-row items-start lg:items-center bg-blue-100 py-3 px-4 text-main gap-y-2 gap-x-4 overflow-x-auto"
  >
    <span class="whitespace-nowrap">
      {{ $t("project.batch.selected", { count: projectList.length }) }}
    </span>
    <div class="flex items-center gap-x-2">
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
    <!-- Project List -->
    <div class="flex flex-col gap-y-3">
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

  <BBAlert
    v-model:show="state.showDeleteConfirm"
    type="error"
    :title="$t('project.batch.delete.title', { count: projectList.length })"
    :description="$t('project.batch.delete.description')"
    :ok-text="$t('common.delete')"
    @ok="handleBatchDelete"
    @cancel="state.showDeleteConfirm = false"
  >
    <!-- Project List -->
    <div class="flex flex-col gap-y-3">
      <div class="max-h-40 overflow-y-auto border rounded-sm p-2 bg-gray-50">
        <div class="text-xs font-medium text-gray-600 mb-2">
          {{ $t("project.batch.projects-to-delete") }}:
        </div>
        <div class="flex flex-col gap-y-1">
          <div
            v-for="project in projectList"
            :key="project.name"
            class="text-sm flex items-center gap-x-2"
          >
            <CheckIcon class="w-3 h-3 text-red-600" />
            <span>{{ project.title }}</span>
            <span class="text-gray-500"
              >({{ extractProjectResourceName(project.name) }})</span
            >
          </div>
        </div>
      </div>

      <!-- Warning about permanent deletion -->
      <NAlert type="error" size="small">
        <template #header>
          {{ $t("common.cannot-undo-this-action") }}
        </template>
        <div class="text-sm">
          {{ $t("project.batch.delete.warning") }}
        </div>
      </NAlert>
    </div>
  </BBAlert>
</template>

<script setup lang="tsx">
import { ArchiveIcon, CheckIcon, Trash2Icon } from "lucide-vue-next";
import { NAlert, NButton } from "naive-ui";
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
  showDeleteConfirm: boolean;
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
  showDeleteConfirm: false,
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
    list.push({
      icon: h(Trash2Icon),
      text: t("common.delete"),
      disabled: props.projectList.length === 0,
      click: () => (state.showDeleteConfirm = true),
    });
  }

  return list;
});

const handleBatchArchive = async () => {
  try {
    state.loading = true;

    // Use the batch delete API
    await projectStore.batchDeleteProjects(
      props.projectList.map((p) => p.name)
    );

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("project.batch.archive.success", {
        count: props.projectList.length,
      }),
    });

    state.showArchiveConfirm = false;
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

const handleBatchDelete = async () => {
  try {
    state.loading = true;

    // Use the batch purge API
    await projectStore.batchPurgeProjects(props.projectList.map((p) => p.name));

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("project.batch.delete.success", {
        count: props.projectList.length,
      }),
    });

    state.showDeleteConfirm = false;
    emit("update");
  } catch (error: unknown) {
    const err = error as { message?: string };
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("project.batch.delete.error"),
      description: err.message,
    });
  } finally {
    state.loading = false;
  }
};
</script>
