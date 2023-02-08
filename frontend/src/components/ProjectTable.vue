<template>
  <BBTable
    :column-list="columnList"
    :data-source="projectList"
    :show-header="true"
    :left-bordered="false"
    :right-bordered="false"
    @click-row="clickProject"
  >
    <template #header>
      <BBTableHeaderCell class="table-cell" :title="columnList[0].title" />
      <BBTableHeaderCell class="table-cell" :title="columnList[1].title" />
    </template>
    <template #body="{ rowData: project }">
      <BBTableCell :left-padding="4" class="table-cell text-gray-500 w-[30%]">
        <span class="flex flex-row items-center space-x-1">
          <span>{{ project.key }}</span>
          <div v-if="project.tenantMode === 'TENANT'" class="tooltip-wrapper">
            <TenantIcon class="ml-1 w-4 h-4 text-control" />
            <span class="tooltip whitespace-nowrap">
              {{ $t("project.mode.tenant") }}
            </span>
          </div>
          <div v-if="project.rowStatus === 'ARCHIVED'" class="tooltip-wrapper">
            <heroicons-outline:archive class="ml-1 w-4 h-4 text-control" />
            <span class="tooltip whitespace-nowrap">
              {{ $t("archive.archived") }}
            </span>
          </div>
          <div v-if="project.workflowType === 'VCS'" class="tooltip-wrapper">
            <heroicons-outline:collection class="ml-1 w-4 h-4 text-control" />
            <span class="tooltip whitespace-nowrap">
              {{ $t("database.gitops-enabled") }}
            </span>
          </div>
        </span>
      </BBTableCell>
      <BBTableCell class="truncate">
        {{ projectName(project) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType, computed, defineComponent } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { projectSlug } from "../utils";
import { Project } from "../types";
import TenantIcon from "./TenantIcon.vue";

export default defineComponent({
  name: "ProjectTable",
  components: {
    TenantIcon,
  },
  props: {
    projectList: {
      required: true,
      type: Object as PropType<Project[]>,
    },
  },

  setup(props) {
    const router = useRouter();
    const { t } = useI18n();
    const columnList = computed(() => [
      {
        title: t("project.table.key"),
      },
      {
        title: t("project.table.name"),
      },
    ]);

    const clickProject = function (
      section: number,
      row: number,
      e: MouseEvent
    ) {
      const project = props.projectList[row];
      const url = `/project/${projectSlug(project)}`;
      if (e.ctrlKey || e.metaKey) {
        window.open(url, "_blank");
      } else {
        router.push(url);
      }
    };

    return {
      columnList,
      clickProject,
    };
  },
});
</script>
