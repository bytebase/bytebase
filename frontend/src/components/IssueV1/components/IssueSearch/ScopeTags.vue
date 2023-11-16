<template>
  <NTag
    v-for="scope in params.scopes"
    :key="scope.id"
    :closable="true"
    :data-search-scope-id="scope.id"
    :bordered="scope.id === focusedTagId"
    :color="
      scope.id === focusedTagId
        ? {
            borderColor: callCssVariable('--color-accent'),
          }
        : undefined
    "
    size="small"
    style="--n-icon-size: 12px"
    @close="$emit('remove-scope', scope.id, scope.value)"
    @click="$emit('select-scope', scope.id, scope.value)"
  >
    <span>{{ scope.id }}</span>
    <span>:</span>
    <span>{{ scope.value }}</span>
  </NTag>
</template>
<script setup lang="ts">
import { NTag } from "naive-ui";
import { SearchParams, SearchScopeId, callCssVariable } from "@/utils";

defineProps<{
  params: SearchParams;
  focusedTagId?: SearchScopeId;
}>();
defineEmits<{
  (event: "remove-scope", id: SearchScopeId, value: string): void;
  (event: "select-scope", id: SearchScopeId, value: string): void;
}>();
</script>
