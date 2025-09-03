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
  />
</template>

<script lang="tsx" setup>
import { CheckIcon } from "lucide-vue-next";
import { NDataTable, type DataTableColumn } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { ProjectNameCell } from "@/components/v2/Model/DatabaseV1Table/cells";
import { PROJECT_V1_ROUTE_DETAIL } from "@/router/dashboard/projectV1";
import { PROJECT_V1_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import { getProjectName } from "@/store/modules/v1/common";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { extractProjectResourceName, hasWorkspacePermissionV2 } from "@/utils";
import HighlightLabelText from "./HighlightLabelText.vue";

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
  }>(),
  {
    bordered: true,
    currentProject: undefined,
    keyword: undefined,
    selectedProjectNames: () => [],
    showSelection: false,
  }
);

const emit = defineEmits<{
  (event: "row-click", project: Project): void;
  (event: "update:selected-project-names", projectNames: string[]): void;
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
  return (
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
