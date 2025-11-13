<template>
  <NDropdown
    :options="dropdownOptions"
    :show="showDropdown"
    placement="bottom-start"
    @select="handleSelect"
    @clickoutside="showDropdown = false"
  >
    <NButton
      :type="isActive ? 'primary' : 'default'"
      size="medium"
      @click="showDropdown = !showDropdown"
    >
      {{ buttonLabel }}
      <template #icon>
        <ChevronDownIcon class="w-4 h-4" />
      </template>
    </NButton>
  </NDropdown>
</template>

<script lang="ts" setup>
import { ChevronDownIcon } from "lucide-vue-next";
import { NButton, NDropdown } from "naive-ui";
import { computed, ref } from "vue";
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
const showDropdown = ref(false);

const currentStatus = computed(() => {
  return getSemanticIssueStatusFromSearchParams(props.params);
});

const isActive = computed(() => {
  return currentStatus.value === "OPEN" || currentStatus.value === "CLOSED";
});

const buttonLabel = computed(() => {
  if (currentStatus.value === "OPEN") {
    return `${t("common.status")}: ${t("issue.table.open")}`;
  }
  if (currentStatus.value === "CLOSED") {
    return `${t("common.status")}: ${t("issue.table.closed")}`;
  }
  return `${t("common.status")} ▾`;
});

const dropdownOptions = computed(() => {
  return [
    {
      key: "OPEN",
      label: t("issue.table.open"),
      type: "checkbox",
      checked: currentStatus.value === "OPEN",
    },
    {
      key: "CLOSED",
      label: t("issue.table.closed"),
      type: "checkbox",
      checked: currentStatus.value === "CLOSED",
    },
  ];
});

const handleSelect = (key: string) => {
  const newStatus = key as SemanticIssueStatus;

  // Mutex behavior: if clicking the currently selected status, deselect it
  if (currentStatus.value === newStatus) {
    // Remove status scope to show "all"
    const updated = {
      ...props.params,
      scopes: props.params.scopes.filter((s) => s.id !== "status"),
    };
    emit("update:params", updated);
  } else {
    // Select the new status
    const updated = upsertScope({
      params: props.params,
      scopes: { id: "status", value: newStatus },
    });
    emit("update:params", updated);
  }

  showDropdown.value = false;
};
</script>
