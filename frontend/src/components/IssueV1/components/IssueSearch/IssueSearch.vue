<template>
  <div class="flex flex-col gap-y-2">
    <div class="flex items-center md:gap-2">
      <AdvancedSearch
        class="flex-1 min-w-0"
        :params="params"
        :scope-options="scopeOptions"
        :default-params="defaultParams"
        @update:params="$emit('update:params', $event)"
      />
      <TimeRange
        v-if="components.includes('time-range')"
        :params="params"
        v-bind="componentProps?.['time-range']"
        @update:params="$emit('update:params', $event)"
      />
      <IssueSortDropdown
        v-if="components.includes('sort')"
        :order-by="orderBy"
        @update:order-by="$emit('update:orderBy', $event)"
      />
      <slot name="searchbox-suffix" />
    </div>

    <PresetButtons
      v-if="components.includes('presets') && !componentProps?.presets?.hidden"
      :params="params"
      @update:params="$emit('update:params', $event)"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import AdvancedSearch from "@/components/AdvancedSearch";
import TimeRange from "@/components/AdvancedSearch/TimeRange.vue";
import type { SearchParams, SearchScopeId } from "@/utils";

import IssueSortDropdown from "./IssueSortDropdown.vue";
import PresetButtons from "./PresetButtons.vue";
import { useIssueSearchScopeOptions } from "./useIssueSearchScopeOptions";

export type SearchComponent = "searchbox" | "presets" | "time-range" | "sort";

const props = withDefaults(
  defineProps<{
    params: SearchParams;
    overrideScopeIdList?: SearchScopeId[];
    components?: SearchComponent[];
    componentProps?: Partial<Record<SearchComponent, Record<string, unknown>>>;
    defaultParams?: SearchParams;
    orderBy?: string;
  }>(),
  {
    overrideScopeIdList: () => [],
    components: () => ["searchbox", "time-range", "presets"],
    componentProps: undefined,
    defaultParams: undefined,
    orderBy: "create_time desc",
  }
);

defineEmits<{
  (event: "update:params", params: SearchParams): void;
  (event: "update:orderBy", value: string): void;
}>();

const allowedScopes = computed((): SearchScopeId[] => {
  if (props.overrideScopeIdList && props.overrideScopeIdList.length > 0) {
    return props.overrideScopeIdList;
  }
  return [
    "creator",
    "current-approver",
    "approval",
    "status",
    "issue-type",
    "issue-label",
    "project",
    "risk-level",
  ];
});

const scopeOptions = useIssueSearchScopeOptions(
  computed(() => props.params),
  allowedScopes
);
</script>
