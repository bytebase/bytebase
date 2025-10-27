<template>
  <div class="space-y-6">
    <!-- Built-in Flows Section -->
    <div class="space-y-2">
      <div>
        <h3 class="text-base font-medium">
          {{ $t("custom-approval.approval-flow.built-in") }}
        </h3>
        <span class="textinfolabel">
          {{ $t("custom-approval.approval-flow.built-in-description") }}
        </span>
      </div>
      <NDataTable
        size="small"
        :columns="builtinColumns"
        :data="builtinFlows"
        :striped="true"
        :bordered="true"
        :row-key="(flow: BuiltinApprovalFlow) => flow.id"
      />
    </div>

    <!-- Custom Flows Section -->
    <div class="space-y-2">
      <h3 class="text-base font-medium">
        {{ $t("custom-approval.approval-flow.custom") }}
      </h3>
      <NDataTable
        v-if="customRules.length > 0"
        size="small"
        :columns="customColumns"
        :data="customRules"
        :striped="true"
        :bordered="true"
        :row-key="(rule: LocalApprovalRule) => rule.template.id"
      />
      <div v-else class="text-sm text-control-light py-4 text-center">
        {{ $t("custom-approval.approval-flow.no-custom-flows") }}
      </div>
    </div>
  </div>

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
import { create as createProto } from "@bufbuild/protobuf";
import { NButton, NDataTable } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { BBModal } from "@/bbkit";
import { SpinnerButton } from "@/components/v2/Form";
import { pushNotification, useWorkspaceApprovalSettingStore } from "@/store";
import {
  type LocalApprovalRule,
  BUILTIN_APPROVAL_FLOWS,
  type BuiltinApprovalFlow,
  isBuiltinFlowId,
} from "@/types";
import type { ApprovalFlow } from "@/types/proto-es/v1/issue_service_pb";
import { ApprovalFlowSchema } from "@/types/proto-es/v1/issue_service_pb";
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

const builtinFlows = computed(() => [...BUILTIN_APPROVAL_FLOWS]);

const customRules = computed(() => {
  return store.config.rules.filter(
    (rule) => !isBuiltinFlowId(rule.template.id)
  );
});

// Helper: View flow details in modal
const viewFlow = (flow: ApprovalFlow) => {
  state.viewFlow = flow;
};

// Helper: Convert BuiltinApprovalFlow to ApprovalFlow proto for viewing
const viewBuiltinFlow = (builtinFlow: BuiltinApprovalFlow) => {
  state.viewFlow = createProto(ApprovalFlowSchema, {
    roles: [...builtinFlow.roles],
  });
};

// Helper: Render approval nodes button
const renderNodesButton = (rolesCount: number, onClick: () => void) => (
  <NButton
    quaternary
    size="small"
    type="info"
    class="!rounded !w-[var(--n-height)] !p-0"
    onClick={onClick}
  >
    {rolesCount}
  </NButton>
);

// Shared column definitions
const createNameColumn = <T extends { title: string }>(
  key: string = "name"
): DataTableColumn<T> => ({
  title: t("common.name"),
  key,
  render: (item) => <div class="whitespace-nowrap">{item.title}</div>,
});

const createDescriptionColumn = <T extends { description: string }>(
  key: string = "description"
): DataTableColumn<T> => ({
  title: t("common.description"),
  key,
  render: (item) => item.description,
});

// Columns for built-in flows (read-only)
const builtinColumns = computed((): DataTableColumn<BuiltinApprovalFlow>[] => {
  return [
    createNameColumn<BuiltinApprovalFlow>(),
    {
      title: t("custom-approval.approval-flow.approval-nodes"),
      key: "nodes",
      width: 120,
      align: "center",
      render: (flow) =>
        renderNodesButton(flow.roles.length, () => viewBuiltinFlow(flow)),
    },
    createDescriptionColumn<BuiltinApprovalFlow>(),
    {
      title: t("common.operations"),
      key: "operations",
      width: 200,
      render: (flow) => (
        <div class="flex gap-x-2">
          <NButton size="small" onClick={() => viewBuiltinFlow(flow)}>
            {t("common.view")}
          </NButton>
        </div>
      ),
    },
  ];
});

// Columns for custom flows (editable)
const customColumns = computed((): DataTableColumn<LocalApprovalRule>[] => {
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
      render: (rule) =>
        renderNodesButton(rule.template.flow?.roles.length ?? 0, () =>
          viewFlow(rule.template.flow!)
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
          {allowAdmin.value && (
            <SpinnerButton
              size="small"
              type="error"
              tooltip={t("custom-approval.approval-flow.delete")}
              onConfirm={() => deleteRule(rule)}
            >
              {t("common.delete")}
            </SpinnerButton>
          )}
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
