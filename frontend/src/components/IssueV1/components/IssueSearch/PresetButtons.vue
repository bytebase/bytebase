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
import { getValueFromSearchParams, upsertScope } from "@/utils";

type PresetValue = "WAITING_APPROVAL" | "CREATED" | "ALL";

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
    value: "CREATED",
    label: t("common.created"),
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
  const myEmail = me.value.email;

  if (preset === "WAITING_APPROVAL") {
    return (
      getValueFromSearchParams(props.params, "approval") ===
      Issue_ApprovalStatus[Issue_ApprovalStatus.PENDING]
    );
  }

  if (preset === "CREATED") {
    return getValueFromSearchParams(props.params, "creator") === myEmail;
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
  } else if (preset === "CREATED") {
    newParams = upsertScope({
      params: newParams,
      scopes: [
        { id: "status", value: IssueStatus[IssueStatus.OPEN] },
        { id: "creator", value: myEmail },
      ],
    });
  }
  // "ALL" preset keeps only readonly scopes (already done above)

  emit("update:params", newParams);
};
</script>
