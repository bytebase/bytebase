<template>
  <div class="flex flex-col">
    <AdvancedSearchBox
      v-if="components.includes('searchbox')"
      :params="params"
      :autofocus="autofocus"
      v-bind="componentProps?.searchbox"
      @update:params="$emit('update:params', $event)"
    />
    <slot name="default" />

    <template v-if="showFeatureAttention">
      <FeatureAttention
        v-if="!!params.query || params.scopes.length > 0"
        feature="bb.feature.issue-advanced-search"
      />
    </template>
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
import { SearchParams } from "@/utils";
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
    autofocus?: boolean;
    components?: SearchComponent[];
    showFeatureAttention?: boolean;
    componentProps?: Partial<Record<SearchComponent, any>>;
  }>(),
  {
    components: () => ["searchbox", "status", "time-range"],
    showFeatureAttention: false,
    componentProps: undefined,
  }
);

defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();
</script>
