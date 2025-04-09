<template>
  <div class="flex flex-col gap-y-1">
    <div
      v-if="components.includes('searchbox')"
      class="flex flex-row items-center gap-x-2"
    >
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

    <div
      v-if="!componentProps?.status?.hidden"
      class="flex flex-col md:flex-row md:items-center gap-y-1"
    >
      <div class="flex-1 flex items-start w-full">
        <Status
          v-if="components.includes('status')"
          :params="params"
          v-bind="componentProps?.status"
          @update:params="$emit('update:params', $event)"
        />
        <slot name="primary" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import AdvancedSearch from "@/components/AdvancedSearch";
import TimeRange from "@/components/AdvancedSearch/TimeRange.vue";
import type { SearchParams, SearchScopeId } from "@/utils";
import { UIIssueFilterScopeIdList, useSearchScopeIdList } from "@/utils";
import Status from "./Status.vue";
import { useIssueSearchScopeOptions } from "./useIssueSearchScopeOptions";

export type SearchComponent = "searchbox" | "status" | "time-range";

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
    components: () => ["searchbox", "time-range", "status"],
    componentProps: undefined,
  }
);
defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const SearchScopeIdList = useSearchScopeIdList();
const showTimeRange = ref(false);

const allowedScopes = computed(() => {
  if (props.overrideScopeIdList && props.overrideScopeIdList.length > 0) {
    return props.overrideScopeIdList;
  }
  return [...UIIssueFilterScopeIdList, ...SearchScopeIdList.value];
});

const scopeOptions = useIssueSearchScopeOptions(
  computed(() => props.params),
  allowedScopes
);
</script>
