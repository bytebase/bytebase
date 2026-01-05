<template>
  <div class="flex flex-col gap-y-2">
    <div class="flex items-center justify-between">
      <div class="font-medium text-base">
        {{ sourceDisplayText }}
      </div>
      <PermissionGuardWrapper
        v-slot="slotProps"
        :permissions="['bb.settings.set']"
      >
        <NButton size="small" :disabled="slotProps.disabled" @click="handleAddRule">
          <template #icon>
            <PlusIcon class="w-4" />
          </template>
          {{ $t("custom-approval.approval-flow.add-rule") }}
        </NButton>
      </PermissionGuardWrapper>
    </div>

    <div class="rules-table border border-gray-200 rounded-sm text-sm">
      <!-- Table Header -->
      <div
        class="rules-table-header grid bg-gray-50 border-b border-gray-200 font-medium text-gray-600"
      >
        <div class="px-2 py-2 w-10"></div>
        <div class="px-3 py-2">{{ $t("common.title") }}</div>
        <div class="px-3 py-2">{{ $t("cel.condition.self") }}</div>
        <div class="px-3 py-2">{{ $t("custom-approval.approval-flow.self") }}</div>
        <div class="px-3 py-2 w-24">{{ $t("common.operations") }}</div>
      </div>

      <!-- Draggable Body -->
      <Draggable
        v-model="localRules"
        item-key="uid"
        handle=".drag-handle"
        animation="150"
        ghost-class="rules-row-ghost"
        @end="handleDragEnd"
      >
        <template #item="{ element: rule, index }">
          <div
            class="rules-table-row grid border-b border-gray-100 last:border-b-0 hover:bg-gray-50"
            :class="{ 'bg-gray-50/50': index % 2 === 1 }"
          >
            <div class="px-2 py-2 w-10 flex items-center justify-center">
              <GripVerticalIcon
                class="drag-handle w-4 h-4 text-gray-400 cursor-grab active:cursor-grabbing"
              />
            </div>
            <div class="px-3 py-2 truncate" :title="rule.title">
              {{ rule.title || "-" }}
            </div>
            <div class="px-3 py-2 truncate">
              <code class="text-xs">{{ rule.condition || "true" }}</code>
            </div>
            <div class="px-3 py-2 truncate">
              {{ formatApprovalFlow(rule.flow) }}
            </div>
            <div class="px-3 py-2 w-24 flex items-center gap-x-1">
              <MiniActionButton @click="handleEditRule(rule)">
                <PencilIcon class="w-3" />
              </MiniActionButton>
              <NPopconfirm
                v-if="allowAdmin"
                :positive-text="$t('common.confirm')"
                :negative-text="$t('common.cancel')"
                @positive-click="handleDeleteRule(rule)"
              >
                <template #trigger>
                  <MiniActionButton>
                    <TrashIcon class="w-3" />
                  </MiniActionButton>
                </template>
                {{ $t("common.confirm") }}?
              </NPopconfirm>
            </div>
          </div>
        </template>
      </Draggable>

      <!-- Empty State -->
      <div
        v-if="localRules.length === 0"
        class="px-3 py-4 text-center text-gray-400"
      >
        {{ $t("common.no-data") }}
      </div>
    </div>

    <RuleEditModal
      v-model:show="showModal"
      :mode="modalMode"
      :source="source"
      :rule="editingRule"
      :is-fallback="isFallback"
      @save="handleSaveRule"
    />
  </div>
</template>

<script lang="ts" setup>
import {
  GripVerticalIcon,
  PencilIcon,
  PlusIcon,
  TrashIcon,
} from "lucide-vue-next";
import { NButton, NPopconfirm } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import Draggable from "vuedraggable";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { MiniActionButton } from "@/components/v2";
import { pushNotification, useWorkspaceApprovalSettingStore } from "@/store";
import type { LocalApprovalRule } from "@/types";
import { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import { formatApprovalFlow } from "@/utils";
import { approvalSourceText } from "../../common/utils";
import { useCustomApprovalContext } from "../context";
import RuleEditModal from "./RuleEditModal.vue";

const props = defineProps<{
  source: WorkspaceApprovalSetting_Rule_Source;
}>();

const isFallback = computed(
  () => props.source === WorkspaceApprovalSetting_Rule_Source.SOURCE_UNSPECIFIED
);

const { t } = useI18n();
const store = useWorkspaceApprovalSettingStore();
const context = useCustomApprovalContext();
const { allowAdmin } = context;

const showModal = ref(false);
const modalMode = ref<"create" | "edit">("create");
const editingRule = ref<LocalApprovalRule | undefined>();

const rules = computed(() => store.getRulesBySource(props.source));

// Local copy for dragging - synced with store
const localRules = ref<LocalApprovalRule[]>([]);

watch(
  rules,
  (newRules) => {
    localRules.value = [...newRules];
  },
  { immediate: true }
);

const sourceDisplayText = computed(() => approvalSourceText(props.source));

const handleDragEnd = async (event: { oldIndex: number; newIndex: number }) => {
  const { oldIndex, newIndex } = event;
  if (oldIndex === newIndex) return;

  if (!context.hasFeature.value) {
    context.showFeatureModal.value = true;
    // Revert local change
    localRules.value = [...rules.value];
    return;
  }

  try {
    await store.reorderRules(props.source, oldIndex, newIndex);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } catch {
    // Revert local change on error
    localRules.value = [...rules.value];
  }
};

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

<style scoped>
.rules-table-header,
.rules-table-row {
  grid-template-columns: 40px 200px 1fr 280px 96px;
}

.rules-row-ghost {
  opacity: 0.5;
  background: #e0f2fe;
}
</style>
