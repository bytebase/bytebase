<template>
  <NTag
    v-for="(scope, i) in params.scopes"
    :key="`${i}-${scope.id}`"
    :closable="!scope.readonly"
    :disabled="scope.readonly"
    :data-search-scope-id="scope.id"
    :bordered="false"
    size="small"
    style="--n-icon-size: 12px"
    v-bind="tagProps(scope)"
    @close="$emit('remove-scope', scope.id, scope.value)"
    @click.stop.prevent="handleClick(scope)"
  >
    <div class="flex items-center gap-1">
      <span class="text-control">{{ scope.id }}:</span>
      <component :is="() => renderValue(scope, i)" />
    </div>
  </NTag>
</template>

<script setup lang="tsx">
import dayjs from "dayjs";
import type { TagProps } from "naive-ui";
import { NTag } from "naive-ui";
import { useI18n } from "vue-i18n";
import { UNKNOWN_ID } from "@/types";
import type { SearchParams, SearchScope, SearchScopeId } from "@/utils";
import { callCssVariable, extractDatabaseResourceName } from "@/utils";
import type { ScopeOption } from "./types";

const props = defineProps<{
  params: SearchParams;
  scopeOptions: ScopeOption[];
  focusedTagId?: SearchScopeId;
}>();

const emit = defineEmits<{
  (event: "remove-scope", id: SearchScopeId, value: string): void;
  (event: "select-scope", id: SearchScopeId, value: string): void;
}>();

const { t } = useI18n();

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

const renderValue = (scope: SearchScope, _index: number) => {
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

const handleClick = (scope: SearchScope) => {
  if (scope.readonly) {
    return;
  }

  emit("select-scope", scope.id, scope.value);
};
</script>
