<template>
  <div v-if="ready" class="mb-2 space-y-2">
    <div
      class="flex flex-col md:flex-row items-end md:items-center gap-y-2 gap-x-2"
    >
      <AdvancedSearch
        v-if="supportOptionIdList.length > 0"
        class="flex-1 hidden md:block"
        :params="params"
        :placeholder="''"
        :scope-options="scopeOptions"
        :readonly-scopes="readonlyScopes"
        @update:params="$emit('update:params', $event)"
      />
      <TimeRange
        :params="params"
        @update:params="$emit('update:params', $event)"
      />
      <div class="flex items-center justify-end space-x-2">
        <slot name="suffix" />
      </div>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { computed } from "vue";
import AdvancedSearch, { TimeRange } from "@/components/AdvancedSearch";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import { useSlowQueryPolicyList } from "@/store";
import type { SearchParams, SearchScopeId, SearchScope } from "@/utils";

const props = withDefaults(
  defineProps<{
    params: SearchParams;
    readonlyScopes?: SearchScope[];
    supportOptionIdList: SearchScopeId[];
    loading?: boolean;
  }>(),
  {
    readonlyScopes: () => [],
    loading: false,
  }
);

defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const { ready } = useSlowQueryPolicyList();

const scopeOptions = useCommonSearchScopeOptions(
  computed(() => props.params),
  computed(() => props.supportOptionIdList)
);
</script>
