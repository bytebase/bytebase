<template>
  <NSelect
    v-bind="$attrs"
    :value="value"
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
  />
</template>

<script lang="tsx" setup>
import { intersection } from "lodash-es";
import type { SelectOption } from "naive-ui";
import { NSelect } from "naive-ui";
import { computed, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { ProjectNameCell } from "@/components/v2/Model/DatabaseV1Table/cells";
import {
  useCurrentUserV1,
  useProjectV1Store,
  useProjectV1List,
  usePermissionStore,
} from "@/store";
import type { ComposedProject } from "@/types";
import {
  DEFAULT_PROJECT_ID,
  UNKNOWN_ID,
  unknownProject,
  defaultProject,
} from "@/types";
import { State } from "@/types/proto/v1/common";
import type { Project } from "@/types/proto/v1/project_service";
import { Workflow } from "@/types/proto/v1/project_service";
import { hasWorkspacePermissionV2 } from "@/utils";

interface ProjectSelectOption extends SelectOption {
  value: string;
  project: Project;
}

const props = withDefaults(
  defineProps<{
    disabled?: boolean;
    project?: string | undefined | null; // UNKNOWN_ID(-1) to "ALL"
    projects?: string[] | undefined | null; // UNKNOWN_ID(-1) to "ALL"
    allowedProjectRoleList?: string[]; // Empty array([]) to "ALL"
    allowedProjectWorkflowTypeList?: Workflow[];
    includeAll?: boolean;
    includeDefaultProject?: boolean;
    includeArchived?: boolean;
    useResourceId?: boolean;
    multiple?: boolean;
    renderSuffix?: (project: string) => string;
    filter?: (project: ComposedProject, index: number) => boolean;
  }>(),
  {
    disabled: false,
    project: undefined,
    projects: undefined,
    allowedProjectRoleList: () => [],
    allowedProjectWorkflowTypeList: () => [Workflow.UI, Workflow.VCS],
    includeAll: false,
    includeDefaultProject: false,
    includeArchived: false,
    useResourceId: false,
    multiple: false,
    filter: () => true,
    renderSuffix: (project: string) => "",
  }
);

const emit = defineEmits<{
  (event: "update:project", id: string | undefined): void;
  (event: "update:projects", id: string[]): void;
}>();

const { t } = useI18n();
const currentUserV1 = useCurrentUserV1();
const projectV1Store = useProjectV1Store();
const permissionStore = usePermissionStore();

const prepare = () => {
  projectV1Store.fetchProjectList(true /* showDeleted */);
};

const value = computed(() => {
  if (props.multiple) {
    return props.projects || [];
  } else {
    return props.project;
  }
});

const handleValueUpdated = (value: string | string[]) => {
  if (props.multiple) {
    if (!value) {
      // normalize value
      value = [];
    }
    emit("update:projects", value as string[]);
  } else {
    if (value === null) {
      // normalize value
      value = "";
    }
    emit("update:project", value as string);
  }
};

const hasWorkspaceManageProjectPermission = computed(() =>
  hasWorkspacePermissionV2(currentUserV1.value, "bb.projects.list")
);

const { projectList } = useProjectV1List();

const rawProjectList = computed(() => {
  return projectList.value.filter((project) => {
    if (project.uid === String(DEFAULT_PROJECT_ID)) {
      return false;
    }
    return props.allowedProjectWorkflowTypeList.includes(project.workflow);
  });
});

const getValue = (project: Project): string => {
  return props.useResourceId ? project.name : project.uid;
};

const isOrphanValue = computed(() => {
  if (props.project === undefined) return false;

  return !rawProjectList.value.find((proj) => getValue(proj) === props.project);
});

const combinedProjectList = computed(() => {
  let list = rawProjectList.value.filter((project) => {
    if (props.includeArchived) return true;
    if (project.state === State.ACTIVE) return true;
    // ARCHIVED
    if (getValue(project) === props.project) return true;
    return false;
  });

  // If the current user is not workspace admin/DBA, filter the project list by the given role list.
  if (
    !hasWorkspaceManageProjectPermission.value &&
    props.allowedProjectRoleList.length > 0
  ) {
    list = list.filter((project) => {
      const roles = permissionStore.roleListInProjectV1(
        project,
        currentUserV1.value
      );
      return intersection(props.allowedProjectRoleList, roles).length > 0;
    });
  }

  if (props.filter) {
    list = list.filter(props.filter);
  }

  if (
    props.project &&
    props.project !== String(DEFAULT_PROJECT_ID) &&
    props.project !== String(UNKNOWN_ID) &&
    isOrphanValue.value
  ) {
    // It may happen the selected id might not be in the project list.
    // e.g. the selected project is deleted after the selection and we
    // are unable to cleanup properly. In such case, the selected project id
    // is orphaned and we just display the id
    const dummyProject = {
      ...unknownProject(),
      name: `projects/${props.project}`,
      uid: props.project,
      title: props.project,
    };
    list.unshift(dummyProject);
  }

  if (
    props.project === String(DEFAULT_PROJECT_ID) ||
    props.includeDefaultProject
  ) {
    list.unshift({ ...defaultProject() });
  }

  if (props.project === String(UNKNOWN_ID) || props.includeAll) {
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
      value: getValue(project),
      label:
        project.uid === String(DEFAULT_PROJECT_ID)
          ? t("common.unassigned")
          : project.uid === String(UNKNOWN_ID)
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

watchEffect(prepare);

const renderLabel = (option: SelectOption) => {
  const { project } = option as ProjectSelectOption;
  return (
    <ProjectNameCell
      project={project}
      mode="ALL_SHORT"
      suffix={props.renderSuffix(project.name)}
    />
  );
};
</script>
