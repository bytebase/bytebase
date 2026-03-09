<template>
  <NTag
    v-for="({ scope, originalIndex }, visibleIndex) in visibleScopes"
    :key="`${originalIndex}-${scope.id}`"
    closable
    :data-search-scope-id="scope.id"
    :bordered="false"
    size="small"
    style="--n-icon-size: 12px"
    v-bind="tagProps(visibleIndex)"
    @close="$emit('remove-scope', originalIndex)"
    @click.stop.prevent="$emit('select-scope', scope)"
  >
    <div class="flex items-center gap-1">
      <span class="text-control">{{ scope.id }}:</span>
      <component :is="() => renderValue(scope)" />
    </div>
  </NTag>
</template>

<script setup lang="tsx">
import dayjs from "dayjs";
import type { TagProps } from "naive-ui";
import { NTag } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { UNKNOWN_ID } from "@/types";
import type { SearchParams, SearchScope } from "@/utils";
import { callCssVariable, extractDatabaseResourceName } from "@/utils";
import type { ScopeOption } from "./types";

const props = defineProps<{
  params: SearchParams;
  scopeOptions: ScopeOption[];
  focusedTagIndex?: number;
}>();

defineEmits<{
  (event: "remove-scope", index: number): void;
  (event: "select-scope", scope: SearchScope): void;
}>();

const { t } = useI18n();

const visibleScopes = computed(() => {
  return props.params.scopes
    .map((scope, originalIndex) => ({ scope, originalIndex }))
    .filter(({ scope }) => !scope.readonly);
});

const tagProps = (index: number): TagProps => {
  if (props.focusedTagIndex !== index) {
    return {};
  }
  return {
    bordered: true,
    color: {
      borderColor: callCssVariable("--color-accent"),
    },
  };
};

const renderValue = (scope: SearchScope) => {
  const scopeOption = props.scopeOptions
    .find((option) => option.id === scope.id)
    ?.options?.find((option) => option.value === scope.value);
  if (scopeOption && scopeOption.render) {
    return scopeOption.render();
  }
  if (scope.id === "created" || scope.id === "updated") {
    const [begin, end] = scope.value.split(",").map((ts) => parseInt(ts, 10));
    return [dayjs(begin).format("L"), dayjs(end).format("L")].join("-");
  } else if (scope.id === "database") {
    const { databaseName } = extractDatabaseResourceName(scope.value);
    return <span>{databaseName}</span>;
  }
  if (scope.value === `${UNKNOWN_ID}`) {
    return <span>{t("common.all").toLocaleLowerCase()}</span>;
  }
  return <span>{scope.value}</span>;
};
</script>
