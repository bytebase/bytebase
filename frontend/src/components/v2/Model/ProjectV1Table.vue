<template>
  <NDataTable
    key="project-table"
    size="small"
    v-bind="$attrs"
    :columns="columnList"
    :data="projectList"
    :striped="true"
    :bordered="bordered"
    :loading="loading"
    :row-key="(data: ComposedProject) => data.name"
    :row-props="rowProps"
    :paginate-single-page="false"
  />
</template>

<script lang="tsx" setup>
import { CheckIcon } from "lucide-vue-next";
import { NDataTable, type DataTableColumn } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import type { BBGridRow } from "@/bbkit";
import { useCurrentProject } from "@/components/Project/useCurrentProject";
import { useProjectSidebar } from "@/components/Project/useProjectSidebar";
import { ProjectNameCell } from "@/components/v2/Model/DatabaseV1Table/cells";
import { PROJECT_V1_ROUTE_DETAIL } from "@/router/dashboard/projectV1";
import { PROJECT_V1_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import { getProjectName } from "@/store/modules/v1/common";
import type { ComposedProject } from "@/types";
import type { Project } from "@/types/proto/v1/project_service";
import { extractProjectResourceName } from "@/utils";
import HighlightLabelText from "./HighlightLabelText.vue";

type ProjectDataTableColumn = DataTableColumn<ComposedProject> & {
  hide?: boolean;
};

export type ProjectGridRow = BBGridRow<Project>;

const props = withDefaults(
  defineProps<{
    projectList: ComposedProject[];
    currentProject?: ComposedProject;
    bordered?: boolean;
    loading?: boolean;
    keyword?: string;
    // If true, the default behavior of the row click event will be prevented.
    preventDefault?: boolean;
  }>(),
  {
    bordered: true,
    currentProject: undefined,
    keyword: undefined,
  }
);

const emit = defineEmits<{
  (event: "row-click", project: ComposedProject): void;
}>();

const { t } = useI18n();
const router = useRouter();

const { project } = useCurrentProject(
  computed(() => ({
    projectId: router.currentRoute.value.params.projectId as string,
  }))
);
const { activeSidebar } = useProjectSidebar(project);

const columnList = computed((): ProjectDataTableColumn[] => {
  return (
    [
      {
        key: "selection",
        width: 32,
        hide: !props.currentProject,
        render: (project) => {
          return (
            props.currentProject?.name === project.name && (
              <CheckIcon class="w-4 text-accent" />
            )
          );
        },
      },
      {
        key: "id",
        title: t("common.id"),
        width: 128,
        resizable: true,
        ellipsis: true,
        render: (project) => {
          return (
            <HighlightLabelText
              text={extractProjectResourceName(project.name)}
              keyword={props.keyword ?? ""}
            />
          );
        },
      },
      {
        key: "title",
        title: t("project.table.name"),
        render: (project) => (
          <ProjectNameCell
            mode="ALL_SHORT"
            project={project}
            keyword={props.keyword}
          />
        ),
      },
    ] as ProjectDataTableColumn[]
  ).filter((column) => !column.hide);
});

const rowProps = (project: ComposedProject) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      if (!props.preventDefault) {
        let routeName = PROJECT_V1_ROUTE_DETAIL;
        const currentRouteName = router.currentRoute.value.name?.toString();
        if (currentRouteName?.startsWith(PROJECT_V1_ROUTE_DASHBOARD)) {
          routeName = activeSidebar.value?.path ?? routeName;

          const { flattenNavigationItems } = useProjectSidebar(
            project,
            router.currentRoute.value
          );
          // Otherwise, redirect to the project detail page.
          if (
            !flattenNavigationItems.value.find(
              (item) => !item.hide && item.path === routeName
            )
          ) {
            routeName = PROJECT_V1_ROUTE_DETAIL;
          }
        }

        const route = router.resolve({
          name: routeName,
          params: {
            projectId: getProjectName(project.name),
          },
        });
        if (e.ctrlKey || e.metaKey) {
          window.open(route.fullPath, "_blank");
        } else {
          router.push(route);
        }
      }

      emit("row-click", project);
    },
  };
};
</script>
