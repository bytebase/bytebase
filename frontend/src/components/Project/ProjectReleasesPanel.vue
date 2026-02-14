<template>
  <div class="w-full flex flex-col">
    <div class="px-4 flex flex-col gap-y-2 pb-2">
      <NAlert type="info">
        <span>{{ $t("release.usage-description") }}</span>
        <LearnMoreLink
          url="https://docs.bytebase.com/gitops/migration-based-workflow/release/?source=console"
          class="ml-1"
        />
      </NAlert>
      <!-- Category Filter -->
      <div class="flex items-center gap-x-4">
        <NSelect
          v-model:value="selectedCategory"
          :options="categoryOptions"
          :placeholder="$t('release.filter-by-category')"
          :loading="categoriesLoading"
          clearable
          class="w-64"
        />
        <NButton
          v-if="selectedCategory"
          @click="clearFilters"
          quaternary
        >
          {{ $t('common.clear-filters') }}
        </NButton>
      </div>
    </div>
    <PagedTable
      :key="`${project.name}-${selectedCategory || 'all'}`"
      :session-key="`project-${project.name}-releases`"
      :footer-class="'mx-4'"
      :fetch-list="fetchReleaseList"
    >
      <template #table="{ list, loading }">
        <ReleaseDataTable
          :bordered="false"
          :loading="loading"
          :release-list="list"
        />
      </template>
    </PagedTable>
  </div>
</template>

<script lang="ts" setup>
import { NAlert, NButton, NSelect } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import PagedTable from "@/components/v2/Model/PagedTable.vue";
import { useReleaseCategories } from "@/composables/useReleaseCategories";
import { useReleaseStore } from "@/store";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import {
  buildCategoryFilter,
  buildCategoryQuery,
  parseCategoryFromUrl,
} from "@/utils/releaseFilter";
import ReleaseDataTable from "../Release/ReleaseDataTable.vue";

const props = defineProps<{
  project: Project;
}>();

const route = useRoute();
const router = useRouter();
const releaseStore = useReleaseStore();

// Categories
const projectName = computed(() => props.project.name);
const { categories, loading: categoriesLoading } =
  useReleaseCategories(projectName);

// Selected category from URL
const selectedCategory = ref<string | undefined>(
  parseCategoryFromUrl(route.query)
);

// Category options for dropdown
const categoryOptions = computed(() => {
  const options = categories.value.map((category) => ({
    label: category,
    value: category,
  }));

  // Add "All" option
  return [{ label: "All", value: undefined }, ...options];
});

// Update URL when selection changes
watch(selectedCategory, (newCategory) => {
  const query = buildCategoryQuery(newCategory);
  router.replace({ query });
});

// Update selection when URL changes (browser back/forward)
watch(
  () => route.query,
  (newQuery) => {
    selectedCategory.value = parseCategoryFromUrl(newQuery);
  }
);

const clearFilters = () => {
  selectedCategory.value = undefined;
};

const fetchReleaseList = async ({
  pageToken,
  pageSize,
}: {
  pageToken: string;
  pageSize: number;
}) => {
  const filter = buildCategoryFilter(selectedCategory.value);
  const { nextPageToken, releases } = await releaseStore.fetchReleasesByProject(
    props.project.name,
    { pageSize, pageToken },
    false,
    filter
  );
  return {
    nextPageToken,
    list: releases,
  };
};
</script>
