<template>
  <NDataTable
    size="small"
    :columns="columns"
    :data="riskList"
    :striped="true"
    :bordered="true"
  />
</template>

<script lang="tsx" setup>
import { NButton, NDataTable } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { SpinnerButton, SpinnerSwitch } from "@/components/v2/Form";
import { pushNotification, useRiskStore } from "@/store";
import type { Risk } from "@/types/proto/v1/risk_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import { levelText } from "../common";
import { useRiskCenterContext } from "./context";

defineProps<{
  riskList: Risk[];
}>();

const { t } = useI18n();
const context = useRiskCenterContext();
const { hasFeature, showFeatureModal } = context;

const columns = computed((): DataTableColumn<Risk>[] => {
  return [
    {
      title: t("custom-approval.risk-rule.risk.self"),
      key: "level",
      width: 120,
      render: (risk) => levelText(risk.level),
    },
    {
      title: t("common.name"),
      key: "title",
    },
    {
      title: t("custom-approval.risk-rule.active"),
      key: "active",
      width: 120,
      align: "center",
      render: (risk) => (
        <SpinnerSwitch
          value={risk.active}
          disabled={!allowUpdateRisk.value}
          onToggle={(active: boolean) => toggleRisk(risk, active)}
          size="small"
        />
      ),
    },
    {
      title: t("common.operations"),
      key: "operations",
      width: 200,
      render: (risk) => (
        <div class="flex gap-x-2">
          <NButton size="small" onClick={() => editRisk(risk)}>
            {allowUpdateRisk.value ? t("common.edit") : t("common.view")}
          </NButton>
          {allowDeleteRisk.value && (
            <SpinnerButton
              size="small"
              tooltip={t("custom-approval.risk-rule.delete")}
              onConfirm={() => deleteRisk(risk)}
            >
              {t("common.delete")}
            </SpinnerButton>
          )}
        </div>
      ),
    },
  ];
});

const allowUpdateRisk = computed(() => {
  return hasWorkspacePermissionV2("bb.risks.update");
});

const allowDeleteRisk = computed(() => {
  return hasWorkspacePermissionV2("bb.risks.delete");
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
