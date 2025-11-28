<template>
  <div>
    <NDataTable
      size="small"
      :columns="columns"
      :data="steps"
      :striped="true"
      :bordered="true"
    />
    <div v-if="editable && allowAdmin" class="mt-4">
      <NButton @click="addStep">
        <template #icon>
          <PlusIcon class="w-4" />
        </template>
        <span>
          {{ $t("custom-approval.approval-flow.node.add") }}
        </span>
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */
import {
  ArrowDownIcon,
  ArrowUpIcon,
  PlusIcon,
  TrashIcon,
} from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NButton, NDataTable } from "naive-ui";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { RoleSelect } from "@/components/v2";
import { SpinnerButton } from "@/components/v2/Form";
import { PresetRoleType } from "@/types";
import type { ApprovalFlow } from "@/types/proto-es/v1/issue_service_pb";
import { displayRoleTitle } from "@/utils";
import { useCustomApprovalContext } from "../context";

const props = defineProps<{
  flow: ApprovalFlow;
  editable: boolean;
}>();

const emit = defineEmits<{
  (event: "update"): void;
}>();

const { t } = useI18n();

const context = useCustomApprovalContext();
const { allowAdmin } = context;

const columns = computed(
  (): DataTableColumn<{ role: string; index: number }>[] => {
    const cols: DataTableColumn<{ role: string; index: number }>[] = [
      {
        title: t("custom-approval.approval-flow.node.order"),
        key: "order",
        width: 80,
        align: "center",
        render: (_, index) => index + 1,
      },
      {
        title: t("custom-approval.approval-flow.node.approver"),
        key: "approver",
        minWidth: 200,
        resizable: true,
        render: (stepData) => {
          if (props.editable) {
            return h(RoleSelect, {
              suffix: "",
              placement: "top-start",
              value: stepData.role,
              style: "width: 80%",
              "onUpdate:value": (val: string | string[]) => {
                const role = Array.isArray(val) ? val[0] : val;
                props.flow.roles[stepData.index] = role;
                emit("update");
              },
            });
          }
          return displayRoleTitle(stepData.role);
        },
      },
    ];

    if (props.editable) {
      cols.push({
        title: t("common.operations"),
        key: "operations",
        render: (stepData, index) =>
          h("div", { class: "flex gap-x-1" }, [
            h(
              NButton,
              {
                disabled: index === 0 || !allowAdmin.value,
                size: "tiny",
                onClick: () => reorder(stepData, -1),
              },
              () => h(ArrowUpIcon, { class: "w-4" })
            ),
            h(
              NButton,
              {
                disabled: index === steps.value.length - 1 || !allowAdmin.value,
                size: "tiny",
                onClick: () => reorder(stepData, 1),
              },
              () => h(ArrowDownIcon, { class: "w-4" })
            ),
            allowAdmin.value
              ? h(
                  SpinnerButton,
                  {
                    size: "tiny",
                    tooltip: t("custom-approval.approval-flow.node.delete"),
                    onConfirm: () => removeStep(stepData),
                  },
                  () => h(TrashIcon, { class: "w-3" })
                )
              : null,
          ]),
      });
    }

    return cols;
  }
);

const steps = computed(() => {
  return props.flow.roles.map((role, index) => ({ role, index }));
});

const reorder = (stepData: { role: string; index: number }, offset: -1 | 1) => {
  const index = stepData.index;
  const target = index + offset;
  if (target < 0 || target >= props.flow.roles.length) return;
  const tmp = props.flow.roles[index];
  props.flow.roles[index] = props.flow.roles[target];
  props.flow.roles[target] = tmp;

  emit("update");
};
const addStep = () => {
  props.flow.roles.push(PresetRoleType.WORKSPACE_ADMIN);
  emit("update");
};

const removeStep = async (stepData: { role: string; index: number }) => {
  props.flow.roles.splice(stepData.index, 1);
  emit("update");
};
</script>
