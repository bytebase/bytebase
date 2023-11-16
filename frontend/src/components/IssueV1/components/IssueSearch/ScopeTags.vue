<template>
  <NTag
    v-for="scope in params.scopes"
    :key="scope.id"
    :closable="true"
    :data-search-scope-id="scope.id"
    :bordered="false"
    size="small"
    style="--n-icon-size: 12px"
    v-bind="tagProps(scope)"
    @close="$emit('remove-scope', scope.id, scope.value)"
    @click="$emit('select-scope', scope.id, scope.value)"
  >
    <span>{{ scope.id }}</span>
    <span>:</span>
    <span>{{ scope.value }}</span>
  </NTag>
</template>
<script setup lang="ts">
import { NTag, TagProps } from "naive-ui";
import {
  SearchParams,
  SearchScope,
  SearchScopeId,
  callCssVariable,
} from "@/utils";

const props = defineProps<{
  params: SearchParams;
  focusedTagId?: SearchScopeId;
}>();
defineEmits<{
  (event: "remove-scope", id: SearchScopeId, value: string): void;
  (event: "select-scope", id: SearchScopeId, value: string): void;
}>();

const tagProps = (scope: SearchScope): TagProps => {
  if (props.focusedTagId !== scope.id) {
    return {};
  }
  return {
    bordered: true,
    color: {
      borderColor: callCssVariable("--color-accent"),
    },
  };
};
</script>
