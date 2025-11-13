<template>
  <div class="flex flex-col gap-y-2">
    <!-- Advanced Search Bar - Always Visible -->
    <div class="flex flex-row items-center gap-x-2">
      <AdvancedSearch
        class="flex-1"
        :params="params"
        :scope-options="scopeOptions"
        @update:params="$emit('update:params', $event)"
      />
      <TimeRange
        v-if="components.includes('time-range')"
        v-model:show="showTimeRange"
        :params="params"
        v-bind="componentProps?.['time-range']"
        @update:params="$emit('update:params', $event)"
      />
      <slot name="searchbox-suffix" />
    </div>

    <slot name="default" />

    <!-- Preset Buttons Row -->
    <div v-if="!componentProps?.presets?.hidden" class="flex flex-col gap-y-2">
      <PresetButtons
        v-if="components.includes('presets')"
        :params="params"
        @update:params="$emit('update:params', $event)"
      />

      <!-- Filter Toggles Row -->
      <FilterToggles
        v-if="components.includes('filters')"
        :params="params"
        @update:params="$emit('update:params', $event)"
      />
    </div>

    <slot name="primary" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import AdvancedSearch from "@/components/AdvancedSearch";
import TimeRange from "@/components/AdvancedSearch/TimeRange.vue";
import type { SearchParams, SearchScopeId } from "@/utils";
import { UIIssueFilterScopeIdList } from "@/utils";
import FilterToggles from "./FilterToggles.vue";
import PresetButtons from "./PresetButtons.vue";
import { useIssueSearchScopeOptions } from "./useIssueSearchScopeOptions";

export type SearchComponent =
  | "searchbox"
  | "presets"
  | "filters"
  | "time-range";

const props = withDefaults(
  defineProps<{
    params: SearchParams;
    overrideScopeIdList?: SearchScopeId[];
    autofocus?: boolean;
    components?: SearchComponent[];
    componentProps?: Partial<Record<SearchComponent, any>>;
  }>(),
  {
    overrideScopeIdList: () => [],
    components: () => ["searchbox", "time-range", "presets", "filters"],
    componentProps: undefined,
  }
);

defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const showTimeRange = ref(false);

const allowedScopes = computed((): SearchScopeId[] => {
  if (props.overrideScopeIdList && props.overrideScopeIdList.length > 0) {
    return props.overrideScopeIdList;
  }
  return [
    ...UIIssueFilterScopeIdList,
    "creator",
    "instance",
    "database",
    "status",
    "taskType",
    "issue-label",
    "project",
    "environment",
  ];
});

const scopeOptions = useIssueSearchScopeOptions(
  computed(() => props.params),
  allowedScopes
);
</script>
