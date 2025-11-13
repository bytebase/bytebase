<template>
  <div class="flex items-center gap-x-2">
    <NButtonGroup>
      <NButton
        v-for="preset in presets"
        :key="preset.value"
        :type="isActive(preset.value) ? 'primary' : 'default'"
        size="medium"
        @click="selectPreset(preset.value)"
      >
        {{ preset.label }}
      </NButton>
    </NButtonGroup>
  </div>
</template>

<script lang="ts" setup>
import { NButton, NButtonGroup } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useCurrentUserV1 } from "@/store";
import type { SearchParams } from "@/utils";
import { getValueFromSearchParams, getSemanticIssueStatusFromSearchParams, upsertScope } from "@/utils";

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

const isActive = (preset: PresetValue): boolean => {
  const myEmail = me.value.email;

  if (preset === "WAITING_APPROVAL") {
    return (
      getSemanticIssueStatusFromSearchParams(props.params) === "OPEN" &&
      getValueFromSearchParams(props.params, "approval") === "pending" &&
      getValueFromSearchParams(props.params, "approver") === myEmail &&
      props.params.scopes.filter((s) => !s.readonly).length === 3
    );
  }

  if (preset === "CREATED") {
    return (
      getValueFromSearchParams(props.params, "creator") === myEmail &&
      props.params.scopes.filter((s) => !s.readonly).length === 1
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
        { id: "status", value: "OPEN" },
        { id: "approval", value: "pending" },
        { id: "approver", value: myEmail },
      ],
    });
  } else if (preset === "CREATED") {
    newParams = upsertScope({
      params: newParams,
      scopes: { id: "creator", value: myEmail },
    });
  }
  // "ALL" preset keeps only readonly scopes (already done above)

  emit("update:params", newParams);
};
</script>
