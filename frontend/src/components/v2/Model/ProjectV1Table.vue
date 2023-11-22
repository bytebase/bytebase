<template>
  <BBGrid
    :column-list="columnList"
    :data-source="projectList"
    class="border"
    :show-placeholder="true"
    @click-row="clickProject"
  >
    <template #item="{ item: project }: ProjectGridRow">
      <div class="bb-grid-cell text-gray-500">
        <span class="flex flex-row items-center space-x-1">
          <span>{{ project.key }}</span>
        </span>
      </div>
      <div class="bb-grid-cell truncate">
        <ProjectCol
          mode="ALL_SHORT"
          :project="project"
          :show-tenant-icon="true"
        />
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { PropType, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBGridColumn, BBGridRow, BBGrid } from "@/bbkit";
import { Project } from "@/types/proto/v1/project_service";
import { projectV1Slug } from "@/utils";

export type ProjectGridRow = BBGridRow<Project>;

defineProps({
  projectList: {
    required: true,
    type: Object as PropType<Project[]>,
  },
});

const router = useRouter();
const { t } = useI18n();
const columnList = computed((): BBGridColumn[] => [
  {
    title: t("project.table.key"),
    width: "minmax(auto, 25%)",
  },
  {
    title: t("project.table.name"),
    width: "1fr",
  },
]);

const clickProject = function (
  project: Project,
  section: number,
  row: number,
  e: MouseEvent
) {
  const url = `/project/${projectV1Slug(project)}`;
  if (e.ctrlKey || e.metaKey) {
    window.open(url, "_blank");
  } else {
    router.push(url);
  }
};
</script>
