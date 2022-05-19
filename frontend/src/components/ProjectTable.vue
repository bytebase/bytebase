<template>
  <BBTable
    :column-list="COLUMN_LIST"
    :data-source="projectList"
    :show-header="true"
    :left-bordered="false"
    :right-bordered="false"
    @click-row="clickProject"
  >
    <template #header>
      <BBTableHeaderCell
        class="w-4 table-cell"
        :title="state.columnList[0].title"
      />
      <BBTableHeaderCell
        class="w-24 table-cell"
        :title="state.columnList[1].title"
      />
      <BBTableHeaderCell
        class="w-8 table-cell"
        :title="state.columnList[2].title"
      />
    </template>
    <template #body="{ rowData: project }">
      <BBTableCell :left-padding="4" class="table-cell text-gray-500">
        <span class="flex flex-row items-center space-x-1">
          <span>{{ project.key }}</span>
          <div v-if="project.rowStatus === 'ARCHIVED'" class="tooltip-wrapper">
            <heroicons-outline:archive class="ml-1 w-4 h-4 text-control" />
            <span class="tooltip whitespace-nowrap">
              {{ $t("archive.archived") }}
            </span>
          </div>
          <div v-if="project.workflowType === 'VCS'" class="tooltip-wrapper">
            <heroicons-outline:collection class="ml-1 w-4 h-4 text-control" />
            <span class="tooltip whitespace-nowrap">
              {{ $t("database.version-control-enabled") }}
            </span>
          </div>
        </span>
      </BBTableCell>
      <BBTableCell class="truncate">
        {{ projectName(project) }}
      </BBTableCell>
      <BBTableCell class="hidden md:table-cell">
        {{ humanizeTs(project.createdTs) }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType, computed, defineComponent } from "vue";
import { useRouter } from "vue-router";
import { projectSlug } from "../utils";
import { Project } from "../types";
import { useI18n } from "vue-i18n";

export default defineComponent({
  name: "ProjectTable",
  components: {},
  props: {
    projectList: {
      required: true,
      type: Object as PropType<Project[]>,
    },
  },

  setup(props) {
    const router = useRouter();
    const { t } = useI18n();
    const COLUMN_LIST = computed(() => [
      {
        title: t("project.table.key"),
      },
      {
        title: t("project.table.name"),
      },
      {
        title: t("project.table.created-at"),
      },
    ]);

    const clickProject = function (section: number, row: number) {
      const project = props.projectList[row];
      router.push(`/project/${projectSlug(project)}`);
    };

    return {
      COLUMN_LIST,
      clickProject,
    };
  },
});
</script>
