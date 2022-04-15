<template>
  <div class="mx-auto">
    <div class="flex justify-start mt-4">
      <div class="flex flex-col items-center w-28">
        <button class="btn-icon-primary p-3" @click.prevent="goToCreationView">
          <heroicons-outline:plus-sm class="w-6 h-6" />
        </button>
        <h3 class="mt-1 text-base font-normal text-main tracking-tight">
          {{ $t("schame-review.add-review") }}
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
        :placeholder="$t('schame-review.search-review-name')"
        @change-text="(text) => (state.searchText = text)"
      />
    </div>
    <SchemaReviewTable :review-list="filteredReviewList" />
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { useRouter } from "vue-router";
import { useEnvironmentStore, useSchemaSystemStore } from "@/store";
import { Environment, DatabaseSchemaReview } from "../types";

const router = useRouter();
const store = useSchemaSystemStore();
const ROUTE_NAME = "setting.workspace.schame-review";

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

const filteredReviewList = computed((): DatabaseSchemaReview[] => {
  const list = store.reviewList;
  if (!state.selectedEnvironment && !state.searchText) {
    // Select "All"
    return list;
  }
  return list.filter((review) => {
    return (
      (!state.selectedEnvironment ||
        new Set(review.environmentList).has(state.selectedEnvironment.id)) &&
      (!state.searchText ||
        review.name.toLowerCase().includes(state.searchText.toLowerCase()))
    );
  });
});
</script>
