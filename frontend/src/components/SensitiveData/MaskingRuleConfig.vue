<template>
  <div class="gap-y-4">
    <div class="flex items-stretch gap-x-4 overflow-hidden">
      <div class="flex-1 space-y-2 py-4 overflow-x-hidden overflow-y-auto">
        <h3 class="font-medium text-sm text-control">
          {{ $t("custom-approval.security-rule.condition.self") }}
        </h3>
        <ExprEditor
          :expr="state.expr"
          :allow-admin="!readonly"
          :factor-list="factorList"
          :factor-support-dropdown="factorSupportDropdown"
          :factor-options-map="getFactorOptionsMap()"
          :factor-operator-override-map="factorOperatorOverrideMap"
          @update="state.dirty = true"
        />
      </div>
      <div class="space-y-2 py-4">
        <h3 class="font-medium text-sm text-control">
          {{ $t("settings.sensitive-data.masking-level.self") }}
        </h3>
        <NSelect
          v-model:value="state.maskingLevel"
          style="width: 10rem"
          :options="options"
          :placeholder="$t('settings.sensitive-data.masking-level.selet-level')"
          :consistent-menu-width="false"
          :disabled="readonly"
          @update:value="state.dirty = true"
        />
      </div>
    </div>
    <div
      v-if="(state.dirty || isCreate) && !readonly"
      class="flex justify-end gap-x-3"
    >
      <NButton @click="onCancel">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton type="primary" :disabled="!isValid" @click="onConfirm">
        {{ $t("common.confirm") }}
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { NSelect, SelectOption } from "naive-ui";
import { computed, ref, onMounted } from "vue";
import { useI18n } from "vue-i18n";
import ExprEditor from "@/components/ExprEditor";
import type { ConditionGroupExpr } from "@/plugins/cel";
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
  convertCELStringToParsedExpr,
  convertParsedExprToCELString,
} from "@/utils";
import {
  factorList,
  getFactorOptionsMap,
  factorSupportDropdown,
  factorOperatorOverrideMap,
} from "./utils";

const props = defineProps<{
  isCreate?: boolean;
  readonly: boolean;
  maskingRule: MaskingRulePolicy_MaskingRule;
}>();

const emit = defineEmits<{
  (event: "cancel"): void;
  (event: "confirm", rule: MaskingRulePolicy_MaskingRule): void;
}>();

type LocalState = {
  expr: ConditionGroupExpr;
  maskingLevel: MaskingLevel;
  dirty: boolean;
};

const { t } = useI18n();
const state = ref<LocalState>({
  expr: wrapAsGroup(resolveCELExpr(CELExpr.fromJSON({}))),
  maskingLevel: MaskingLevel.FULL,
  dirty: false,
});

onMounted(async () => {
  await resetLocalState();
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
  await resetLocalState();
  emit("cancel");
};

const resetLocalState = async () => {
  const rule = cloneDeep(props.maskingRule);
  const parsedExpr = await convertCELStringToParsedExpr(
    rule.condition?.expression ?? ""
  );
  state.value = {
    dirty: false,
    maskingLevel: rule.maskingLevel,
    expr: wrapAsGroup(resolveCELExpr(parsedExpr.expr ?? CELExpr.fromJSON({}))),
  };
};

const isValid = computed(() => {
  const { expr } = state.value;
  if (!expr) return false;
  return validateSimpleExpr(expr);
});

const onConfirm = async () => {
  const expression = await convertParsedExprToCELString(
    ParsedExpr.fromJSON({
      expr: buildCELExpr(state.value.expr),
    })
  );
  emit("confirm", {
    ...props.maskingRule,
    maskingLevel: state.value.maskingLevel,
    condition: Expr.fromJSON({
      expression,
    }),
  });
  state.value.dirty = false;
};
</script>
