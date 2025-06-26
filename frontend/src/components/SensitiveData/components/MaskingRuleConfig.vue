<template>
  <div class="space-y-4 w-full">
    <div
      class="flex flex-col md:flex-row items-start md:items-stretch gap-x-4 gap-y-4 overflow-hidden"
    >
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
          :option-config-map="optionConfigMap"
          :factor-operator-override-map="factorOperatorOverrideMap"
          @update="state.dirty = true"
        />
      </div>
      <div>
        <h3 class="font-medium text-sm text-main py-2">
          {{ $t("settings.sensitive-data.semantic-types.table.semantic-type") }}
        </h3>
        <NSelect
          v-model:value="state.semanticType"
          style="min-width: 10rem"
          :filterable="true"
          :options="options"
          :filter="filterByTitle"
          :placeholder="$t('settings.sensitive-data.semantic-types.select')"
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
            quaternary
            tag="div"
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
import { head } from "lodash-es";
import { TrashIcon } from "lucide-vue-next";
import type { SelectOption } from "naive-ui";
import { NSelect, NPopconfirm, NInput, NButton } from "naive-ui";
import { computed, reactive, onMounted, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import ExprEditor from "@/components/ExprEditor";
import { type OptionConfig } from "@/components/ExprEditor/context";
import type { ConditionGroupExpr, Factor, SimpleExpr } from "@/plugins/cel";
import {
  resolveCELExpr,
  wrapAsGroup,
  buildCELExpr,
  validateSimpleExpr,
  emptySimpleExpr,
} from "@/plugins/cel";
import { useSettingV1Store } from "@/store";
import { Expr } from "@/types/proto/google/type/expr";
import type { MaskingRulePolicy_MaskingRule } from "@/types/proto/v1/org_policy_service";
import type { SemanticTypeSetting_SemanticType as SemanticType } from "@/types/proto-es/v1/setting_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import {
  batchConvertCELStringToParsedExpr,
  batchConvertParsedExprToCELString,
} from "@/utils";
import { factorSupportDropdown, factorOperatorOverrideMap } from "./utils";

export interface SemanticTypeSelectOption extends SelectOption {
  value: string;
  semanticType: SemanticType;
}

const props = defineProps<{
  index: number;
  readonly: boolean;
  disabled: boolean;
  allowDelete: boolean;
  factorList: Factor[];
  optionConfigMap: Map<Factor, OptionConfig>;
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
  semanticType?: string;
  dirty: boolean;
};

const { t } = useI18n();
const state = reactive<LocalState>({
  title: "",
  expr: wrapAsGroup(emptySimpleExpr()),
  dirty: false,
});
const settingStore = useSettingV1Store();

const semanticTypeSettingValue = computed(() => {
  const semanticTypeSetting = settingStore.getSettingByName(
    Setting_SettingName.SEMANTIC_TYPES
  );
  if (semanticTypeSetting?.value?.value?.case === "semanticTypeSettingValue") {
    return semanticTypeSetting.value.value.value.types ?? [];
  }
  return [];
});

const options = computed(() => {
  return semanticTypeSettingValue.value.map<SemanticTypeSelectOption>(
    (semanticType) => ({
      value: semanticType.id,
      label: semanticType.title,
      semanticType,
    })
  );
});

const filterByTitle = (pattern: string, option: SelectOption) => {
  const { semanticType } = option as SemanticTypeSelectOption;
  pattern = pattern.toLowerCase();
  return semanticType.title.toLowerCase().includes(pattern);
};

const resetLocalState = async (rule: MaskingRulePolicy_MaskingRule) => {
  let expr: SimpleExpr = emptySimpleExpr();
  if (rule.condition?.expression) {
    const parsedExprs = await batchConvertCELStringToParsedExpr([
      rule.condition.expression,
    ]);
    const celExpr = head(parsedExprs);
    if (celExpr) {
      expr = resolveCELExpr(celExpr);
    }
  }

  state.dirty = false;
  state.title = rule.condition?.title ?? "";
  state.expr = wrapAsGroup(expr);
  state.semanticType = rule.semanticType ? rule.semanticType : undefined;
};

onMounted(async () => {
  await resetLocalState(props.maskingRule);
});

const defaultTitle = computed(() => {
  return `${t("settings.sensitive-data.global-rules.condition-order")} ${
    props.index
  }`;
});

const onCancel = async () => {
  await resetLocalState(props.maskingRule);
  emit("cancel");
  nextTick(() => (state.dirty = false));
};

const isValid = computed(() => {
  const { expr, semanticType } = state;
  if (!expr || !semanticType) return false;
  return validateSimpleExpr(expr);
});

const onConfirm = async () => {
  const celexpr = await buildCELExpr(state.expr);
  if (!celexpr) {
    return;
  }
  const expressions = await batchConvertParsedExprToCELString([celexpr]);
  emit("confirm", {
    ...props.maskingRule,
    semanticType: state.semanticType!,
    condition: Expr.fromPartial({
      expression: expressions[0],
      title: state.title,
    }),
  });
  state.dirty = false;
};
</script>
