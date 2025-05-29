<template>
  <NDataTable
    size="small"
    :columns="columns"
    :data="store.config.rules"
    :striped="true"
    :bordered="true"
    :row-key="(rule: LocalApprovalRule) => rule.uid"
  />

  <BBModal
    v-if="state.viewFlow"
    :title="$t('custom-approval.approval-flow.approval-nodes')"
    @close="state.viewFlow = undefined"
  >
    <div class="w-[20rem]">
      <StepsTable :flow="state.viewFlow" :editable="false" />
    </div>
  </BBModal>
</template>

<script lang="tsx" setup>
import { NButton, NDataTable } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { BBModal } from "@/bbkit";
import { pushNotification, useWorkspaceApprovalSettingStore } from "@/store";
import type { LocalApprovalRule } from "@/types";
import type { ApprovalFlow } from "@/types/proto/v1/issue_service";
import { SpinnerButton } from "../../common";
import { StepsTable } from "../common";
import { useCustomApprovalContext } from "../context";

type LocalState = {
  viewFlow: ApprovalFlow | undefined;
};

const state = reactive<LocalState>({
  viewFlow: undefined,
});
const { t } = useI18n();
const store = useWorkspaceApprovalSettingStore();
const context = useCustomApprovalContext();
const { hasFeature, showFeatureModal, allowAdmin, dialog } = context;

const columns = computed((): DataTableColumn<LocalApprovalRule>[] => {
  return [
    {
      title: t("common.name"),
      key: "name",
      render: (rule) => (
        <div class="whitespace-nowrap">{rule.template?.title}</div>
      ),
    },
    {
      title: t("custom-approval.approval-flow.approval-nodes"),
      key: "nodes",
      width: 120,
      align: "center",
      render: (rule) => (
        <NButton
          quaternary
          size="small"
          type="info"
          class="!rounded !w-[var(--n-height)] !p-0"
          onClick={() => (state.viewFlow = rule.template.flow)}
        >
          {rule.template.flow?.steps.length}
        </NButton>
      ),
    },
    {
      title: t("common.description"),
      key: "description",
      render: (rule) => rule.template?.description,
    },
    {
      title: t("common.operations"),
      key: "operations",
      width: 200,
      render: (rule) => (
        <div class="flex gap-x-2">
          <NButton size="small" onClick={() => editApprovalTemplate(rule)}>
            {allowAdmin.value ? t("common.edit") : t("common.view")}
          </NButton>
          <SpinnerButton
            size="small"
            tooltip={t("custom-approval.approval-flow.delete")}
            disabled={!allowAdmin.value}
            onConfirm={() => deleteRule(rule)}
          >
            {t("common.delete")}
          </SpinnerButton>
        </div>
      ),
    },
  ];
});

const editApprovalTemplate = (rule: LocalApprovalRule) => {
  if (!hasFeature.value) {
    showFeatureModal.value = true;
    return;
  }
  dialog.value = {
    mode: "EDIT",
    rule,
  };
};

const deleteRule = async (rule: LocalApprovalRule) => {
  try {
    await store.deleteRule(rule);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.deleted"),
    });
  } catch {
    // nothing, exception has been handled already
  }
};
</script>
