<template>
  <div
    class="w-[calc(100vw-8rem)] lg:max-w-[75vw] 2xl:max-w-[55vw] max-h-[calc(100vh-10rem)] flex flex-col"
  >
    <div class="flex-1 flex flex-col divide-y overflow-hidden">
      <div class="flex items-start gap-x-4 mb-4">
        <div class="space-y-2 flex-1">
          <label class="block font-medium text-sm text-control">
            {{ $t("common.name") }}
          </label>
          <NInput
            v-model:value="state.risk.title"
            :disabled="!allowAdmin"
            :placeholder="$t('custom-approval.security-rule.input-rule-name')"
            @input="$emit('update')"
          />
        </div>
        <div class="space-y-2">
          <label class="block font-medium text-sm text-control">
            {{ $t("custom-approval.security-rule.risk.self") }}
          </label>
          <RiskLevelSelect
            v-model:value="state.risk.level"
            :disabled="!allowAdmin"
            @update:value="$emit('update')"
          />
        </div>
        <div class="space-y-2">
          <label class="block font-medium text-sm text-control">
            {{ $t("custom-approval.security-rule.source.self") }}
          </label>
          <RiskSourceSelect
            v-model:value="state.risk.source"
            :disabled="mode === 'EDIT' || !allowAdmin"
            @update:value="$emit('update')"
          />
        </div>
      </div>

      <div class="flex-1 flex items-stretch gap-x-4 overflow-hidden">
        <div class="flex-1 space-y-2 py-4 overflow-x-hidden overflow-y-auto">
          <h3 class="font-medium text-sm text-control">
            {{ $t("custom-approval.security-rule.condition.self") }}
          </h3>
          <div class="text-sm text-control-light">
            {{ $t("custom-approval.security-rule.condition.description-tips") }}
            <LearnMoreLink
              v-if="false"
              url="https://www.bytebase.com/404"
              class="ml-1"
            />
          </div>
          <ExprEditor
            :expr="state.expr"
            :allow-admin="allowAdmin"
            :factor-list="getFactorList(state.risk.source)"
            :factor-support-dropdown="factorSupportDropdown"
            :factor-options-map="getFactorOptionsMap(state.risk.source)"
            @update="$emit('update')"
          />
        </div>

        <div
          v-if="allowAdmin"
          class="w-[45%] max-w-[40rem] overflow-y-auto py-4 shrink-0"
        >
          <h3 class="font-medium text-sm text-control mb-2">
            {{ $t("custom-approval.security-rule.template.templates") }}
          </h3>
          <RuleTemplateTable
            :dirty="dirty"
            @apply-template="handleApplyRuleTemplate"
          />
        </div>
      </div>
    </div>
    <footer
      v-if="allowAdmin"
      class="flex items-center justify-end gap-x-2 pt-4 border-t"
    >
      <NButton @click="$emit('cancel')">{{ $t("common.cancel") }}</NButton>
      <NButton
        type="primary"
        :disabled="!allowCreateOrUpdate"
        @click="handleUpsert"
      >
        {{ mode === "CREATE" ? $t("common.add") : $t("common.update") }}
      </NButton>
    </footer>
  </div>
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { NButton, NInput } from "naive-ui";
import { computed, ref, watch } from "vue";
import ExprEditor from "@/components/ExprEditor";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import type { ConditionGroupExpr } from "@/plugins/cel";
import {
  resolveCELExpr,
  buildCELExpr,
  wrapAsGroup,
  validateSimpleExpr,
} from "@/plugins/cel";
import {
  Expr as CELExpr,
  ParsedExpr,
} from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { Expr } from "@/types/proto/google/type/expr";
import { Risk } from "@/types/proto/v1/risk_service";
import {
  convertCELStringToParsedExpr,
  convertParsedExprToCELString,
} from "@/utils";
import {
  getFactorList,
  getFactorOptionsMap,
  factorSupportDropdown,
} from "../../common/utils";
import { useRiskCenterContext } from "../context";
import RiskLevelSelect from "./RiskLevelSelect.vue";
import RiskSourceSelect from "./RiskSourceSelect.vue";
import RuleTemplateTable from "./RuleTemplateTable.vue";

type LocalState = {
  risk: Risk;
  expr: ConditionGroupExpr;
};

const props = defineProps<{
  dirty?: boolean;
}>();

const emit = defineEmits<{
  (event: "cancel"): void;
  (event: "update"): void;
  (event: "save", risk: Risk): void;
}>();

const context = useRiskCenterContext();
const { allowAdmin } = context;

const state = ref<LocalState>({
  risk: Risk.fromJSON({}),
  expr: wrapAsGroup(resolveCELExpr(CELExpr.fromJSON({}))),
});
const mode = computed(() => context.dialog.value?.mode ?? "CREATE");

const resolveLocalState = async () => {
  const risk = cloneDeep(context.dialog.value!.risk);
  const parsedExpr = await convertCELStringToParsedExpr(
    risk.condition?.expression ?? ""
  );
  state.value = {
    risk,
    expr: wrapAsGroup(resolveCELExpr(parsedExpr.expr ?? CELExpr.fromJSON({}))),
  };
};

const allowCreateOrUpdate = computed(() => {
  if (mode.value === "EDIT") {
    if (!props.dirty) return false;
  }

  const { risk, expr } = state.value;
  if (!risk.title.trim()) return false;
  if (!expr) return false;

  if (!validateSimpleExpr(expr)) {
    return false;
  }

  return true;
});

const handleUpsert = async () => {
  if (!context.hasFeature.value) {
    context.showFeatureModal.value = true;
    return;
  }
  if (!state.value.expr) return;

  const risk = cloneDeep(state.value.risk);

  const expression = await convertParsedExprToCELString(
    ParsedExpr.fromJSON({
      expr: buildCELExpr(state.value.expr),
    })
  );
  risk.condition = Expr.fromJSON({
    expression,
  });
  emit("save", risk);
};

const handleApplyRuleTemplate = (
  overrides: Partial<Risk>,
  expr: ConditionGroupExpr
) => {
  Object.assign(state.value.risk, overrides);
  state.value.expr = cloneDeep(expr);
  emit("update");
};

watch(() => context.dialog, resolveLocalState, {
  immediate: true,
});
</script>
