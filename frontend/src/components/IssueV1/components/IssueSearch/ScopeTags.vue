<template>
  <NTag
    v-for="(scope, i) in params.scopes"
    :key="scope.id"
    :closable="!isReadonlyScope(scope)"
    :disabled="isReadonlyScope(scope)"
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
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { UNKNOWN_ID } from "@/types";
import {
  SearchParams,
  SearchScope,
  SearchScopeId,
  callCssVariable,
} from "@/utils";

const props = defineProps<{
  params: SearchParams;
  focusedTagId?: SearchScopeId;
  readonlyScopes?: SearchScope[];
}>();

defineEmits<{
  (event: "remove-scope", id: SearchScopeId, value: string): void;
  (event: "select-scope", id: SearchScopeId, value: string): void;
}>();

const { t } = useI18n();

const readonlyScopeIds = computed(() => {
  return new Set((props.readonlyScopes ?? []).map((s) => s.id));
});

const isReadonlyScope = (scope: SearchScope) => {
  return readonlyScopeIds.value.has(scope.id);
};

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
    return [dayjs(begin).format("L"), dayjs(end).format("L")].join("-");
  }
  if (scope.value === `${UNKNOWN_ID}`) {
    return h("span", {}, t("common.all").toLocaleLowerCase());
  }
  return h("span", {}, scope.value);
};
</script>
