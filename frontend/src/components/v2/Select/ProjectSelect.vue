<template>
  <RemoteResourceSelector
    v-bind="$attrs"
    :multiple="multiple"
    :disabled="disabled"
    :value="projectName"
    :values="projectNames"
    :custom-label="renderLabel"
    class="bb-project-select"
    :additional-data="additionalData"
    :search="handleSearch"
    :get-option="getOption"
    :filter="filterProject"
    @update:value="(val) => $emit('update:project-name', val)"
    @update:values="(val) => $emit('update:project-names', val)"
  >
    <template v-if="$slots.empty" #empty>
      <slot name="empty" />
    </template>
  </RemoteResourceSelector>
</template>

<script lang="tsx" setup>
import { computedAsync } from "@vueuse/core";
import { intersection } from "lodash-es";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { ProjectNameCell } from "@/components/v2/Model/cells";
import { usePermissionStore, useProjectV1Store } from "@/store";
import {
  DEFAULT_PROJECT_NAME,
  isValidProjectName,
  UNKNOWN_PROJECT_NAME,
  unknownProject,
} from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import RemoteResourceSelector from "./RemoteResourceSelector.vue";

const props = withDefaults(
  defineProps<{
    disabled?: boolean;
    projectName?: string | undefined | null; // UNKNOWN_PROJECT_NAME to "ALL"
    projectNames?: string[] | undefined | null;
    allowedProjectRoleList?: string[]; // Empty array([]) to "ALL"
    includeAll?: boolean;
    includeDefaultProject?: boolean;
    multiple?: boolean;
    renderSuffix?: (project: string) => string;
    filter?: (project: Project) => boolean;
  }>(),
  {
    disabled: false,
    projectName: undefined,
    projectNames: undefined,
    allowedProjectRoleList: () => [],
    includeAll: false,
    includeDefaultProject: false,
    multiple: false,
    filter: () => true,
    renderSuffix: () => "",
  }
);

const emit = defineEmits<{
  (event: "update:project-name", name: string | undefined): void;
  (event: "update:project-names", names: string[]): void;
}>();

const { t } = useI18n();
const permissionStore = usePermissionStore();
const projectStore = useProjectV1Store();

const hasWorkspaceManageProjectPermission = computed(() =>
  hasWorkspacePermissionV2("bb.projects.list")
);

const filterProject = (project: Project) => {
  if (
    !hasWorkspaceManageProjectPermission.value &&
    props.allowedProjectRoleList.length > 0
  ) {
    const roles = permissionStore.currentRoleListInProjectV1(project);
    if (intersection(props.allowedProjectRoleList, roles).length == 0) {
      return false;
    }
  }

  if (props.filter) {
    return props.filter(project);
  }

  return true;
};

const additionalData = computedAsync(async () => {
  const data = [];

  if (props.projectName === UNKNOWN_PROJECT_NAME || props.includeAll) {
    const dummyAll = {
      ...unknownProject(),
      title: t("project.all"),
    };
    data.unshift(dummyAll);
  }

  let projectNames: string[] = [];
  if (props.projectName) {
    projectNames = [props.projectName];
  } else if (props.projectNames) {
    projectNames = props.projectNames;
  }

  for (const projectName of projectNames) {
    if (isValidProjectName(projectName)) {
      const project = await projectStore.getOrFetchProjectByName(projectName);
      data.push(project);
    }
  }

  return data;
}, []);

const handleSearch = async (params: {
  search: string;
  pageToken: string;
  pageSize: number;
}) => {
  const { projects, nextPageToken } = await projectStore.fetchProjectList({
    filter: {
      query: params.search,
      excludeDefault: !props.includeDefaultProject,
    },
    pageToken: params.pageToken,
    pageSize: params.pageSize,
  });

  return {
    nextPageToken,
    data: projects,
  };
};

const getOption = (project: Project) => ({
  value: project.name,
  label:
    project.name === DEFAULT_PROJECT_NAME
      ? t("common.unassigned")
      : project.name === UNKNOWN_PROJECT_NAME
        ? t("project.all")
        : project.title,
});

const renderLabel = (project: Project) => {
  if (!project) return null;
  return (
    <ProjectNameCell
      project={project}
      mode="ALL_SHORT"
      suffix={props.renderSuffix(project.name)}
    >
      {{
        suffix: () => (
          <span class="opacity-60">{props.renderSuffix(project.name)}</span>
        ),
      }}
    </ProjectNameCell>
  );
};
</script>
