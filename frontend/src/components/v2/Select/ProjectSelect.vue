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
import { computed, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { NSelect, SelectOption } from "naive-ui";

import { useCurrentUser, useProjectStore } from "@/store";
import {
  DEFAULT_PROJECT_ID,
  Project,
  ProjectId,
  ProjectRoleType,
  ProjectTenantMode,
  ProjectWorkflowType,
  UNKNOWN_ID,
  unknown,
} from "@/types";
import { memberListInProject } from "@/utils";

interface ProjectSelectOption extends SelectOption {
  value: ProjectId;
  project: Project;
}

const props = withDefaults(
  defineProps<{
    project: ProjectId | undefined; // UNKNOWN_ID(-1) to "ALL"
    allowedProjectRoleList?: ProjectRoleType[];
    allowedProjectTenantModeList?: ProjectTenantMode[];
    allowedProjectWorkflowTypeList?: ProjectWorkflowType[];
    includeAll?: boolean;
    includeDefaultProject?: boolean;
    includeArchived?: boolean;
    filter?: (project: Project, index: number) => boolean;
  }>(),
  {
    allowedProjectRoleList: () => ["OWNER", "DEVELOPER"],
    allowedProjectTenantModeList: () => ["DISABLED", "TENANT"],
    allowedProjectWorkflowTypeList: () => ["UI", "VCS"],
    includeAll: false,
    includeDefaultProject: false,
    includeArchived: false,
    filter: () => true,
  }
);

defineEmits<{
  (event: "update:project", id: ProjectId | undefined): void;
}>();

const { t } = useI18n();
const currentUser = useCurrentUser();
const projectStore = useProjectStore();

const prepare = () => {
  projectStore.fetchProjectListByUser({
    userId: currentUser.value.id,
    rowStatusList: ["NORMAL", "ARCHIVED"],
  });
};

const rawProjectList = computed(() => {
  let list = projectStore.getProjectListByUser(currentUser.value.id, [
    "NORMAL",
    "ARCHIVED",
  ]);
  // Filter by role
  list = list.filter((project) => {
    const memberList = memberListInProject(
      project,
      currentUser.value,
      props.allowedProjectRoleList
    );
    return memberList.length > 0;
  });
  // Filter by project tenant mode
  list = list.filter((project) => {
    return props.allowedProjectTenantModeList.includes(project.tenantMode);
  });
  // Filter by project workflow type
  list = list.filter((project) => {
    return props.allowedProjectWorkflowTypeList.includes(project.workflowType);
  });

  return list;
});

const isOrphanValue = computed(() => {
  if (props.project === undefined) return false;

  return !rawProjectList.value.find((proj) => proj.id === props.project);
});

const combinedProjectList = computed(() => {
  let list = rawProjectList.value.filter((project) => {
    if (props.includeArchived) return true;
    if (project.rowStatus === "NORMAL") return true;
    // ARCHIVED
    if (project.id === props.project) return true;
    return false;
  });

  if (props.includeDefaultProject) {
    list.unshift(projectStore.getProjectById(DEFAULT_PROJECT_ID));
  }

  if (props.filter) {
    list = list.filter(props.filter);
  }

  if (
    props.project &&
    props.project !== DEFAULT_PROJECT_ID &&
    props.project !== UNKNOWN_ID &&
    isOrphanValue.value
  ) {
    // It may happen the selected id might not be in the project list.
    // e.g. the selected project is deleted after the selection and we
    // are unable to cleanup properly. In such case, the selected project id
    // is orphaned and we just display the id
    const dummyProject = unknown("PROJECT");
    dummyProject.id = props.project;
    dummyProject.name = props.project.toString();
    list.unshift(dummyProject);
  }

  if (props.project === UNKNOWN_ID || props.includeAll) {
    const dummyAll = unknown("PROJECT");
    dummyAll.name = t("project.all");
    list.unshift(dummyAll);
  }

  return list;
});

const options = computed(() => {
  return combinedProjectList.value.map<ProjectSelectOption>((project) => {
    return {
      project,
      value: project.id,
      label:
        project.id === DEFAULT_PROJECT_ID
          ? t("common.unassigned")
          : project.id === UNKNOWN_ID
          ? t("project.all")
          : project.name,
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
