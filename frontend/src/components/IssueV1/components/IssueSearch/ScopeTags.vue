<template>
  <NTag
    v-for="scope in params.scopes"
    :key="scope.id"
    :closable="true"
    size="small"
    style="--n-icon-size: 12px"
    :data-search-scope-id="scope.id"
    @close="removeScope(scope)"
    @click="$emit('select-scope', scope.id, scope.value)"
  >
    <span>{{ scope.id }}</span>
    <span>:</span>
    <span>{{ scope.value }}</span>
  </NTag>
</template>
<script setup lang="ts">
import { NTag } from "naive-ui";
import { SearchParams, SearchScope, SearchScopeId, upsertScope } from "@/utils";

const props = defineProps<{
  params: SearchParams;
}>();
const emit = defineEmits<{
  (event: "update:params", params: SearchParams): void;
  (event: "select-scope", id: SearchScopeId, value: string): void;
}>();

const removeScope = (scope: SearchScope) => {
  const updated = upsertScope(props.params, {
    id: scope.id,
    value: "",
  });
  emit("update:params", updated);
};
</script>
