<template>
  <NSelect
    :value="project"
    :options="options"
    :placeholder="$t('project.select')"
    :filterable="true"
    :filter="filterByName"
    style="width: 12rem"
    @update:value="$emit('update:project', $event)"
  />
</template>

<script lang="ts" setup>
import { intersection } from "lodash-es";
import { NSelect, SelectOption } from "naive-ui";
import { computed, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useCurrentUserV1, useProjectV1Store } from "@/store";
import { DEFAULT_PROJECT_ID, UNKNOWN_ID, unknownProject } from "@/types";
import { State } from "@/types/proto/v1/common";
import {
  Project,
  TenantMode,
  Workflow,
} from "@/types/proto/v1/project_service";
import { roleListInProjectV1 } from "@/utils";

interface ProjectSelectOption extends SelectOption {
  value: string;
  project: Project;
}

const props = withDefaults(
  defineProps<{
    project: string | undefined; // UNKNOWN_ID(-1) to "ALL"
    allowedProjectRoleList?: string[]; // Empty array([]) to "ALL"
    allowedProjectTenantModeList?: TenantMode[];
    allowedProjectWorkflowTypeList?: Workflow[];
    includeAll?: boolean;
    includeDefaultProject?: boolean;
    includeArchived?: boolean;
    filter?: (project: Project, index: number) => boolean;
  }>(),
  {
    allowedProjectRoleList: () => [],
    allowedProjectTenantModeList: () => [
      TenantMode.TENANT_MODE_DISABLED,
      TenantMode.TENANT_MODE_ENABLED,
    ],
    allowedProjectWorkflowTypeList: () => [Workflow.UI, Workflow.VCS],
    includeAll: false,
    includeDefaultProject: false,
    includeArchived: false,
    filter: () => true,
  }
);

defineEmits<{
  (event: "update:project", id: string | undefined): void;
}>();

const { t } = useI18n();
const currentUserV1 = useCurrentUserV1();
const projectV1Store = useProjectV1Store();

const prepare = () => {
  projectV1Store.fetchProjectList(true /* showDeleted */);
};

const rawProjectList = computed(() => {
  let list = projectV1Store.getProjectListByUser(
    currentUserV1.value,
    true /* showDeleted */
  );
  // Filter the default project
  list = list.filter((project) => {
    return project.uid !== String(DEFAULT_PROJECT_ID);
  });
  // Filter by project tenant mode
  list = list.filter((project) => {
    return props.allowedProjectTenantModeList.includes(project.tenantMode);
  });
  // Filter by project workflow type
  list = list.filter((project) => {
    return props.allowedProjectWorkflowTypeList.includes(project.workflow);
  });

  return list;
});

const isOrphanValue = computed(() => {
  if (props.project === undefined) return false;

  return !rawProjectList.value.find((proj) => proj.uid === props.project);
});

const combinedProjectList = computed(() => {
  let list = rawProjectList.value.filter((project) => {
    if (props.includeArchived) return true;
    if (project.state === State.ACTIVE) return true;
    // ARCHIVED
    if (project.uid === props.project) return true;
    return false;
  });

  if (props.allowedProjectRoleList.length > 0) {
    list = list.filter((project) => {
      const roles = roleListInProjectV1(project.iamPolicy, currentUserV1.value);
      return intersection(props.allowedProjectRoleList, roles).length > 0;
    });
  }

  if (props.includeDefaultProject) {
    list.unshift(projectV1Store.getProjectByUID(String(DEFAULT_PROJECT_ID)));
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
      value: project.uid,
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
</script>
