<template>
  <div class="textlabel">
    <div
      class="flex flex-col md:flex-row md:items-center gap-y-2 justify-between"
    >
      <div
        v-if="project.name !== DEFAULT_PROJECT_V1_NAME"
        class="radio-set-row"
      >
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
          :instance="instanceFilter?.uid ?? String(UNKNOWN_ID)"
          :include-all="true"
          :filter="filterInstance"
          @update:instance="changeInstanceFilter"
        />
        <ProjectSelect
          v-else-if="state.transferSource == 'OTHER'"
          :include-all="true"
          :project="projectFilter?.uid ?? String(UNKNOWN_ID)"
          :allowed-project-role-list="[PresetRoleType.PROJECT_OWNER]"
          :filter="filterSourceProject"
          @update:project="changeProjectFilter"
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
import {
  UNKNOWN_ID,
  ComposedDatabase,
  ComposedInstance,
  DEFAULT_PROJECT_V1_NAME,
  PresetRoleType,
} from "@/types";
import { Project } from "@/types/proto/v1/project_service";
import { TransferSource } from "./utils";

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
  return project.uid !== props.project.uid;
};

const nonEmptyInstanceUidSet = computed(() => {
  const instanceList = props.rawDatabaseList.map((db) => db.instanceEntity);
  return new Set(instanceList.map((instance) => instance.uid));
});

const changeInstanceFilter = (uid: string | undefined) => {
  if (!uid || uid === String(UNKNOWN_ID)) {
    return emit("select-instance", undefined);
  }
  emit("select-instance", useInstanceV1Store().getInstanceByUID(uid));
};

const filterInstance = (instance: ComposedInstance) => {
  if (instance.uid === String(UNKNOWN_ID)) return true; // "ALL" can be displayed.
  return nonEmptyInstanceUidSet.value.has(instance.uid);
};

const changeProjectFilter = (uid: string | undefined) => {
  if (!uid || uid === String(UNKNOWN_ID)) {
    return emit("select-project", undefined);
  }
  emit("select-project", useProjectV1Store().getProjectByUID(uid));
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
