<template>
  <div class="shrink-0">
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
  </div>
</template>

<script lang="ts" setup>
import { NTab, NTabs } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { SearchParams, SearchScope } from "@/utils";
import { getValuesFromSearchParams, upsertScope } from "@/utils";

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
      value: IssueStatus.OPEN,
      label: t("issue.table.open"),
    },
    {
      value: IssueStatus.DONE,
      label: t("issue.table.closed"),
    },
  ] as {
    value: IssueStatus;
    label: string;
  }[];
});

const tab = computed((): IssueStatus => {
  const statusList = getValuesFromSearchParams(props.params, "status").map(
    (status) => IssueStatus[status as keyof typeof IssueStatus]
  );

  switch (statusList.length) {
    case 0:
      return IssueStatus.ISSUE_STATUS_UNSPECIFIED;
    case 1:
      if (statusList[0] === IssueStatus.OPEN) {
        return IssueStatus.OPEN;
      }
      return IssueStatus.DONE;
    default:
      if (!statusList.includes(IssueStatus.OPEN)) {
        return IssueStatus.DONE;
      }
      return IssueStatus.ISSUE_STATUS_UNSPECIFIED;
  }
});

const updateStatus = (value: IssueStatus) => {
  const scopes: SearchScope[] = [];
  let allowMultiple = false;
  if (value === IssueStatus.OPEN) {
    scopes.push({
      id: "status",
      value: IssueStatus[IssueStatus.OPEN],
    });
  } else {
    scopes.push(
      {
        id: "status",
        value: IssueStatus[IssueStatus.DONE],
      },
      {
        id: "status",
        value: IssueStatus[IssueStatus.CANCELED],
      }
    );
    allowMultiple = true;
  }

  const updated = upsertScope({
    params: {
      ...props.params,
      scopes: props.params.scopes.filter((scope) => scope.id !== "status"),
    },
    scopes,
    allowMultiple,
  });
  emit("update:params", updated);
};
</script>
