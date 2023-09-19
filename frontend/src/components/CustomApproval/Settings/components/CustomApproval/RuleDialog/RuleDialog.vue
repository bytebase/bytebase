<template>
  <BBModal
    v-if="dialog"
    :title="title"
    :esc-closable="false"
    :before-close="beforeClose"
    @close="dialog = undefined"
  >
    <RuleForm
      :dirty="state.dirty"
      @cancel="cancel"
      @update="state.dirty = true"
      @save="handleSave"
    />

    <div
      v-if="state.loading"
      class="absolute inset-0 flex flex-col items-center justify-center bg-white/50 rounded-lg"
    >
      <BBSpin />
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { useDialog } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { BBModal } from "@/bbkit";
import { pushNotification, useWorkspaceApprovalSettingStore } from "@/store";
import { LocalApprovalRule, SYSTEM_BOT_USER_NAME } from "@/types";
import { creatorOfRule, defer } from "@/utils";
import { useCustomApprovalContext } from "../context";
import RuleForm from "./RuleForm.vue";

type LocalState = {
  loading: boolean;
  dirty: boolean;
};

const { t } = useI18n();
const context = useCustomApprovalContext();
const { allowAdmin, dialog } = context;

const store = useWorkspaceApprovalSettingStore();
const state = reactive<LocalState>({
  loading: false,
  dirty: false,
});

const allowEditRule = computed(() => {
  if (!allowAdmin.value) return false;
  const rule = dialog.value?.rule;
  if (!rule) return false;
  return creatorOfRule(rule).name !== SYSTEM_BOT_USER_NAME;
});

const title = computed(() => {
  if (dialog.value) {
    if (!allowEditRule.value) {
      return t("custom-approval.approval-flow.view-approval-flow");
    }
    const { mode } = dialog.value;
    if (mode === "CREATE") {
      return t("custom-approval.approval-flow.create-approval-flow");
    }
    if (mode === "EDIT") {
      return t("custom-approval.approval-flow.edit-approval-flow");
    }
  }
  return "";
});

const nDialog = useDialog();

const cancel = async () => {
  const pass = await beforeClose();
  if (pass) {
    dialog.value = undefined;
  }
};

const beforeClose = async () => {
  if (!state.dirty) {
    return true;
  }
  if (!allowAdmin.value) {
    return true;
  }
  const d = defer<boolean>();
  nDialog.info({
    title: t("common.close"),
    content: t("common.will-lose-unsaved-data"),
    maskClosable: false,
    closeOnEsc: false,
    positiveText: t("common.confirm"),
    negativeText: t("common.cancel"),
    onPositiveClick: () => d.resolve(true),
    onNegativeClick: () => d.resolve(false),
  });
  return d.promise;
};

const handleSave = async (
  newRule: LocalApprovalRule,
  oldRule: LocalApprovalRule | undefined
) => {
  state.loading = true;
  try {
    await store.upsertRule(newRule, oldRule);
    state.dirty = false;
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    dialog.value = undefined;
  } catch {
    // nothing, exception has been handled already
  } finally {
    state.loading = false;
  }
};

watch(
  dialog,
  () => {
    state.dirty = false;
  },
  { immediate: true }
);
</script>
