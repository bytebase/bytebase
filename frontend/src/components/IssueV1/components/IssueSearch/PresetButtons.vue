<template>
  <div class="shrink-0">
    <NTabs
      :value="activePreset"
      :type="'line'"
      :size="'small'"
      @update:value="selectPreset"
    >
      <NTab v-for="preset in presets" :key="preset.value" :name="preset.value">
        {{ preset.label }}
      </NTab>
    </NTabs>
  </div>
</template>

<script lang="ts" setup>
import { NTab, NTabs } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useCurrentUserV1 } from "@/store";
import {
  Issue_ApprovalStatus,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { SearchParams } from "@/utils";
import {
  getValueFromSearchParams,
  getValuesFromSearchParams,
  upsertScope,
} from "@/utils";

type PresetValue = "WAITING_APPROVAL" | "OPEN" | "CLOSED" | "ALL";

interface Preset {
  value: PresetValue;
  label: string;
}

const props = defineProps<{
  params: SearchParams;
}>();

const emit = defineEmits<{
  (event: "update:params", params: SearchParams): void;
}>();

const { t } = useI18n();
const me = useCurrentUserV1();

const presets = computed((): Preset[] => [
  {
    value: "WAITING_APPROVAL",
    label: t("issue.waiting-approval"),
  },
  {
    value: "OPEN",
    label: t("issue.table.open"),
  },
  {
    value: "CLOSED",
    label: t("issue.table.closed"),
  },
  {
    value: "ALL",
    label: t("common.all"),
  },
]);

const activePreset = computed((): PresetValue | "" => {
  const preset = presets.value.find((p) => isActive(p.value));
  return preset?.value ?? "";
});

const isActive = (preset: PresetValue): boolean => {
  if (preset === "WAITING_APPROVAL") {
    return (
      getValueFromSearchParams(props.params, "approval") ===
      Issue_ApprovalStatus[Issue_ApprovalStatus.PENDING]
    );
  }

  if (preset === "OPEN") {
    const status = getValueFromSearchParams(props.params, "status");
    const approval = getValueFromSearchParams(props.params, "approval");
    return status === IssueStatus[IssueStatus.OPEN] && !approval;
  }

  if (preset === "CLOSED") {
    const statuses = getValuesFromSearchParams(props.params, "status");
    return (
      statuses.includes(IssueStatus[IssueStatus.DONE]) &&
      !statuses.includes(IssueStatus[IssueStatus.OPEN])
    );
  }

  if (preset === "ALL") {
    return props.params.scopes.filter((s) => !s.readonly).length === 0;
  }

  return false;
};

const selectPreset = (preset: PresetValue) => {
  const myEmail = me.value.email;
  const readonlyScopes = props.params.scopes.filter((s) => s.readonly);

  let newParams: SearchParams = {
    query: "",
    scopes: [...readonlyScopes],
  };

  if (preset === "WAITING_APPROVAL") {
    newParams = upsertScope({
      params: newParams,
      scopes: [
        { id: "status", value: IssueStatus[IssueStatus.OPEN] },
        {
          id: "approval",
          value: Issue_ApprovalStatus[Issue_ApprovalStatus.PENDING],
        },
        { id: "current-approver", value: myEmail },
      ],
    });
  } else if (preset === "OPEN") {
    newParams = upsertScope({
      params: newParams,
      scopes: [{ id: "status", value: IssueStatus[IssueStatus.OPEN] }],
    });
  } else if (preset === "CLOSED") {
    newParams = upsertScope({
      params: newParams,
      scopes: [
        { id: "status", value: IssueStatus[IssueStatus.DONE] },
        { id: "status", value: IssueStatus[IssueStatus.CANCELED] },
      ],
      allowMultiple: true,
    });
  }
  // "ALL" preset keeps only readonly scopes (already done above)

  emit("update:params", newParams);
};
</script>
