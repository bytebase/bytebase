<template>
  <div class="mx-auto">
    <div class="flex justify-end">
      <button
        type="button"
        class="btn-primary inline-flex justify-center w-60 mt-8"
        @click="goToCreationView"
      >
        <div class="flex">
          <heroicons-solid:plus-circle class="w-5 h-5 mr-2" />
          {{ $t("database-review-guide.add-guideline") }}
        </div>
      </button>
    </div>
    <div class="py-2 flex justify-between items-center mt-10">
      <!-- eslint-disable vue/attribute-hyphenation -->
      <EnvironmentTabFilter
        :selectedId="state.selectedEnvironment?.id"
        @select-environment="selectEnvironment"
      />
      <BBTableSearch
        ref="searchField"
        :placeholder="$t('database-review-guide.search-guideline-name')"
        @change-text="(text) => (state.searchText = text)"
      />
    </div>
    <SchemaGuideTable :guide-list="filteredList(guideList)" />
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import {
  useEnvironmentList,
  useEnvironmentStore,
  useSchemaSystemStore,
} from "@/store";
import { Environment, DatabaseSchemaGuide } from "../types";

const { t } = useI18n();
const router = useRouter();
const store = useSchemaSystemStore();
const environmentList = useEnvironmentList(["NORMAL"]);
const ROUTE_NAME = "setting.workspace.database-review-guide";

interface LocalState {
  searchText: string;
  selectedEnvironment?: Environment;
  showGuide: boolean;
}

const state = reactive<LocalState>({
  searchText: "",
  selectedEnvironment: router.currentRoute.value.query.environment
    ? useEnvironmentStore().getEnvironmentById(
        parseInt(router.currentRoute.value.query.environment as string, 10)
      )
    : undefined,
  showGuide: false,
});

const selectEnvironment = (environment: Environment) => {
  state.selectedEnvironment = environment;
  if (environment) {
    router.replace({
      name: ROUTE_NAME,
      query: { environment: environment.id },
    });
  } else {
    router.replace({ name: ROUTE_NAME });
  }
};

const goToCreationView = () => {
  router.push({
    name: `${ROUTE_NAME}.create`,
  });
};

const guideList = computed(() => {
  return store.guideList;
});

const filteredList = (list: DatabaseSchemaGuide[]) => {
  if (!state.selectedEnvironment && !state.searchText) {
    // Select "All"
    return list;
  }
  return list.filter((guide) => {
    return (
      (!state.selectedEnvironment ||
        new Set(guide.environmentList).has(state.selectedEnvironment.id)) &&
      (!state.searchText ||
        guide.name.toLowerCase().includes(state.searchText.toLowerCase()))
    );
  });
};
</script>
