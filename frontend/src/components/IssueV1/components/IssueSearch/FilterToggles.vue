<template>
  <div class="flex items-center gap-x-2">
    <StatusDropdown
      :params="params"
      @update:params="$emit('update:params', $event)"
    />
    <span class="text-control-border">|</span>
    <FilterDropdown
      v-for="filter in filters"
      :key="filter.scopeId"
      :scope-id="filter.scopeId"
      :label="filter.label"
      :options="filter.options"
      :params="params"
      :multiple="filter.multiple"
      @select="handleFilterSelect(filter.scopeId, $event)"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useCurrentUserV1 } from "@/store";
import type { SearchParams, SearchScopeId } from "@/utils";
import { upsertScope } from "@/utils";
import FilterDropdown from "./FilterDropdown.vue";
import StatusDropdown from "./StatusDropdown.vue";

const props = defineProps<{
  params: SearchParams;
}>();

const emit = defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const { t } = useI18n();
const me = useCurrentUserV1();

const filters = computed(() => [
  {
    scopeId: "creator" as SearchScopeId,
    label: t("common.creator"),
    options: [
      { value: me.value.email, label: t("common.me") },
    ],
    multiple: false,
  },
  {
    scopeId: "assignee" as SearchScopeId,
    label: t("common.assignee"),
    options: [
      { value: me.value.email, label: t("common.me") },
    ],
    multiple: false,
  },
]);

const handleFilterSelect = (scopeId: SearchScopeId, value: string) => {
  const currentValue = props.params.scopes.find((s) => s.id === scopeId)?.value;

  // Toggle behavior: if clicking current value, remove it
  if (currentValue === value) {
    const updated = {
      ...props.params,
      scopes: props.params.scopes.filter((s) => s.id !== scopeId),
    };
    emit("update:params", updated);
  } else {
    const updated = upsertScope({
      params: props.params,
      scopes: { id: scopeId, value },
    });
    emit("update:params", updated);
  }
};
</script>
