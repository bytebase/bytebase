<template>
  <NSelect
    v-bind="$attrs"
    :value="combinedValue"
    :options="options"
    :placeholder="$t('project.select')"
    :filterable="true"
    :multiple="multiple"
    :filter="filterByName"
    :disabled="disabled"
    :render-label="renderLabel"
    class="bb-project-select"
    style="width: 12rem"
    @update:value="handleValueUpdated"
  >
    <template v-if="$slots.empty" #empty>
      <slot name="empty" />
    </template>
  </NSelect>
</template>

<script lang="tsx" setup>
import { intersection } from "lodash-es";
import type { SelectOption } from "naive-ui";
import { NSelect } from "naive-ui";
import { computed } from "vue";
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
import { Workflow } from "@/types/proto/v1/project_service";
import { extractProjectResourceName, hasWorkspacePermissionV2 } from "@/utils";

interface ProjectSelectOption extends SelectOption {
  value: string;
  project: Project;
}

const props = withDefaults(
  defineProps<{
    disabled?: boolean;
    projectName?: string | undefined | null; // UNKNOWN_PROJECT_NAME to "ALL"
    projectNames?: string[] | undefined | null;
    allowedProjectRoleList?: string[]; // Empty array([]) to "ALL"
    allowedProjectWorkflowTypeList?: Workflow[];
    includeAll?: boolean;
    includeDefaultProject?: boolean;
    includeArchived?: boolean;
    multiple?: boolean;
    renderSuffix?: (project: string) => string;
    filter?: (project: ComposedProject, index: number) => boolean;
  }>(),
  {
    disabled: false,
    projectName: undefined,
    projectNames: undefined,
    allowedProjectRoleList: () => [],
    allowedProjectWorkflowTypeList: () => [Workflow.UI, Workflow.VCS],
    includeAll: false,
    includeDefaultProject: false,
    includeArchived: false,
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
const { projectList } = useProjectV1List(true /* showDeleted */);

const combinedValue = computed(() => {
  if (props.multiple) {
    return props.projectNames || [];
  } else {
    return props.projectName;
  }
});

const handleValueUpdated = (value: string | string[]) => {
  if (props.multiple) {
    if (!value) {
      // normalize value
      value = [];
    }
    emit("update:project-names", value as string[]);
  } else {
    if (value === null) {
      // normalize value
      value = "";
    }
    emit("update:project-name", value as string);
  }
};

const hasWorkspaceManageProjectPermission = computed(() =>
  hasWorkspacePermissionV2("bb.projects.list")
);

const rawProjectList = computed(() => {
  return projectList.value.filter((project) => {
    if (project.name === DEFAULT_PROJECT_NAME) {
      return false;
    }
    return props.allowedProjectWorkflowTypeList.includes(project.workflow);
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
  return combinedProjectList.value.map<ProjectSelectOption>((project) => {
    return {
      project,
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

const filterByName = (pattern: string, option: SelectOption) => {
  const { project } = option as ProjectSelectOption;
  pattern = pattern.toLowerCase();
  return (
    project.name.toLowerCase().includes(pattern) ||
    project.key.toLowerCase().includes(pattern)
  );
};

const renderLabel = (option: SelectOption) => {
  const { project } = option as ProjectSelectOption;
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
