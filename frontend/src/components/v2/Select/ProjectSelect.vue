<template>
  <ResourceSelect
    v-bind="$attrs"
    :placeholder="$t('project.select')"
    :multiple="multiple"
    :disabled="disabled"
    :value="projectName"
    :values="projectNames"
    :options="options"
    :custom-label="renderLabel"
    class="bb-project-select"
    @update:value="(val) => $emit('update:project-name', val)"
    @update:values="(val) => $emit('update:project-names', val)"
  >
    <template v-if="$slots.empty" #empty>
      <slot name="empty" />
    </template>
  </ResourceSelect>
</template>

<script lang="tsx" setup>
import { intersection } from "lodash-es";
import { computed, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { ProjectNameCell } from "@/components/v2/Model/DatabaseV1Table/cells";
import { useProjectV1List, usePermissionStore } from "@/store";
import type { ComposedProject } from "@/types";
import {
  unknownProject,
  defaultProject,
  DEFAULT_PROJECT_NAME,
  UNKNOWN_PROJECT_NAME,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import type { Project } from "@/types/proto/v1/project_service";
import { extractProjectResourceName, hasWorkspacePermissionV2 } from "@/utils";
import ResourceSelect from "./ResourceSelect.vue";

const props = withDefaults(
  defineProps<{
    disabled?: boolean;
    projectName?: string | undefined | null; // UNKNOWN_PROJECT_NAME to "ALL"
    projectNames?: string[] | undefined | null;
    allowedProjectRoleList?: string[]; // Empty array([]) to "ALL"
    includeAll?: boolean;
    includeDefaultProject?: boolean;
    includeArchived?: boolean;
    multiple?: boolean;
    renderSuffix?: (project: string) => string;
    filter?: (project: ComposedProject, index: number) => boolean;
    defaultSelectFirst?: boolean;
  }>(),
  {
    disabled: false,
    projectName: undefined,
    projectNames: undefined,
    allowedProjectRoleList: () => [],
    includeAll: false,
    includeDefaultProject: false,
    includeArchived: false,
    multiple: false,
    filter: () => true,
    renderSuffix: () => "",
    defaultSelectFirst: false,
  }
);

const emit = defineEmits<{
  (event: "update:project-name", name: string | undefined): void;
  (event: "update:project-names", names: string[]): void;
}>();

const { t } = useI18n();
const permissionStore = usePermissionStore();
const { projectList } = useProjectV1List(true /* showDeleted */);

const hasWorkspaceManageProjectPermission = computed(() =>
  hasWorkspacePermissionV2("bb.projects.list")
);

const rawProjectList = computed(() => {
  return projectList.value.filter((project) => {
    if (project.name === DEFAULT_PROJECT_NAME) {
      return false;
    }
    return true;
  });
});

const isOrphanValue = computed(() => {
  if (props.projectName === undefined) return false;

  return !rawProjectList.value.find((proj) => proj.name === props.projectName);
});

const combinedProjectList = computed(() => {
  let list = rawProjectList.value.filter((project) => {
    if (props.includeArchived) return true;
    if (project.state === State.ACTIVE) return true;
    // ARCHIVED
    if (project.name === props.projectName) return true;
    return false;
  });

  // If the current user is not workspace admin/DBA, filter the project list by the given role list.
  if (
    !hasWorkspaceManageProjectPermission.value &&
    props.allowedProjectRoleList.length > 0
  ) {
    list = list.filter((project) => {
      const roles = permissionStore.currentRoleListInProjectV1(project);
      return intersection(props.allowedProjectRoleList, roles).length > 0;
    });
  }

  if (props.filter) {
    list = list.filter(props.filter);
  }

  if (
    props.projectName &&
    props.projectName !== DEFAULT_PROJECT_NAME &&
    props.projectName !== UNKNOWN_PROJECT_NAME &&
    isOrphanValue.value
  ) {
    // It may happen the selected id might not be in the project list.
    // e.g. the selected project is deleted after the selection and we
    // are unable to cleanup properly. In such case, the selected project id
    // is orphaned and we just display the id
    const dummyProject = {
      ...unknownProject(),
      name: props.projectName,
      title: extractProjectResourceName(props.projectName),
    };
    list.unshift(dummyProject);
  }

  if (
    props.projectName === DEFAULT_PROJECT_NAME ||
    props.includeDefaultProject
  ) {
    list.unshift({ ...defaultProject() });
  }

  if (props.projectName === UNKNOWN_PROJECT_NAME || props.includeAll) {
    const dummyAll = {
      ...unknownProject(),
      title: t("project.all"),
    };
    list.unshift(dummyAll);
  }

  return list;
});

const options = computed(() => {
  return combinedProjectList.value.map((project) => {
    return {
      resource: project,
      value: project.name,
      label:
        project.name === DEFAULT_PROJECT_NAME
          ? t("common.unassigned")
          : project.name === UNKNOWN_PROJECT_NAME
            ? t("project.all")
            : project.title,
    };
  });
});

watchEffect(() => {
  if (!props.defaultSelectFirst || props.projectName || props.multiple) {
    return;
  }
  if (options.value.length === 0) {
    return;
  }

  emit("update:project-name", options.value[0].value);
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
