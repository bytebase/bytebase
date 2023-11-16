<template>
  <NTag
    v-for="(scope, i) in params.scopes"
    :key="scope.id"
    :closable="true"
    :data-search-scope-id="scope.id"
    :bordered="false"
    size="small"
    style="--n-icon-size: 12px"
    v-bind="tagProps(scope)"
    @close="$emit('remove-scope', scope.id, scope.value)"
    @click.stop.prevent="$emit('select-scope', scope.id, scope.value)"
  >
    <span>{{ scope.id }}</span>
    <span>:</span>
    <component :is="() => renderValue(scope, i)" />
  </NTag>
</template>
<script setup lang="ts">
import dayjs from "dayjs";
import { NTag, TagProps } from "naive-ui";
import { h } from "vue";
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

const renderValue = (scope: SearchScope, index: number) => {
  if (scope.id === "created") {
    const [begin, end] = scope.value.split(",").map((ts) => parseInt(ts, 10));
    const format = "YYYY-MM-DD";
    return [dayjs(begin).format(format), dayjs(end).format(format)].join(
      " -> "
    );
  }
  return h("span", {}, scope.value);
};
</script>
