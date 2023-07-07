<template>
  <BBGrid
    :column-list="COLUMN_LIST"
    :data-source="riskList"
    :row-clickable="false"
    :show-placeholder="true"
    row-key="name"
    class="border"
  >
    <template #item="{ item: risk }: { item: Risk }">
      <div class="bb-grid-cell">
        {{ levelText(risk.level) }}
      </div>
      <div class="bb-grid-cell">
        {{ risk.title }}
      </div>
      <div class="bb-grid-cell justify-center">
        <SpinnerSwitch
          :value="risk.active"
          :disabled="!allowAdmin"
          :on-toggle="(active) => toggleRisk(risk, active)"
          size="small"
        />
      </div>
      <div class="bb-grid-cell gap-x-2">
        <NButton size="small" @click="editRisk(risk)">
          {{ allowAdmin ? $t("common.edit") : $t("common.view") }}
        </NButton>
        <SpinnerButton
          size="small"
          :tooltip="$t('custom-approval.security-rule.delete')"
          :disabled="!allowAdmin"
          :on-confirm="() => deleteRisk(risk)"
        >
          {{ $t("common.delete") }}
        </SpinnerButton>
      </div>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { NButton } from "naive-ui";

import { BBGrid, type BBGridColumn } from "@/bbkit";
import { SpinnerButton, SpinnerSwitch, levelText } from "../common";
import { useRiskCenterContext } from "./context";
import { Risk } from "@/types/proto/v1/risk_service";
import { pushNotification, useRiskStore } from "@/store";

defineProps<{
  riskList: Risk[];
}>();

const { t } = useI18n();
const context = useRiskCenterContext();
const { hasFeature, showFeatureModal, allowAdmin } = context;

const COLUMN_LIST = computed(() => {
  const columns: BBGridColumn[] = [
    {
      title: t("custom-approval.security-rule.risk.self"),
      width: "6rem",
    },
    { title: t("common.name"), width: "1fr" },
    {
      title: t("custom-approval.security-rule.active"),
      width: "6rem",
      class: "justify-center",
    },
    {
      title: t("common.operations"),
      width: "10rem",
    },
  ];

  return columns;
});

const editRisk = (risk: Risk) => {
  if (!hasFeature.value) {
    showFeatureModal.value = true;
    return;
  }
  context.dialog.value = {
    mode: "EDIT",
    risk,
  };
};

const toggleRisk = async (risk: Risk, active: boolean) => {
  if (!context.hasFeature.value) {
    context.showFeatureModal.value = true;
    return;
  }

  risk.active = active;
  await useRiskStore().upsertRisk(risk);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};

const deleteRisk = async (risk: Risk) => {
  if (!context.hasFeature.value) {
    context.showFeatureModal.value = true;
    return;
  }

  await useRiskStore().deleteRisk(risk);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.deleted"),
  });
};
</script>
