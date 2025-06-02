<template>
  <div class="w-relative">
    <div
      v-if="shouldShowSpecFilter"
      class="w-full sticky top-0 z-10 bg-white px-4 pt-2 pb-1"
    >
      <SpecFilter
        :disabled="state.isRequesting"
        v-model:advice-status-list="state.adviceStatusFilters"
      />
    </div>
    <div class="w-full relative">
      <div
        ref="specBar"
        class="spec-list gap-2 px-4 py-2 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 3xl:grid-cols-5 4xl:grid-cols-6 overflow-y-auto"
        :class="{
          'more-bottom': specBarScrollState.bottom,
          'more-top': specBarScrollState.top,
        }"
        :style="{
          'max-height': `${MAX_LIST_HEIGHT}px`,
        }"
      >
        <SpecCard
          v-for="(spec, i) in filteredSpecList.slice(0, state.index)"
          :key="i"
          :spec="spec"
        />
        <div
          v-show="state.isRequesting"
          class="col-span-full flex items-center justify-center py-4"
        >
          <BBSpin />
        </div>
        <div
          v-if="filteredSpecList.length > state.index"
          class="col-span-full flex flex-row items-center justify-end"
        >
          <NButton
            size="small"
            quaternary
            :loading="state.isRequesting"
            @click="loadMore"
          >
            {{ $t("common.load-more") }}
          </NButton>
        </div>
      </div>
    </div>
  </div>

  <CurrentSpecSection v-if="shouldShowCurrentSpecView" />
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import { NButton } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { BBSpin } from "@/bbkit";
import { useVerticalScrollState } from "@/composables/useScrollState";
import {
  batchGetOrFetchDatabases,
  useDBGroupStore,
} from "@/store";
import { DEBOUNCE_SEARCH_DELAY, isValidDatabaseName } from "@/types";
import type { Advice_Status } from "@/types/proto/v1/sql_service";
import { isDev } from "@/utils";
import {
  isDatabaseChangeSpec,
  isGroupingChangeSpec,
  targetsForSpec,
  usePlanContext,
} from "../../logic";
import { usePlanSQLCheckContext } from "../SQLCheckSection/context";
import CurrentSpecSection from "./CurrentSpecSection.vue";
import SpecCard from "./SpecCard.vue";
import SpecFilter from "./SpecFilter.vue";
import { filterSpec } from "./filter";

interface LocalState {
  // Index is the current number of specs to show.
  index: number;
  adviceStatusFilters: Advice_Status[];
  isRequesting: boolean;
}

const MAX_LIST_HEIGHT = 256;

// The default number of specs to show per page.
// This is set to 4 in development mode for easier testing.
const SPEC_PER_PAGE = isDev() ? 4 : 20;

const planContext = usePlanContext();
const { resultMap } = usePlanSQLCheckContext();
const { plan, selectedSpec } = planContext;
const dbGroupStore = useDBGroupStore();
const state = reactive<LocalState>({
  index: 0,
  adviceStatusFilters: [],
  isRequesting: false,
});
const specBar = ref<HTMLDivElement>();
const specBarScrollState = useVerticalScrollState(specBar, MAX_LIST_HEIGHT);

const specList = computed(() => plan.value?.specs || []);

const filteredSpecList = computed(() => {
  return specList.value.filter((spec) => {
    if (state.adviceStatusFilters.length > 0) {
      if (
        !state.adviceStatusFilters.some((status) =>
          filterSpec(planContext, resultMap.value, spec, {
            adviceStatus: status,
          })
        )
      ) {
        return false;
      }
    }
    return true;
  });
});

const shouldShowSpecFilter = computed(() => {
  return (
    specList.value.length > 2 &&
    // Only show the filter when every spec is a database change spec.
    // Excluding database group spec.
    specList.value.every((spec) => isDatabaseChangeSpec(spec))
  );
});

const shouldShowCurrentSpecView = computed(() => {
  // Only show the current spec view when the selected spec is not in the filtered list.
  const visibleSpecs = filteredSpecList.value.slice(0, state.index);
  return !visibleSpecs.some((spec) => spec.id === selectedSpec.value.id);
});

const loadMore = useDebounceFn(async () => {
  if (state.isRequesting) return;

  const isGroupChangingPlan = specList.value.every((spec) =>
    isGroupingChangeSpec(spec)
  );
  if (isGroupChangingPlan) {
    // Fetch all database groups referenced in specs
    const dbGroupNames = new Set<string>();
    for (const spec of specList.value) {
      const targets = targetsForSpec(spec);
      targets.forEach((target) => dbGroupNames.add(target));
    }
    
    await Promise.all(
      Array.from(dbGroupNames).map((name) =>
        dbGroupStore.getOrFetchDBGroupByName(name)
      )
    );
  } else {
    const fromIndex = state.index;
    const toIndex = fromIndex + SPEC_PER_PAGE;
    
    // Collect all database names from all targets in the visible specs
    const databaseNames = new Set<string>();
    filteredSpecList.value
      .slice(fromIndex, toIndex)
      .forEach((spec) => {
        const targets = targetsForSpec(spec);
        targets.forEach((target) => {
          if (isValidDatabaseName(target)) {
            databaseNames.add(target);
          }
        });
      });

    state.isRequesting = true;
    try {
      await batchGetOrFetchDatabases(Array.from(databaseNames));
    } finally {
      state.index = toIndex;
      state.isRequesting = false;
    }
  }
}, DEBOUNCE_SEARCH_DELAY);

watch(
  () => plan.value.name,
  async () => {
    await loadMore();
  },
  { immediate: true }
);
</script>

<style scoped lang="postcss">
.spec-list::before {
  @apply absolute top-0 h-4 w-full -ml-4 z-10 pointer-events-none transition-shadow;
  content: "";
  box-shadow: none;
}
.spec-list::after {
  @apply absolute bottom-0 h-4 w-full -ml-4 z-10 pointer-events-none transition-shadow;
  content: "";
  box-shadow: none;
}
.spec-list.more-top::before {
  box-shadow: inset 0 0.3rem 0.25rem -0.25rem rgb(0 0 0 / 10%);
}
.spec-list.more-bottom::after {
  box-shadow: inset 0 -0.3rem 0.25rem -0.25rem rgb(0 0 0 / 10%);
}
</style>
