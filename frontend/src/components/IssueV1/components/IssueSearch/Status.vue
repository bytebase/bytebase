<template>
  <TabFilter
    :value="tab"
    :items="tabItemList"
    :responsive="false"
    @update:value="updateStatus"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { TabFilter, TabFilterItem } from "@/components/v2";
import {
  SearchParams,
  SemanticIssueStatus,
  getSemanticIssueStatusFromSearchParams,
} from "@/utils";
import { upsertScope } from "./utils";

const props = defineProps<{
  params: SearchParams;
}>();

const emit = defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const { t } = useI18n();

const tabItemList = computed(() => {
  const OPEN: TabFilterItem<SemanticIssueStatus> = {
    value: "OPEN",
    label: t("issue.table.open"),
  };
  const CLOSED: TabFilterItem<SemanticIssueStatus> = {
    value: "CLOSED",
    label: t("issue.table.closed"),
  };
  return [OPEN, CLOSED];
});

const tab = computed(() => {
  return getSemanticIssueStatusFromSearchParams(props.params);
});

const updateStatus = (value: string | number | undefined) => {
  const status = value as SemanticIssueStatus;
  if (!["OPEN", "CLOSED"].includes(status)) return;

  const updated = upsertScope(props.params, {
    id: "status",
    value: status,
  });
  emit("update:params", updated);
};
</script>
