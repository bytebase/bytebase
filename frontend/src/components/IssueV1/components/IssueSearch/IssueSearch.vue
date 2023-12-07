<template>
  <div class="flex flex-col">
    <div
      v-if="components.includes('searchbox')"
      class="flex flex-row items-center gap-x-2"
    >
      <AdvancedSearchBox
        :params="params"
        :autofocus="autofocus"
        :readonly-scopes="readonlyScopes"
        :support-option-id-list="supportOptionIdList"
        class="flex-1"
        v-bind="componentProps?.searchbox"
        @update:params="$emit('update:params', $event)"
        @select-unsupported-scope="handleSelectScope"
      />
      <slot name="searchbox-suffix" />
    </div>
    <slot name="default" />

    <div class="flex flex-col md:flex-row md:items-center gap-y-1">
      <div class="flex-1 flex items-start">
        <Status
          v-if="components.includes('status')"
          :params="params"
          v-bind="componentProps?.status"
          @update:params="$emit('update:params', $event)"
        />
        <slot name="primary" />
      </div>
      <div class="flex flex-row space-x-4">
        <NInputGroup>
          <TimeRange
            v-if="components.includes('time-range')"
            v-model:show="showTimeRange"
            :params="params"
            v-bind="componentProps?.['time-range']"
            @update:params="$emit('update:params', $event)"
          />
          <slot name="secondary" />
        </NInputGroup>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";
import {
  SearchParams,
  SearchScope,
  SearchScopeId,
  UIIssueFilterScopeIdList,
  SearchScopeIdList,
} from "@/utils";
import AdvancedSearchBox from "./AdvancedSearchBox.vue";
import Status from "./Status.vue";
import TimeRange from "./TimeRange.vue";

export type SearchComponent =
  | "searchbox"
  | "status"
  | "type"
  | "time-range"
  | "project"
  | "instance"
  | "database"
  | "assignee"
  | "creator"
  | "approver"
  | "approval";

withDefaults(
  defineProps<{
    params: SearchParams;
    readonlyScopes?: SearchScope[];
    autofocus?: boolean;
    components?: SearchComponent[];
    componentProps?: Partial<Record<SearchComponent, any>>;
  }>(),
  {
    readonlyScopes: () => [],
    components: () => ["searchbox", "status", "time-range"],
    componentProps: undefined,
  }
);
defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const showTimeRange = ref(false);

const handleSelectScope = (id: SearchScopeId) => {
  if (id === "created") {
    showTimeRange.value = true;
  }
};

const supportOptionIdList = computed(() => [
  ...UIIssueFilterScopeIdList,
  ...SearchScopeIdList,
]);
</script>
