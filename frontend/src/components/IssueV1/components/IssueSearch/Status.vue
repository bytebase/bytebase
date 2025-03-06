<template>
  <NTabs
    :value="tab"
    :type="'line'"
    :size="'small'"
    @update:value="updateStatus"
  >
    <NTab v-for="item in tabItemList" :key="item.value" :name="item.value">
      {{ item.label }}
    </NTab>
  </NTabs>
</template>

<script lang="ts" setup>
import { NTabs, NTab } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { SearchParams, SemanticIssueStatus } from "@/utils";
import { getSemanticIssueStatusFromSearchParams, upsertScope } from "@/utils";

const props = defineProps<{
  params: SearchParams;
}>();

const emit = defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const { t } = useI18n();

const tabItemList = computed(() => {
  return [
    {
      value: "OPEN",
      label: t("issue.table.open"),
    },
    {
      value: "CLOSED",
      label: t("issue.table.closed"),
    },
    {
      value: "",
      label: t("common.all"),
    },
  ] as {
    value: SemanticIssueStatus;
    label: string;
  }[];
});

const tab = computed(() => {
  return getSemanticIssueStatusFromSearchParams(props.params);
});

const updateStatus = (value: SemanticIssueStatus) => {
  if (!["", "OPEN", "CLOSED"].includes(value)) return;

  const updated = upsertScope({
    params: props.params,
    scopes: {
      id: "status",
      value: value,
    },
  });
  emit("update:params", updated);
};
</script>
