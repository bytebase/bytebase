<template>
  <div class="flex flex-col gap-y-2">
    <div class="flex items-center justify-between">
      <div class="font-medium text-base">
        {{ sourceDisplayText }}
      </div>
      <div class="flex items-center gap-x-2">
        <NButton size="small" @click="handleAddRule">
          <template #icon>
            <PlusIcon class="w-4" />
          </template>
          {{ $t("custom-approval.approval-flow.add-rule") }}
        </NButton>
        <NTooltip>
          <template #trigger>
            <HelpCircleIcon class="w-4 h-4 text-control-light cursor-help" />
          </template>
          {{ $t("custom-approval.rule.first-match-wins") }}
        </NTooltip>
      </div>
    </div>

    <NDataTable
      size="small"
      :columns="columns"
      :data="rules"
      :striped="true"
      :bordered="true"
      :row-key="(row: LocalApprovalRule) => row.uid"
    />

    <RuleEditModal
      v-model:show="showModal"
      :mode="modalMode"
      :source="source"
      :rule="editingRule"
      @save="handleSaveRule"
    />
  </div>
</template>

<script lang="tsx" setup>
import {
  HelpCircleIcon,
  PencilIcon,
  PlusIcon,
  TrashIcon,
} from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NButton, NDataTable, NPopconfirm, NTooltip } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useWorkspaceApprovalSettingStore } from "@/store";
import type { LocalApprovalRule } from "@/types";
import type { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import { formatApprovalFlow } from "@/utils";
import { approvalSourceText } from "../../common/utils";
import { useCustomApprovalContext } from "../context";
import RuleEditModal from "./RuleEditModal.vue";

const props = defineProps<{
  source: WorkspaceApprovalSetting_Rule_Source;
}>();

const { t } = useI18n();
const store = useWorkspaceApprovalSettingStore();
const context = useCustomApprovalContext();

const showModal = ref(false);
const modalMode = ref<"create" | "edit">("create");
const editingRule = ref<LocalApprovalRule | undefined>();

const rules = computed(() => store.getRulesBySource(props.source));

const sourceDisplayText = computed(() => approvalSourceText(props.source));

const columns = computed((): DataTableColumn<LocalApprovalRule>[] => [
  {
    title: t("common.title"),
    key: "title",
    width: 200,
    ellipsis: { tooltip: true },
    render: (row) => row.title || "-",
  },
  {
    title: t("cel.condition.self"),
    key: "condition",
    ellipsis: { tooltip: true },
    render: (row) => (
      <code class="text-xs bg-control-bg px-1 py-0.5 rounded">
        {row.condition || "true"}
      </code>
    ),
  },
  {
    title: t("custom-approval.approval-flow.self"),
    key: "flow",
    width: 280,
    render: (row) => formatApprovalFlow(row.flow),
  },
  {
    title: t("common.operations"),
    key: "operations",
    width: 100,
    render: (row) => (
      <div class="flex gap-x-1">
        <NButton size="tiny" onClick={() => handleEditRule(row)}>
          <PencilIcon class="w-3" />
        </NButton>
        <NPopconfirm
          onPositiveClick={() => handleDeleteRule(row)}
          positiveText={t("common.confirm")}
          negativeText={t("common.cancel")}
        >
          {{
            trigger: () => (
              <NButton size="tiny">
                <TrashIcon class="w-3" />
              </NButton>
            ),
            default: () => t("common.confirm") + "?",
          }}
        </NPopconfirm>
      </div>
    ),
  },
]);

const handleAddRule = () => {
  if (!context.hasFeature.value) {
    context.showFeatureModal.value = true;
    return;
  }
  modalMode.value = "create";
  editingRule.value = undefined;
  showModal.value = true;
};

const handleEditRule = (rule: LocalApprovalRule) => {
  if (!context.hasFeature.value) {
    context.showFeatureModal.value = true;
    return;
  }
  modalMode.value = "edit";
  editingRule.value = rule;
  showModal.value = true;
};

const handleDeleteRule = async (rule: LocalApprovalRule) => {
  if (!context.hasFeature.value) {
    context.showFeatureModal.value = true;
    return;
  }
  try {
    await store.deleteRule(rule.uid);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.deleted"),
    });
  } catch {
    // Error handled by store
  }
};

const handleSaveRule = async (ruleData: Partial<LocalApprovalRule>) => {
  try {
    if (modalMode.value === "create") {
      await store.addRule(ruleData as Omit<LocalApprovalRule, "uid">);
    } else if (editingRule.value && ruleData.uid) {
      await store.updateRule(ruleData.uid, ruleData);
    }
    showModal.value = false;
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } catch {
    // Error handled by store
  }
};
</script>
