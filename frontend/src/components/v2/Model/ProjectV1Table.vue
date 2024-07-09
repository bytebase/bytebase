<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="projectList"
    :striped="true"
    :bordered="bordered"
    :loading="loading"
    :row-key="(data: ComposedProject) => data.name"
    :row-props="rowProps"
    :pagination="pagination"
    :paginate-single-page="false"
  />
</template>

<script lang="tsx" setup>
import { CheckIcon } from "lucide-vue-next";
import {
  NDataTable,
  type DataTableColumn,
  type PaginationProps,
} from "naive-ui";
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
    pagination?: false | PaginationProps;
    onClick?: (project: ComposedProject, e: MouseEvent) => void;
  }>(),
  {
    bordered: true,
    currentProject: undefined,
    pagination: () => ({ pageSize: 20 }) as PaginationProps,
    onClick: undefined,
  }
);

const emit = defineEmits<{
  (event: "row-click"): void;
}>();

const router = useRouter();
const { t } = useI18n();

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
        width: 50,
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
        key: "key",
        title: t("project.table.key"),
        width: 200,
        render: (project) => project.key,
      },
      {
        key: "title",
        title: t("project.table.name"),
        render: (project) => (
          <ProjectNameCell mode="ALL_SHORT" project={project} />
        ),
      },
    ] as ProjectDataTableColumn[]
  ).filter((column) => !column.hide);
});

const rowProps = (project: ComposedProject) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      if (props.onClick) {
        props.onClick(project, e);
        return;
      }

      let routeName = PROJECT_V1_ROUTE_DETAIL;
      const currentRouteName = router.currentRoute.value.name?.toString();
      if (currentRouteName?.startsWith(PROJECT_V1_ROUTE_DASHBOARD)) {
        routeName = activeSidebar.value?.path ?? routeName;

        const { flattenNavigationItems } = useProjectSidebar(
          project,
          router.currentRoute.value
        );
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
      emit("row-click");
    },
  };
};
</script>
