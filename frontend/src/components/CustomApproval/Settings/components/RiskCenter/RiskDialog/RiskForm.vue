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
            :placeholder="$t('custom-approval.risk-rule.input-rule-name')"
            @input="$emit('update')"
          />
        </div>
        <div class="space-y-2">
          <label class="block font-medium text-sm text-control">
            {{ $t("custom-approval.risk-rule.risk.self") }}
          </label>
          <RiskLevelSelect
            v-model:value="state.risk.level"
            :disabled="!allowAdmin"
            @update:value="$emit('update')"
          />
        </div>
        <div class="space-y-2">
          <label class="block font-medium text-sm text-control">
            {{ $t("custom-approval.risk-rule.source.self") }}
          </label>
          <RiskSourceSelect
            v-model:value="state.risk.source"
            :disabled="sourceList.length <= 1 || !allowAdmin"
            :sources="sourceList"
            @update:value="$emit('update')"
          />
        </div>
      </div>

      <div class="flex-1 flex items-stretch gap-x-4 overflow-hidden">
        <div class="flex-1 space-y-2 py-4 overflow-x-hidden overflow-y-auto">
          <h3 class="font-medium text-sm text-control">
            {{ $t("cel.condition.self") }}
          </h3>
          <div class="text-sm text-control-light">
            {{ $t("cel.condition.description-tips") }}
            <LearnMoreLink
              url="https://docs.bytebase.com/administration/risk-center/#configuration?source=console"
              class="ml-1"
            />
          </div>
          <ExprEditor
            :expr="state.expr"
            :allow-admin="allowAdmin"
            :factor-list="getFactorList(state.risk.source)"
            :factor-support-dropdown="factorSupportDropdown"
            :option-config-map="getOptionConfigMap(state.risk.source)"
            @update="$emit('update')"
          />
        </div>

        <div
          v-if="allowAdmin"
          class="w-[45%] max-w-[40rem] overflow-y-auto py-4 shrink-0"
        >
          <h3 class="font-medium text-sm text-control mb-2">
            {{ $t("custom-approval.risk-rule.template.templates") }}
          </h3>
          <RuleTemplateTable
            :dirty="dirty"
            :source="state.risk.source"
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
import { create } from "@bufbuild/protobuf";
import { cloneDeep, head, uniq, flatten } from "lodash-es";
import { NButton, NInput } from "naive-ui";
import { computed, reactive, watch } from "vue";
import ExprEditor from "@/components/ExprEditor";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import type { ConditionGroupExpr, Factor, SimpleExpr } from "@/plugins/cel";
import { ExprType } from "@/plugins/cel";
import {
  resolveCELExpr,
  buildCELExpr,
  wrapAsGroup,
  validateSimpleExpr,
  emptySimpleExpr,
} from "@/plugins/cel";
import { useSupportedSourceList } from "@/types";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import type { Risk } from "@/types/proto-es/v1/risk_service_pb";
import { Risk_Source, RiskSchema } from "@/types/proto-es/v1/risk_service_pb";
import {
  batchConvertCELStringToParsedExpr,
  batchConvertParsedExprToCELString,
  hasWorkspacePermissionV2,
} from "@/utils";
import {
  getFactorList,
  getOptionConfigMap,
  factorSupportDropdown,
  RiskSourceFactorMap,
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
const supportedSourceList = useSupportedSourceList();

const state = reactive<LocalState>({
  risk: create(RiskSchema, {}),
  expr: wrapAsGroup(emptySimpleExpr()),
});
const mode = computed(() => context.dialog.value?.mode ?? "CREATE");

const extractFactorList = (expr: SimpleExpr): Factor[] => {
  switch (expr.type) {
    case ExprType.Condition:
      return expr.args.length > 1 ? [expr.args[0]] : [];
    case ExprType.ConditionGroup:
      return uniq(flatten(expr.args.map(extractFactorList)));
    case ExprType.RawString:
      return [];
  }
};

const selectedFactor = computed(() => extractFactorList(state.expr));

const sourceList = computed(() => {
  if (mode.value !== "EDIT") {
    return supportedSourceList.value;
  }

  const sourceList: Risk_Source[] = [];
  for (const [source, factorList] of RiskSourceFactorMap.entries()) {
    if (!supportedSourceList.value.includes(source)) {
      continue;
    }
    if (selectedFactor.value.every((v) => factorList.includes(v))) {
      sourceList.push(source);
    }
  }

  return sourceList;
});

const resolveLocalState = async () => {
  const risk = cloneDeep(context.dialog.value!.risk);
  let expr: SimpleExpr = emptySimpleExpr();
  if (risk.condition?.expression) {
    const parsedExprs = await batchConvertCELStringToParsedExpr([
      risk.condition.expression,
    ]);
    const celExpr = head(parsedExprs);
    if (celExpr) {
      expr = resolveCELExpr(celExpr);
    }
  }

  state.risk = risk;
  state.expr = wrapAsGroup(expr);
};

const allowCreateOrUpdate = computed(() => {
  // Check create or update permission.
  if (mode.value === "CREATE") {
    if (!hasWorkspacePermissionV2("bb.risks.create")) {
      return false;
    }
  } else if (mode.value === "EDIT") {
    if (!hasWorkspacePermissionV2("bb.risks.update")) {
      return false;
    }
    if (!props.dirty) return false;
  }

  const { risk, expr } = state;
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
  if (!state.expr) return;

  const risk = cloneDeep(state.risk);

  const celexpr = await buildCELExpr(state.expr);
  if (!celexpr) {
    return;
  }
  const expressions = await batchConvertParsedExprToCELString([celexpr]);
  risk.condition = create(ExprSchema, {
    expression: expressions[0],
  });
  emit("save", risk);
};

const handleApplyRuleTemplate = (
  overrides: Partial<Risk>,
  expr: ConditionGroupExpr
) => {
  Object.assign(state.risk, overrides);
  state.expr = cloneDeep(expr);
  emit("update");
};

watch(() => context.dialog, resolveLocalState, {
  immediate: true,
});
</script>
