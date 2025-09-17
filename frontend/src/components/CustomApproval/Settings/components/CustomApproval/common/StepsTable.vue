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

<script lang="tsx" setup>
import { create } from "@bufbuild/protobuf";
import {
  ArrowUpIcon,
  ArrowDownIcon,
  TrashIcon,
  PlusIcon,
} from "lucide-vue-next";
import { NButton, NDataTable } from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { RoleSelect } from "@/components/v2";
import { SpinnerButton } from "@/components/v2/Form";
import { PresetRoleType } from "@/types";
import type {
  ApprovalFlow,
  ApprovalStep,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  ApprovalNode_Type,
  ApprovalStep_Type,
  ApprovalNodeSchema,
  ApprovalStepSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import { approvalNodeText } from "@/utils";
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

const columns = computed((): DataTableColumn<ApprovalStep>[] => {
  const cols: DataTableColumn<ApprovalStep>[] = [
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
      render: (step) => {
        if (props.editable) {
          return (
            <RoleSelect
              suffix=""
              value={step.nodes[0]?.role}
              style="width: 80%"
              onUpdate:value={(val: string | string[]) => {
                const role = Array.isArray(val) ? val[0] : val;
                step.nodes[0] = create(ApprovalNodeSchema, {
                  type: ApprovalNode_Type.ANY_IN_GROUP,
                  role: role,
                });
                emit("update");
              }}
            />
          );
        }
        return approvalNodeText(step.nodes[0]);
      },
    },
  ];

  if (props.editable) {
    cols.push({
      title: t("common.operations"),
      key: "operations",
      render: (step, index) => (
        <div class="flex gap-x-1">
          <NButton
            disabled={index === 0 || !allowAdmin.value}
            size="tiny"
            onClick={() => reorder(step, index, -1)}
          >
            <ArrowUpIcon class={"w-4"} />
          </NButton>
          <NButton
            disabled={index === steps.value.length - 1 || !allowAdmin.value}
            size="tiny"
            onClick={() => reorder(step, index, 1)}
          >
            <ArrowDownIcon class={"w-4"} />
          </NButton>
          {allowAdmin.value && (
            <SpinnerButton
              size="tiny"
              tooltip={t("custom-approval.approval-flow.node.delete")}
              onConfirm={() => removeStep(step, index)}
            >
              <TrashIcon class={"w-3"} />
            </SpinnerButton>
          )}
        </div>
      ),
    });
  }

  return cols;
});

const steps = computed(() => {
  return props.flow.steps;
});

const reorder = (step: ApprovalStep, index: number, offset: -1 | 1) => {
  const target = index + offset;
  if (target < 0 || target >= steps.value.length) return;
  const tmp = steps.value[index];
  steps.value[index] = steps.value[target];
  steps.value[target] = tmp;

  emit("update");
};
const addStep = () => {
  steps.value.push(
    create(ApprovalStepSchema, {
      type: ApprovalStep_Type.ANY,
      nodes: [
        create(ApprovalNodeSchema, {
          type: ApprovalNode_Type.ANY_IN_GROUP,
          role: PresetRoleType.WORKSPACE_ADMIN,
        }),
      ],
    })
  );
  emit("update");
};

const removeStep = async (step: ApprovalStep, index: number) => {
  steps.value.splice(index, 1);
  emit("update");
};
</script>
