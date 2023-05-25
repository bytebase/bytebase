<template>
  <div class="textlabel">
    <div v-if="state.transferSource == 'DEFAULT'" class="textinfolabel mb-2">
      {{ $t("quick-action.unassigned-db-hint") }}
    </div>
    <div class="flex items-center justify-between">
      <div class="radio-set-row">
        <template v-if="project.uid !== String(DEFAULT_PROJECT_ID)">
          <label class="radio">
            <input
              v-model="state.transferSource"
              tabindex="-1"
              type="radio"
              class="btn"
              value="DEFAULT"
            />
            <span class="label">
              {{ $t("quick-action.from-unassigned-databases") }}
            </span>
          </label>
          <label class="radio">
            <input
              v-model="state.transferSource"
              tabindex="-1"
              type="radio"
              class="btn"
              value="OTHER"
            />
            <span class="label">
              {{ $t("quick-action.from-projects") }}
            </span>
          </label>
        </template>
      </div>
      <NInputGroup style="width: auto">
        <InstanceSelect
          :instance="String(instanceFilter?.id ?? UNKNOWN_ID)"
          :include-all="true"
          :filter="filterInstance"
          @update:instance="changeInstanceFilter"
        />
        <SearchBox
          :value="searchText"
          :placeholder="$t('database.search-database')"
          @update:value="$emit('search-text-change', $event)"
        />
      </NInputGroup>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { type PropType, computed, reactive, watch } from "vue";
import { NInputGroup } from "naive-ui";

import { TransferSource } from "./utils";
import {
  type Instance,
  DEFAULT_PROJECT_ID,
  UNKNOWN_ID,
  InstanceId,
  Database,
  ComposedInstance,
} from "@/types";
import { InstanceSelect, SearchBox } from "@/components/v2";
import { useInstanceStore } from "@/store";
import { Project } from "@/types/proto/v1/project_service";

interface LocalState {
  transferSource: TransferSource;
}

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
  rawDatabaseList: {
    type: Array as PropType<Database[]>,
    default: () => [],
  },
  transferSource: {
    type: String as PropType<TransferSource>,
    required: true,
  },
  instanceFilter: {
    type: Object as PropType<Instance>,
    default: undefined,
  },
  searchText: {
    type: String,
    default: "",
  },
});

const emit = defineEmits<{
  (event: "change", src: TransferSource): void;
  (event: "select-instance", instance: Instance | undefined): void;
  (event: "search-text-change", searchText: string): void;
}>();

const state = reactive<LocalState>({
  transferSource: props.transferSource,
});

const nonEmptyInstanceIdSet = computed(() => {
  const instanceList = props.rawDatabaseList.map((db) => db.instance);
  return new Set(instanceList.map((instance) => String(instance.id)));
});

const changeInstanceFilter = (instanceId: InstanceId | undefined) => {
  if (!instanceId || instanceId === UNKNOWN_ID) {
    return emit("select-instance", undefined);
  }
  emit("select-instance", useInstanceStore().getInstanceById(instanceId));
};

const filterInstance = (instance: ComposedInstance) => {
  if (instance.uid === String(UNKNOWN_ID)) return true; // "ALL" can be displayed.
  return nonEmptyInstanceIdSet.value.has(instance.uid);
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
