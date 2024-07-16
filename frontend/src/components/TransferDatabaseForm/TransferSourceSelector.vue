<template>
  <div class="textlabel">
    <div
      class="flex flex-col md:flex-row md:items-center gap-y-2 justify-between"
    >
      <div v-if="project.name !== DEFAULT_PROJECT_NAME" class="radio-set-row">
        <NRadioGroup v-model:value="state.transferSource">
          <NRadio v-if="hasPermissionForDefaultProject" :value="'DEFAULT'">
            {{ $t("quick-action.from-unassigned-databases") }}
          </NRadio>
          <NRadio :value="'OTHER'">
            {{ $t("quick-action.from-projects") }}
          </NRadio>
        </NRadioGroup>
      </div>
      <NInputGroup style="width: auto">
        <InstanceSelect
          v-if="
            state.transferSource == 'DEFAULT' && hasPermissionForDefaultProject
          "
          class="!w-48"
          :instance="instanceFilter?.name ?? UNKNOWN_INSTANCE_NAME"
          :include-all="true"
          :filter="filterInstance"
          @update:instance-name="changeInstanceFilter"
        />
        <ProjectSelect
          v-else-if="state.transferSource == 'OTHER'"
          :include-all="true"
          :project-name="projectFilter?.name ?? UNKNOWN_PROJECT_NAME"
          :allowed-project-role-list="[PresetRoleType.PROJECT_OWNER]"
          :filter="filterSourceProject"
          @update:project-name="changeProjectFilter"
        />
        <SearchBox
          :value="searchText"
          :placeholder="$t('database.filter-database')"
          @update:value="$emit('search-text-change', $event)"
        />
      </NInputGroup>
    </div>
    <div v-if="state.transferSource == 'DEFAULT'" class="textinfolabel mt-2">
      {{ $t("quick-action.unassigned-db-hint") }}
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NInputGroup, NRadio, NRadioGroup } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { InstanceSelect, ProjectSelect, SearchBox } from "@/components/v2";
import { useInstanceV1Store, useProjectV1Store } from "@/store";
import type { ComposedDatabase, ComposedInstance } from "@/types";
import {
  DEFAULT_PROJECT_NAME,
  PresetRoleType,
  UNKNOWN_INSTANCE_NAME,
  UNKNOWN_PROJECT_NAME,
  isValidInstanceName,
  isValidProjectName,
} from "@/types";
import type { Project } from "@/types/proto/v1/project_service";
import type { TransferSource } from "./utils";

interface LocalState {
  transferSource: TransferSource;
}

const props = withDefaults(
  defineProps<{
    project: Project;
    rawDatabaseList?: ComposedDatabase[];
    transferSource: TransferSource;
    hasPermissionForDefaultProject: boolean;
    instanceFilter?: ComposedInstance;
    projectFilter?: Project;
    searchText: string;
  }>(),
  {
    rawDatabaseList: () => [],
    instanceFilter: undefined,
    projectFilter: undefined,
    searchText: "",
  }
);

const emit = defineEmits<{
  (event: "change", src: TransferSource): void;
  (event: "select-instance", instance: ComposedInstance | undefined): void;
  (event: "select-project", project: Project | undefined): void;
  (event: "search-text-change", searchText: string): void;
}>();

const state = reactive<LocalState>({
  transferSource: props.transferSource,
});

const filterSourceProject = (project: Project) => {
  return project.name !== props.project.name;
};

const nonEmptyInstanceNameSet = computed(() => {
  return new Set(props.rawDatabaseList.map((db) => db.instance));
});

const changeInstanceFilter = (name: string | undefined) => {
  if (!isValidInstanceName(name)) {
    return emit("select-instance", undefined);
  }
  emit("select-instance", useInstanceV1Store().getInstanceByName(name));
};

const filterInstance = (instance: ComposedInstance) => {
  if (instance.name === UNKNOWN_INSTANCE_NAME) return true; // "ALL" can be displayed.
  return nonEmptyInstanceNameSet.value.has(instance.name);
};

const changeProjectFilter = (name: string | undefined) => {
  if (!isValidProjectName(name)) {
    return emit("select-project", undefined);
  }
  emit("select-project", useProjectV1Store().getProjectByName(name));
};

watch(
  () => props.transferSource,
  (src) => (state.transferSource = src)
);

watch(
  () => state.transferSource,
  (src) => {
    if (src !== props.transferSource) {
      emit("change", src);
    }
  }
);
</script>
