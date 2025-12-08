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
      class="w-5xl max-w-[100vw] relative"
    >
      <AdvancedSearch
        v-model:params="state.params"
        class="flex-1"
        :autofocus="false"
        :placeholder="$t('database.filter-database')"
        :scope-options="scopeOptions"
        :cache-query="false"
      />

      <PagedDatabaseTable
        class="mt-2"
        mode="PROJECT_SHORT"
        :parent="project"
        :filter="filter"
        :custom-click="true"
        v-model:selected-database-names="state.selectedDatabaseNameList"
      />

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
import { computed, reactive, watch } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import {
  environmentNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { SearchParams, SearchScope } from "@/utils";
import {
  CommonFilterScopeIdList,
  extractProjectResourceName,
  getValueFromSearchParams,
} from "@/utils";
import AdvancedSearch from "../AdvancedSearch";
import { useCommonSearchScopeOptions } from "../AdvancedSearch/useCommonSearchScopeOptions";
import { PagedDatabaseTable } from "../v2/Model/DatabaseV1Table";

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

const readonlyScopes = computed((): SearchScope[] => [
  {
    id: "project",
    value: extractProjectResourceName(props.project),
    readonly: true,
  },
]);

const emit = defineEmits<{
  (event: "close"): void;
  (event: "update", databaseNameList: string[]): void;
}>();

const databaseStore = useDatabaseV1Store();

const state = reactive<LocalState>({
  selectedDatabaseNameList: props.selectedDatabaseNameList || [],
  params: {
    query: "",
    scopes: [...readonlyScopes.value],
  },
});

watch(
  () => props.project,
  () => {
    state.params = {
      query: "",
      scopes: [...readonlyScopes.value],
    };
  }
);

const scopeOptions = useCommonSearchScopeOptions([...CommonFilterScopeIdList]);

const selectedDatabaseList = computed(() =>
  state.selectedDatabaseNameList.map((name) =>
    databaseStore.getDatabaseByName(name)
  )
);

const selectedInstance = computed(() => {
  return getValueFromSearchParams(state.params, "instance", instanceNamePrefix);
});

const selectedEnvironment = computed(() => {
  return getValueFromSearchParams(
    state.params,
    "environment",
    environmentNamePrefix
  );
});

const filter = computed(() => ({
  instance: selectedInstance.value,
  environment: selectedEnvironment.value,
  query: state.params.query,
}));

const handleConfirm = async () => {
  emit("update", state.selectedDatabaseNameList);
};
</script>
