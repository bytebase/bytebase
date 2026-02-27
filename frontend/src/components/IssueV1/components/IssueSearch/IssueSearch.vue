<template>
  <div class="flex flex-col gap-y-2">
    <!-- Advanced Search Bar - Always Visible -->
    <div class="flex flex-row items-center gap-x-2">
      <AdvancedSearch
        class="flex-1"
        :params="params"
        :scope-options="scopeOptions"
        :default-params="defaultParams"
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

    <!-- Preset Buttons -->
    <PresetButtons
      v-if="components.includes('presets') && !componentProps?.presets?.hidden"
      :params="params"
      @update:params="$emit('update:params', $event)"
    />

    <slot name="primary" />
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import AdvancedSearch from "@/components/AdvancedSearch";
import TimeRange from "@/components/AdvancedSearch/TimeRange.vue";
import type { SearchParams, SearchScopeId } from "@/utils";

import PresetButtons from "./PresetButtons.vue";
import { useIssueSearchScopeOptions } from "./useIssueSearchScopeOptions";

export type SearchComponent = "searchbox" | "presets" | "time-range";

const props = withDefaults(
  defineProps<{
    params: SearchParams;
    overrideScopeIdList?: SearchScopeId[];
    autofocus?: boolean;
    components?: SearchComponent[];
    componentProps?: Partial<Record<SearchComponent, Record<string, unknown>>>;
    defaultParams?: SearchParams;
  }>(),
  {
    overrideScopeIdList: () => [],
    components: () => ["searchbox", "time-range", "presets"],
    componentProps: undefined,
    defaultParams: undefined,
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
    "creator",
    "current-approver",
    "approval",
    "status",
    "issue-label",
    "project",
  ];
});

const scopeOptions = useIssueSearchScopeOptions(
  computed(() => props.params),
  allowedScopes
);
</script>
