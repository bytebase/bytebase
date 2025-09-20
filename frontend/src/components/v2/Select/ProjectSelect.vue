<template>
  <ResourceSelect
    v-bind="$attrs"
    :remote="true"
    :loading="state.loading"
    :placeholder="$t('project.select')"
    :multiple="multiple"
    :disabled="disabled"
    :value="projectName"
    :values="projectNames"
    :options="options"
    :custom-label="renderLabel"
    class="bb-project-select"
    @search="handleSearch"
    @update:value="(val) => $emit('update:project-name', val)"
    @update:values="(val) => $emit('update:project-names', val)"
  >
    <template v-if="$slots.empty" #empty>
      <slot name="empty" />
    </template>
  </ResourceSelect>
</template>

<script lang="tsx" setup>
import { useDebounceFn } from "@vueuse/core";
import { intersection } from "lodash-es";
import { computed, watchEffect, reactive, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import { ProjectNameCell } from "@/components/v2/Model/cells";
import { useProjectV1Store, usePermissionStore } from "@/store";
import {
  unknownProject,
  defaultProject,
  DEFAULT_PROJECT_NAME,
  UNKNOWN_PROJECT_NAME,
  isValidProjectName,
  DEBOUNCE_SEARCH_DELAY,
} from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { hasWorkspacePermissionV2, getDefaultPagination } from "@/utils";
import ResourceSelect from "./ResourceSelect.vue";

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
    filter?: (project: Project, index: number) => boolean;
    defaultSelectFirst?: boolean;
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
    defaultSelectFirst: false,
  }
);

const emit = defineEmits<{
  (event: "update:project-name", name: string | undefined): void;
  (event: "update:project-names", names: string[]): void;
}>();

interface LocalState {
  loading: boolean;
  rawProjectList: Project[];
}

const { t } = useI18n();
const permissionStore = usePermissionStore();
const projectStore = useProjectV1Store();
const state = reactive<LocalState>({
  loading: true,
  rawProjectList: [],
});

const initSelectedProjects = async (projectNames: string[]) => {
  for (const projectName of projectNames) {
    if (isValidProjectName(projectName)) {
      const project = await projectStore.getOrFetchProjectByName(projectName);
      if (!state.rawProjectList.find((p) => p.name === project.name)) {
        state.rawProjectList.unshift(project);
      }
    }
  }
};

const initProjectList = async () => {
  if (props.projectName) {
    await initSelectedProjects([props.projectName]);
  }
  if (props.projectNames) {
    await initSelectedProjects(props.projectNames);
  }

  if (
    props.projectName === DEFAULT_PROJECT_NAME ||
    props.includeDefaultProject
  ) {
    if (
      !state.rawProjectList.find((proj) => proj.name === DEFAULT_PROJECT_NAME)
    ) {
      state.rawProjectList.unshift({ ...defaultProject() });
    }
  }

  if (props.projectName === UNKNOWN_PROJECT_NAME || props.includeAll) {
    const dummyAll = {
      ...unknownProject(),
      title: t("project.all"),
    };
    state.rawProjectList.unshift(dummyAll);
  }
};

const hasWorkspaceManageProjectPermission = computed(() =>
  hasWorkspacePermissionV2("bb.projects.list")
);

const combinedProjectList = computed(() => {
  let list = [...state.rawProjectList];

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

  return list;
});

const handleSearch = useDebounceFn(async (search: string) => {
  state.loading = true;
  try {
    const { projects } = await projectStore.fetchProjectList({
      filter: {
        query: search,
        excludeDefault: !props.includeDefaultProject,
      },
      pageSize: getDefaultPagination(),
    });
    state.rawProjectList = projects;
    if (!search) {
      await initProjectList();
    }
  } finally {
    state.loading = false;
  }
}, DEBOUNCE_SEARCH_DELAY);

onMounted(async () => {
  await handleSearch("");
});

const options = computed(
  (): {
    resource: Project;
    value: string;
    label: string;
  }[] => {
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
  }
);

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
