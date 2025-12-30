<template>
  <RemoteResourceSelector
    v-bind="$attrs"
    :multiple="multiple"
    :disabled="disabled"
    :size="size"
    :value="value"
    :render-label="renderLabel"
    :render-tag="renderTag"
    class="bb-project-select"
    :additional-options="additionalOptions"
    :search="handleSearch"
    :filter="filter"
    @update:value="(val) => $emit('update:value', val)"
  >
    <template v-if="$slots.empty" #empty>
      <slot name="empty" />
    </template>
  </RemoteResourceSelector>
</template>

<script lang="tsx" setup>
import { computedAsync } from "@vueuse/core";
import { computed, type VNodeChild } from "vue";
import { useI18n } from "vue-i18n";
import { ProjectNameCell } from "@/components/v2/Model/cells";
import { useProjectV1Store } from "@/store";
import {
  DEFAULT_PROJECT_NAME,
  UNKNOWN_PROJECT_NAME,
  unknownProject,
} from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import RemoteResourceSelector from "./RemoteResourceSelector/index.vue";
import type {
  ResourceSelectOption,
  SelectSize,
} from "./RemoteResourceSelector/types";
import {
  getRenderLabelFunc,
  getRenderTagFunc,
} from "./RemoteResourceSelector/utils";

const props = defineProps<{
  disabled?: boolean;
  value?: string[] | string | undefined; // UNKNOWN_PROJECT_NAME to "ALL"
  includeAll?: boolean;
  includeDefaultProject?: boolean;
  multiple?: boolean;
  size?: SelectSize;
  renderSuffix?: (project: Project) => VNodeChild;
  filter?: (project: Project) => boolean;
}>();

defineEmits<{
  (event: "update:value", name: string[] | string | undefined): void;
}>();

const { t } = useI18n();
const projectStore = useProjectV1Store();

const getOption = (project: Project): ResourceSelectOption<Project> => ({
  resource: project,
  value: project.name,
  label:
    project.name === DEFAULT_PROJECT_NAME
      ? t("common.unassigned")
      : project.title,
});

const additionalOptions = computedAsync(async () => {
  const options: ResourceSelectOption<Project>[] = [];

  let projectNames: string[] = [];
  if (Array.isArray(props.value)) {
    projectNames = props.value;
  } else if (props.value) {
    projectNames = [props.value];
  }

  if (projectNames.includes(UNKNOWN_PROJECT_NAME) || props.includeAll) {
    const dummyAll = {
      ...unknownProject(),
      title: t("project.all"),
    };
    options.unshift(getOption(dummyAll));
  }

  const projects = await projectStore.batchGetOrFetchProjects(projectNames);
  options.push(...projects.map(getOption));

  return options;
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
    options: projects.map(getOption),
  };
};

const customLabel = (project: Project, keyword: string) => {
  if (!project) return null;
  return (
    <ProjectNameCell project={project} keyword={keyword}>
      {{
        suffix: () => (props.renderSuffix ? props.renderSuffix(project) : null),
      }}
    </ProjectNameCell>
  );
};

const renderLabel = computed(() => {
  return getRenderLabelFunc({
    multiple: props.multiple,
    customLabel,
    showResourceName: true,
  });
});

const renderTag = computed(() => {
  return getRenderTagFunc({
    multiple: props.multiple,
    disabled: props.disabled,
    size: props.size,
    customLabel,
  });
});
</script>
