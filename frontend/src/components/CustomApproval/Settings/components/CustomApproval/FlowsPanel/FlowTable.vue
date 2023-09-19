<template>
  <BBGrid
    :column-list="COLUMN_LIST"
    :data-source="filteredApprovalRuleList"
    :row-clickable="false"
    :show-placeholder="true"
    row-key="uid"
    v-bind="$attrs"
  >
    <template #item="{ item: rule }: { item: LocalApprovalRule }">
      <div class="bb-grid-cell">
        {{ rule.template?.title }}
      </div>
      <div class="bb-grid-cell justify-center">
        <template v-if="creatorOfRule(rule).name === SYSTEM_BOT_USER_NAME">
          {{ $t("custom-approval.approval-flow.type.system") }}
        </template>
        <template v-else>
          {{ $t("custom-approval.approval-flow.type.custom") }}
        </template>
      </div>
      <div class="bb-grid-cell">
        {{ creatorOfRule(rule).title }}
      </div>
      <div class="bb-grid-cell justify-center">
        <NButton
          quaternary
          size="small"
          type="info"
          class="!rounded !w-[var(--n-height)] !p-0"
          @click="state.viewFlow = rule.template.flow"
        >
          {{ rule.template.flow?.steps.length }}
        </NButton>
      </div>
      <div class="bb-grid-cell">
        {{ rule.template?.description }}
      </div>
      <div class="bb-grid-cell gap-x-2">
        <template v-if="creatorOfRule(rule).name !== SYSTEM_BOT_USER_NAME">
          <NButton size="small" @click="editApprovalTemplate(rule)">
            {{ allowAdmin ? $t("common.edit") : $t("common.view") }}
          </NButton>
          <SpinnerButton
            size="small"
            :tooltip="$t('custom-approval.approval-flow.delete')"
            :disabled="!allowAdmin"
            :on-confirm="() => deleteRule(rule)"
          >
            {{ $t("common.delete") }}
          </SpinnerButton>
        </template>
      </div>
    </template>
  </BBGrid>

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

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { BBGrid, type BBGridColumn } from "@/bbkit";
import { pushNotification, useWorkspaceApprovalSettingStore } from "@/store";
import { LocalApprovalRule, SYSTEM_BOT_USER_NAME } from "@/types";
import { ApprovalFlow } from "@/types/proto/store/approval";
import { creatorOfRule } from "@/utils";
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

const COLUMN_LIST = computed(() => {
  const columns: BBGridColumn[] = [
    { title: t("common.name"), width: "1fr" },
    {
      title: t("common.type"),
      width: "8rem",
      class: "justify-center",
    },
    { title: t("common.creator"), width: "minmax(auto, 10rem)" },
    {
      title: t("custom-approval.approval-flow.approval-nodes"),
      width: "6rem",
      class: "justify-center text-center whitespace-pre-wrap capitalize",
    },
    { title: t("common.description"), width: "2fr" },
    {
      title: t("common.operations"),
      width: "10rem",
    },
  ];

  return columns;
});

const filteredApprovalRuleList = computed(() => {
  // const { searchText } = approvalConfigContext.value;
  const list = [...store.config.rules];
  // if (searchText) {
  //   list = list.filter((ap) => ap.template?.title.includes(searchText));
  // }
  return list;
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
