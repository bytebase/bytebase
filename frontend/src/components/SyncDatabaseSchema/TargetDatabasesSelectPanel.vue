<template>
  <Drawer
    :show="true"
    :close-on-esc="!loading"
    :mask-closable="!loading"
    width="auto"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <DrawerContent
      :title="$t('database.sync-schema.target-databases')"
      :closable="true"
      class="w-[64rem] max-w-[100vw] relative"
    >
      <AdvancedSearch
        v-model:params="state.params"
        class="flex-1"
        :autofocus="false"
        :placeholder="$t('database.filter-database')"
        :scope-options="scopeOptions"
      />
      <DatabaseV1Table
        class="mt-2"
        mode="PROJECT"
        :database-list="filteredDatabaseList"
        :show-selection="true"
        :selected-database-names="state.selectedDatabaseNameList"
        :keyword="state.params.query.trim().toLowerCase()"
        @update:selected-databases="
          state.selectedDatabaseNameList = Array.from($event)
        "
      />
      <MaskSpinner v-if="loading || !ready" />

      <template #footer>
        <div class="flex-1 flex items-center justify-between">
          <NTooltip :disabled="state.selectedDatabaseNameList.length === 0">
            <template #trigger>
              <div class="textinfolabel">
                {{
                  $t("database.selected-n-databases", {
                    n: state.selectedDatabaseNameList.length,
                  })
                }}
              </div>
            </template>
            <div class="mx-2">
              <ul class="list-disc">
                <li v-for="db in selectedDatabaseList" :key="db.name">
                  {{ db.databaseName }}
                </li>
              </ul>
            </div>
          </NTooltip>
          <div class="flex items-center justify-end gap-x-2">
            <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
            <NButton
              :disabled="state.selectedDatabaseNameList.length === 0"
              type="primary"
              @click="handleConfirm"
            >
              {{ $t("common.select") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { NButton, NTooltip } from "naive-ui";
import { computed, reactive } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import { useDatabaseV1List } from "@/store/modules/v1/databaseList";
import { UNKNOWN_ID } from "@/types";
import type { Engine } from "@/types/proto/v1/common";
import {
  CommonFilterScopeIdList,
  extractEnvironmentResourceName,
  extractInstanceResourceName,
  filterDatabaseV1ByKeyword,
  type SearchParams,
} from "@/utils";
import AdvancedSearch from "../AdvancedSearch";
import { useCommonSearchScopeOptions } from "../AdvancedSearch/useCommonSearchScopeOptions";
import MaskSpinner from "../misc/MaskSpinner.vue";
import DatabaseV1Table from "../v2/Model/DatabaseV1Table";

type LocalState = {
  selectedDatabaseNameList: string[];
  params: SearchParams;
};

const props = defineProps<{
  project: string;
  engine: Engine;
  selectedDatabaseNameList?: string[];
  loading?: boolean;
}>();

const emit = defineEmits<{
  (event: "close"): void;
  (event: "update", databaseNameList: string[]): void;
}>();

const databaseStore = useDatabaseV1Store();
const state = reactive<LocalState>({
  selectedDatabaseNameList: props.selectedDatabaseNameList || [],
  params: {
    query: "",
    scopes: [],
  },
});

const { databaseList, ready } = useDatabaseV1List(props.project);

const scopeOptions = useCommonSearchScopeOptions(
  computed(() => state.params),
  [...CommonFilterScopeIdList]
);

const selectedDatabaseList = computed(() =>
  state.selectedDatabaseNameList.map((name) =>
    databaseStore.getDatabaseByName(name)
  )
);

const selectedInstance = computed(() => {
  return (
    state.params.scopes.find((scope) => scope.id === "instance")?.value ??
    `${UNKNOWN_ID}`
  );
});

const selectedEnvironment = computed(() => {
  return (
    state.params.scopes.find((scope) => scope.id === "environment")?.value ??
    `${UNKNOWN_ID}`
  );
});

const filteredDatabaseList = computed(() => {
  let list = databaseList.value;
  if (selectedEnvironment.value !== `${UNKNOWN_ID}`) {
    list = list.filter(
      (db) =>
        extractEnvironmentResourceName(db.effectiveEnvironment) ===
        selectedEnvironment.value
    );
  }
  if (selectedInstance.value !== `${UNKNOWN_ID}`) {
    list = list.filter(
      (db) =>
        extractInstanceResourceName(db.instance) === selectedInstance.value
    );
  }
  const keyword = state.params.query.trim().toLowerCase();
  if (keyword) {
    list = list.filter((db) =>
      filterDatabaseV1ByKeyword(db, keyword, [
        "name",
        "environment",
        "instance",
      ])
    );
  }
  return list;
});

const handleConfirm = async () => {
  emit("update", state.selectedDatabaseNameList);
};
</script>
