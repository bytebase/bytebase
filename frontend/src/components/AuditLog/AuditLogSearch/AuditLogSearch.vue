<template>
  <div class="flex flex-row items-center gap-x-2">
    <AdvancedSearchBox
      :params="params"
      :autofocus="autofocus"
      :readonly-scopes="readonlyScopes"
      :support-option-id-list="supportOptionIdList"
      class="flex-1"
      @update:params="$emit('update:params', $event)"
    />
    <TimeRange
      v-model:show="showTimeRange"
      :params="params"
      @update:params="$emit('update:params', $event)"
    />
    <slot name="searchbox-suffix" />
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import type { SearchParams, SearchScope } from "@/utils";
import { SearchScopeIdList } from "@/utils";
import AdvancedSearchBox from "./AdvancedSearchBox.vue";
import TimeRange from "./TimeRange.vue";

withDefaults(
  defineProps<{
    params: SearchParams;
    readonlyScopes?: SearchScope[];
    autofocus?: boolean;
  }>(),
  {
    readonlyScopes: () => [],
    components: () => ["searchbox", "time-range"],
    componentProps: undefined,
  }
);
defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const showTimeRange = ref(false);

const supportOptionIdList = computed(() => [...SearchScopeIdList]);
</script>
