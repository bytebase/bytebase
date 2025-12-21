<template>
  <div
    class="w-full overflow-hidden flex flex-col gap-y-2 py-0.5"
    :class="[root ? '' : 'border rounded-[3px] bg-gray-50']"
  >
    <div v-if="!root" class="pl-2.5 pr-1 text-gray-500 flex items-center">
      <div class="flex-1">
        <template v-if="args.length > 0">
          <template v-if="operator === '_||_'">
            {{ $t("cel.condition.group.or.description") }}
          </template>
          <template v-if="operator === '_&&_'">
            {{ $t("cel.condition.group.and.description") }}
          </template>
        </template>
        <template v-else>
          <i18n-t
            keypath="cel.condition.add-condition-in-group-placeholder"
            tag="div"
            class="inline-flex items-center"
          >
            <template #plus>
              <heroicons:plus class="w-3 h-3 mx-1" />
            </template>
          </i18n-t>
        </template>
      </div>
      <div class="flex items-center justify-end">
        <NButton
          size="tiny"
          quaternary
          type="default"
          :style="`shrink: 0; padding-left: 0; padding-right: 0; --n-width: 22px;`"
          :disabled="readonly"
          @click="emit('remove')"
        >
          <heroicons:trash class="w-3.5 h-3.5" />
        </NButton>
      </div>
    </div>

    <div v-if="root && args.length === 0" class="text-gray-500">
      {{ $t("cel.condition.add-root-condition-placeholder") }}
    </div>
    <div
      v-for="(operand, i) in args"
      :key="i"
      class="flex items-start gap-x-1 w-full"
      :class="[root ? '' : 'px-1']"
    >
      <div class="w-14 shrink-0">
        <div v-if="i === 0" class="pl-1.5 pt-1 text-control">Where</div>
        <NSelect
          v-else-if="i === 1"
          v-model:value="operator"
          :options="OPERATORS"
          :consistent-menu-width="false"
          :disabled="readonly"
          size="small"
          @update:value="$emit('update')"
        />
        <div v-else class="pl-2 pt-1 text-control lowercase">
          {{ operatorLabel(operator) }}
        </div>
      </div>
      <div class="flex-1 flex flex-col gap-y-1 overflow-x-hidden">
        <ConditionGroup
          v-if="isConditionGroupExpr(operand)"
          :key="i"
          :expr="operand"
          @remove="removeConditionGroup(operand)"
          @update="$emit('update')"
        />
        <Condition
          v-if="isConditionExpr(operand)"
          :expr="operand"
          @remove="removeCondition(operand)"
          @update="$emit('update')"
        />
        <RawString
          v-if="isRawStringExpr(operand)"
          :expr="operand"
          @remove="removeRawString(operand)"
          @update="$emit('update')"
        />
      </div>
    </div>

    <div v-if="!root" class="pl-1.5 pb-1">
      <NButton size="small" quaternary :disabled="readonly" @click="addCondition">
        <template #icon
          ><heroicons:plus class="w-4 h-4 text-gray-500"
        /></template>
        <span class="text-gray-500">{{ $t("cel.condition.add") }}</span>
      </NButton>
      <NButton size="small" quaternary :disabled="readonly" @click="addRawString">
        <template #icon
          ><heroicons:plus class="w-4 h-4 text-gray-500"
        /></template>
        <span class="text-gray-500">{{
          $t("cel.condition.add-raw-expression")
        }}</span>
      </NButton>
    </div>

    <div v-if="root" class="flex gap-x-1">
      <NButton size="small" quaternary :disabled="readonly" @click="addCondition">
        <template #icon><heroicons:plus class="w-4 h-4" /></template>
        <span>{{ $t("cel.condition.add") }}</span>
      </NButton>
      <NButton size="small" quaternary :disabled="readonly" @click="addConditionGroup">
        <template #icon><heroicons:plus class="w-4 h-4" /></template>
        <span>{{ $t("cel.condition.add-group") }}</span>
        <NTooltip>
          <template #trigger>
            <heroicons:question-mark-circle class="ml-1 w-3 h-3" />
          </template>
          <div class="max-w-[18rem]">
            {{ $t("cel.condition.group.tooltip") }}
          </div>
        </NTooltip>
      </NButton>
      <NButton
        v-if="enableRawExpression"
        size="small"
        quaternary
        :disabled="readonly"
        @click="addRawString"
      >
        <template #icon><heroicons:plus class="w-4 h-4" /></template>
        <span>{{ $t("cel.condition.add-raw-expression") }}</span>
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
/* eslint-disable vue/no-mutating-props */
import { NButton, NSelect, NTooltip, type SelectOption } from "naive-ui";
import { computed } from "vue";
import {
  type ConditionExpr,
  type ConditionGroupExpr,
  ExprType,
  type Factor,
  isConditionExpr,
  isConditionGroupExpr,
  isNumberFactor,
  isRawStringExpr,
  isStringFactor,
  isTimestampFactor,
  type LogicalOperator,
  LogicalOperatorList,
  type RawStringExpr,
} from "@/plugins/cel";
import Condition from "./Condition.vue";
import { getOperatorListByFactor } from "./components/common";
import { useExprEditorContext } from "./context";
import RawString from "./RawString.vue";

const props = defineProps<{
  expr: ConditionGroupExpr;
  root?: boolean;
}>();

const emit = defineEmits<{
  (event: "remove"): void;
  (event: "update"): void;
}>();

const { readonly, enableRawExpression, factorList, factorOperatorOverrideMap } =
  useExprEditorContext();

const operator = computed({
  get() {
    return props.expr.operator;
  },
  set(op) {
    props.expr.operator = op;
  },
});
const args = computed(() => props.expr.args);

const operatorLabel = (op: LogicalOperator) => {
  if (op === "_&&_") return "and";
  if (op === "_||_") return "or";
  throw new Error(`unknown logical operator "${op}"`);
};

const OPERATORS: SelectOption[] = [
  { label: operatorLabel("_&&_"), value: "_&&_" },
  { label: operatorLabel("_||_"), value: "_||_" },
];

const getDefaultValue = (factor: Factor): string | number | Date => {
  if (isNumberFactor(factor)) return 0;
  if (isStringFactor(factor)) return "";
  if (isTimestampFactor(factor)) return new Date();
  return "";
};

const addCondition = () => {
  const factor = factorList.value[0];
  const operators = getOperatorListByFactor(
    factor,
    factorOperatorOverrideMap.value
  );

  const operator = operators[0];
  if (!operator) {
    return;
  }

  args.value.push({
    type: ExprType.Condition,
    operator,
    args: [factor, getDefaultValue(factor)],
  } as ConditionExpr);
  emit("update");
};

const addRawString = () => {
  args.value.push({
    type: ExprType.RawString,
    content: "",
  });
  emit("update");
};

const addConditionGroup = () => {
  args.value.push({
    type: ExprType.ConditionGroup,
    operator: LogicalOperatorList[0],
    args: [],
  });
  emit("update");
};

const removeCondition = (condition: ConditionExpr) => {
  const index = args.value.indexOf(condition);
  if (index >= 0) {
    args.value.splice(index, 1);
    emit("update");
  }
};

const removeConditionGroup = (group: ConditionGroupExpr) => {
  const index = args.value.indexOf(group);
  if (index >= 0) {
    args.value.splice(index, 1);
    emit("update");
  }
};

const removeRawString = (rawString: RawStringExpr) => {
  const index = args.value.indexOf(rawString);
  if (index >= 0) {
    args.value.splice(index, 1);
    emit("update");
  }
};
</script>
