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
    :row-key="(data: Project) => data.name"
    :checked-row-keys="shouldShowSelection ? selectedProjectNames : undefined"
    :row-props="rowProps"
    :paginate-single-page="false"
    @update:checked-row-keys="updateSelectedProjects"
    @update:sorter="$emit('update:sorters', $event)"
  />
</template>

<script lang="tsx" setup>
import { CheckIcon } from "lucide-vue-next";
import {
  type DataTableColumn,
  type DataTableSortState,
  NDataTable,
} from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { LabelsCell, ProjectNameCell } from "@/components/v2/Model/cells";
import { PROJECT_V1_ROUTE_DETAIL } from "@/router/dashboard/projectV1";
import { PROJECT_V1_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import { getProjectName } from "@/store/modules/v1/common";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { extractProjectResourceName, hasWorkspacePermissionV2 } from "@/utils";
import HighlightLabelText from "./HighlightLabelText.vue";
import { mapSorterStatus } from "./utils";

type ProjectDataTableColumn = DataTableColumn<Project> & {
  hide?: boolean;
};

const props = withDefaults(
  defineProps<{
    projectList: Project[];
    currentProject?: Project;
    bordered?: boolean;
    loading?: boolean;
    keyword?: string;
    // If true, the default behavior of the row click event will be prevented.
    preventDefault?: boolean;
    // Selected project names for batch operations
    selectedProjectNames?: string[];
    // Whether to show selection checkboxes
    showSelection?: boolean;
    // Whether to show labels column (hidden in dropdowns for cleaner UI)
    showLabels?: boolean;
    sorters?: DataTableSortState[];
  }>(),
  {
    bordered: true,
    currentProject: undefined,
    keyword: undefined,
    selectedProjectNames: () => [],
    showSelection: false,
    showLabels: true,
  }
);

const emit = defineEmits<{
  (event: "row-click", project: Project): void;
  (event: "update:selected-project-names", projectNames: string[]): void;
  (event: "update:sorters", sorters: DataTableSortState[]): void;
}>();

const { t } = useI18n();
const router = useRouter();

const hasDeletePermission = computed(() =>
  hasWorkspacePermissionV2("bb.projects.delete")
);

const shouldShowSelection = computed(
  () => props.showSelection && hasDeletePermission.value
);

const updateSelectedProjects = (checkedRowKeys: (string | number)[]) => {
  emit("update:selected-project-names", checkedRowKeys as string[]);
};

const columnList = computed((): ProjectDataTableColumn[] => {
  const columns: ProjectDataTableColumn[] = (
    [
      {
        key: "selection",
        type: shouldShowSelection.value ? "selection" : undefined,
        width: !shouldShowSelection.value ? 32 : undefined,
        hide: !shouldShowSelection.value && !props.currentProject,
        disabled: shouldShowSelection.value
          ? (project: Project) => {
              // Disable selection for default project
              return extractProjectResourceName(project.name) === "default";
            }
          : undefined,
        cellProps: shouldShowSelection.value
          ? () => {
              return {
                onClick: (e: MouseEvent) => {
                  e.stopPropagation();
                },
              };
            }
          : undefined,
        render: shouldShowSelection.value
          ? undefined
          : (project) => {
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
        ellipsis: {
          tooltip: true,
        },
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
        resizable: true,
        title: t("project.table.name"),
        render: (project) => (
          <ProjectNameCell project={project} keyword={props.keyword} />
        ),
      },
      {
        key: "labels",
        title: t("common.labels"),
        resizable: true,
        width: 300,
        hide: !props.showLabels,
        render: (project) => (
          <LabelsCell labels={project.labels} showCount={3} placeholder="-" />
        ),
      },
    ] as ProjectDataTableColumn[]
  ).filter((column) => !column.hide);
  return mapSorterStatus(columns, props.sorters);
});

const rowProps = (project: Project) => {
  return {
    style: "cursor: pointer;",
    onClick: (e: MouseEvent) => {
      if (!props.preventDefault) {
        const currentRouteName = router.currentRoute.value.name?.toString();
        let routeName = PROJECT_V1_ROUTE_DETAIL;

        if (currentRouteName?.startsWith(PROJECT_V1_ROUTE_DASHBOARD)) {
          routeName = PROJECT_V1_ROUTE_DETAIL;
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
