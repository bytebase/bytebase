<template>
  <div class="gap-y-4 w-full">
    <div class="flex items-stretch gap-x-4 overflow-hidden">
      <div class="flex-1 space-y-2 overflow-x-hidden overflow-y-auto">
        <div class="flex items-center h-[36px]">
          <NInput
            v-if="!readonly"
            v-model:value="state.title"
            class="!w-64"
            :placeholder="defaultTitle"
            type="text"
            size="small"
            :disabled="disabled"
            @input="state.dirty = true"
          />
          <h3 v-else class="font-medium text-sm text-main">
            {{ state.title || defaultTitle }}
          </h3>
        </div>
        <ExprEditor
          :expr="state.expr"
          :allow-admin="!readonly"
          :factor-list="factorList"
          :factor-support-dropdown="factorSupportDropdown"
          :factor-options-map="factorOptionsMap"
          :factor-operator-override-map="factorOperatorOverrideMap"
          @update="state.dirty = true"
        />
      </div>
      <div>
        <h3 class="font-medium text-sm text-main py-2">
          {{ $t("settings.sensitive-data.masking-level.self") }}
        </h3>
        <NSelect
          v-model:value="state.maskingLevel"
          style="width: 10rem"
          :options="options"
          :placeholder="$t('settings.sensitive-data.masking-level.selet-level')"
          :consistent-menu-width="false"
          :disabled="disabled || readonly"
          @update:value="state.dirty = true"
        />
      </div>
    </div>
    <div v-if="!readonly" class="flex justify-between items-center">
      <NPopconfirm v-if="allowDelete" @positive-click="$emit('delete')">
        <template #trigger>
          <NButton
            tag="div"
            size="small"
            style="--n-padding: 0 6px; --n-icon-margin: 4px"
            :disabled="disabled"
            @click.stop=""
          >
            <template #icon>
              <TrashIcon class="w-3.5 h-3.5" />
            </template>
            {{ $t("common.delete") }}
          </NButton>
        </template>
        <div class="whitespace-nowrap">
          {{ $t("settings.sensitive-data.global-rules.delete-rule-tip") }}
        </div>
      </NPopconfirm>
      <div class="flex justify-end gap-x-2 ml-auto">
        <NButton :disabled="disabled" @click="onCancel">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          type="primary"
          :disabled="!isValid || disabled || !state.dirty"
          @click="onConfirm"
        >
          {{ $t("common.confirm") }}
        </NButton>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { TrashIcon } from "lucide-vue-next";
import { NSelect, SelectOption, NPopconfirm } from "naive-ui";
import { computed, reactive, onMounted, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import ExprEditor from "@/components/ExprEditor";
import type { ConditionGroupExpr, Factor } from "@/plugins/cel";
import {
  resolveCELExpr,
  wrapAsGroup,
  buildCELExpr,
  validateSimpleExpr,
} from "@/plugins/cel";
import {
  Expr as CELExpr,
  ParsedExpr,
} from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { Expr } from "@/types/proto/google/type/expr";
import { MaskingLevel, maskingLevelToJSON } from "@/types/proto/v1/common";
import { MaskingRulePolicy_MaskingRule } from "@/types/proto/v1/org_policy_service";
import {
  batchConvertCELStringToParsedExpr,
  batchConvertParsedExprToCELString,
} from "@/utils";
import { factorSupportDropdown, factorOperatorOverrideMap } from "./utils";

const props = defineProps<{
  index: number;
  readonly: boolean;
  disabled: boolean;
  allowDelete: boolean;
  factorList: Factor[];
  factorOptionsMap: Map<Factor, SelectOption[]>;
  maskingRule: MaskingRulePolicy_MaskingRule;
}>();

const emit = defineEmits<{
  (event: "cancel"): void;
  (event: "delete"): void;
  (event: "confirm", rule: MaskingRulePolicy_MaskingRule): void;
}>();

type LocalState = {
  title: string;
  expr: ConditionGroupExpr;
  maskingLevel: MaskingLevel;
  dirty: boolean;
};

const { t } = useI18n();
const state = reactive<LocalState>({
  title: "",
  expr: wrapAsGroup(resolveCELExpr(CELExpr.fromJSON({}))),
  maskingLevel: MaskingLevel.FULL,
  dirty: false,
});

const resetLocalState = async (rule: MaskingRulePolicy_MaskingRule) => {
  let expr = CELExpr.fromJSON({});
  if (rule.condition?.expression) {
    const parsedExprs = await batchConvertCELStringToParsedExpr([
      rule.condition.expression,
    ]);
    expr = parsedExprs[0].expr ?? CELExpr.fromJSON({});
  }

  state.dirty = false;
  state.title = rule.condition?.title ?? "";
  state.expr = wrapAsGroup(resolveCELExpr(expr));
  state.maskingLevel = rule.maskingLevel;
};

onMounted(async () => {
  await resetLocalState(props.maskingRule);
});

const defaultTitle = computed(() => {
  return `${t("settings.sensitive-data.global-rules.condition-order")} ${
    props.index
  }`;
});

const options = computed(() => {
  return [
    MaskingLevel.FULL,
    MaskingLevel.PARTIAL,
    MaskingLevel.NONE,
  ].map<SelectOption>((level) => ({
    label: t(
      `settings.sensitive-data.masking-level.${maskingLevelToJSON(
        level
      ).toLowerCase()}`
    ),
    value: level,
  }));
});

const onCancel = async () => {
  await resetLocalState(props.maskingRule);
  emit("cancel");
  nextTick(() => (state.dirty = false));
};

const isValid = computed(() => {
  const { expr } = state;
  if (!expr) return false;
  return validateSimpleExpr(expr);
});

const onConfirm = async () => {
  const expressions = await batchConvertParsedExprToCELString([
    ParsedExpr.fromJSON({
      expr: buildCELExpr(state.expr),
    }),
  ]);
  emit("confirm", {
    ...props.maskingRule,
    maskingLevel: state.maskingLevel,
    condition: Expr.fromJSON({
      expression: expressions[0],
      title: state.title,
    }),
  });
  state.dirty = false;
};
</script>
