<template>
  <div class="textlabel">
    <div
      class="flex flex-col md:flex-row md:items-center gap-y-2 justify-between"
    >
      <div v-if="project.name !== DEFAULT_PROJECT_NAME" class="radio-set-row">
        <NRadioGroup v-model:value="state.transferSource">
          <NRadio :value="'OTHER'">
            {{ $t("quick-action.from-projects") }}
          </NRadio>
          <NRadio v-if="hasPermissionForDefaultProject" :value="'DEFAULT'">
            {{ $t("quick-action.from-unassigned-databases") }}
          </NRadio>
        </NRadioGroup>
      </div>
      <NInputGroup style="width: auto">
        <InstanceSelect
          class="!w-44"
          :instance="instanceFilter?.name ?? UNKNOWN_INSTANCE_NAME"
          :include-all="true"
          :filter="filterInstance"
          @update:instance-name="changeInstanceFilter"
        />
        <SearchBox
          class="!w-44"
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
import { InstanceSelect, SearchBox } from "@/components/v2";
import { useInstanceResourceByName } from "@/store";
import type { ComposedDatabase } from "@/types";
import {
  DEFAULT_PROJECT_NAME,
  UNKNOWN_INSTANCE_NAME,
  isValidInstanceName,
} from "@/types";
import type { InstanceResource } from "@/types/proto/v1/instance_service";
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
    instanceFilter?: InstanceResource;
    searchText: string;
  }>(),
  {
    rawDatabaseList: () => [],
    instanceFilter: undefined,
    searchText: "",
  }
);

const emit = defineEmits<{
  (event: "change", src: TransferSource): void;
  (event: "select-instance", instance: InstanceResource | undefined): void;
  (event: "search-text-change", searchText: string): void;
}>();

const state = reactive<LocalState>({
  transferSource: props.transferSource,
});

const nonEmptyInstanceNameSet = computed(() => {
  return new Set(props.rawDatabaseList.map((db) => db.instance));
});

const changeInstanceFilter = (name: string | undefined) => {
  if (!isValidInstanceName(name)) {
    return emit("select-instance", undefined);
  }
  emit("select-instance", useInstanceResourceByName(name));
};

const filterInstance = (instance: InstanceResource) => {
  if (instance.name === UNKNOWN_INSTANCE_NAME) return true; // "ALL" can be displayed.
  return nonEmptyInstanceNameSet.value.has(instance.name);
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
