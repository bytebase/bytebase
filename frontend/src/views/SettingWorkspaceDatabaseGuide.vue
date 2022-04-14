<template>
  <div class="mx-auto">
    <div class="flex justify-start mt-4">
      <div class="flex flex-col items-center w-28">
        <button class="btn-icon-primary p-3" @click.prevent="goToCreationView">
          <heroicons-outline:plus-sm class="w-6 h-6" />
        </button>
        <h3 class="mt-1 text-base font-normal text-main tracking-tight">
          {{ $t("database-review-guide.add-guideline") }}
        </h3>
      </div>
    </div>
    <div class="py-2 flex justify-between items-center mt-2">
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
    <SchemaGuideTable :guide-list="filteredGuideList" />
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
}

const state = reactive<LocalState>({
  searchText: "",
  selectedEnvironment: router.currentRoute.value.query.environment
    ? useEnvironmentStore().getEnvironmentById(
        parseInt(router.currentRoute.value.query.environment as string, 10)
      )
    : undefined,
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

const filteredGuideList = computed((): DatabaseSchemaGuide[] => {
  const list = store.guideList;
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
});
</script>
