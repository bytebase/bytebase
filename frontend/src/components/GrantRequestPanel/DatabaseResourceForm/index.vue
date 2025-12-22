<template>
  <div class="w-full mb-2">
    <NRadioGroup
      :value="state.radioValue"
      :disabled="disabled || state.loading"
      class="w-full flex! flex-row justify-start items-center gap-4"
      @update:value="onSelectUpdate"
    >
      <NTooltip trigger="hover">
        <template #trigger>
          <NRadio
            :value="'ALL'"
            :label="$t('issue.grant-request.all-databases')"
          />
        </template>
        {{ $t("issue.grant-request.all-databases-tip") }}
      </NTooltip>
      <NRadio class="leading-6!" :value="'EXPRESSION'" :disabled="!project">
        <div class="flex items-center gap-x-1">
          <FeatureBadge :feature="requiredFeature" />
          <span>{{ $t("issue.grant-request.use-cel") }}</span>
        </div>
      </NRadio>
      <NRadio class="leading-6!" :value="'SELECT'" :disabled="!project">
        <div class="flex items-center gap-x-1">
          <FeatureBadge :feature="requiredFeature" />
          <span>{{ $t("issue.grant-request.manually-select") }}</span>
        </div>
      </NRadio>
    </NRadioGroup>
  </div>
  <div
    v-if="state.radioValue === 'SELECT'"
    class="w-full flex flex-row justify-start items-center"
  >
    <DatabaseResourceSelector
      v-model:database-resources="state.databaseResources"
      :disabled="disabled || state.loading"
      :project-name="project.name"
      :include-cloumn="includeCloumn"
    />
  </div>
  <ExprEditor
    v-if="state.radioValue === 'EXPRESSION'"
    :expr="state.expr"
    :readonly="disabled || state.loading"
    :factor-list="factorList"
    :option-config-map="factorOptionConfigMap"
    :factor-operator-override-map="factorOperatorOverrideMap"
  />
  <FeatureModal
    :open="state.showFeatureModal"
    :feature="requiredFeature"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { cloneDeep, head } from "lodash-es";
import { NRadio, NRadioGroup, NTooltip } from "naive-ui";
import { computed, reactive, watch } from "vue";
import ExprEditor from "@/components/ExprEditor";
import { type OptionConfig } from "@/components/ExprEditor/context";
import { FeatureBadge, FeatureModal } from "@/components/FeatureGuard";
import type { ConditionGroupExpr, Factor, Operator } from "@/plugins/cel";
import {
  buildCELExpr,
  emptySimpleExpr,
  resolveCELExpr,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import { hasFeature, useProjectByName } from "@/store";
import { type DatabaseResource } from "@/types";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  batchConvertCELStringToParsedExpr,
  getDatabaseNameOptionConfig,
} from "@/utils";
import {
  CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
  CEL_ATTRIBUTE_RESOURCE_DATABASE,
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
} from "@/utils/cel-attributes";
import {
  convertFromExpr,
  stringifyConditionExpression,
} from "@/utils/issue/cel";
import DatabaseResourceSelector from "./DatabaseResourceSelector.vue";

type RadioValue = "ALL" | "EXPRESSION" | "SELECT";

interface LocalState {
  loading: boolean;
  radioValue?: RadioValue;
  expr: ConditionGroupExpr;
  showFeatureModal: boolean;
  databaseResources: DatabaseResource[];
}

const props = withDefaults(
  defineProps<{
    disabled?: boolean;
    projectName: string;
    requiredFeature: PlanFeature;
    includeCloumn: boolean;
    databaseResources?: DatabaseResource[];
  }>(),
  {
    disabled: false,
    databaseResources: undefined,
  }
);

const state = reactive<LocalState>({
  loading: false,
  showFeatureModal: false,
  databaseResources: props.databaseResources || [],
  expr: wrapAsGroup(emptySimpleExpr()),
});

const convertToDatabaseResourceList = async (expr: ConditionGroupExpr) => {
  try {
    const parsedExpr = await buildCELExpr(expr);
    if (!parsedExpr) {
      return;
    }
    const conditionExpr = convertFromExpr(parsedExpr);
    return conditionExpr.databaseResources;
  } catch {
    return;
  }
};

const convertToConditionGroupExpr = async (
  databaseResources: DatabaseResource[]
) => {
  if (databaseResources.length === 0) {
    return wrapAsGroup(emptySimpleExpr());
  }

  try {
    const expression = stringifyConditionExpression({
      databaseResources,
    });

    const parsedExprs = await batchConvertCELStringToParsedExpr([expression]);
    const celExpr = head(parsedExprs);
    if (celExpr) {
      return wrapAsGroup(resolveCELExpr(celExpr));
    }
  } catch {
    return;
  }
};

const onSelectUpdate = async (select: RadioValue) => {
  if (!hasRequiredFeature.value && select !== "ALL") {
    state.showFeatureModal = true;
    return;
  }

  state.loading = true;
  if (select === "EXPRESSION" && state.radioValue === "SELECT") {
    // parse DatabaseResource[] to ConditionGroupExpr
    const expr = await convertToConditionGroupExpr(state.databaseResources);
    if (expr) {
      state.expr = expr;
    }
  } else if (select === "SELECT" && state.radioValue === "EXPRESSION") {
    // parse ConditionGroupExpr to DatabaseResource[]
    const resources = await convertToDatabaseResourceList(state.expr);
    if (resources) {
      state.databaseResources = resources;
    }
  }
  state.radioValue = select;
  state.loading = false;
};

watch(
  () => props.databaseResources,
  async (databaseResources) => {
    state.databaseResources = cloneDeep(databaseResources ?? []);
    if (!databaseResources || databaseResources.length <= 0) {
      state.radioValue = "ALL";
      return;
    }

    state.loading = true;
    const expr = await convertToConditionGroupExpr(databaseResources);
    if (expr) {
      state.expr = expr;
      state.radioValue = "EXPRESSION";
    } else {
      // fallback
      state.radioValue = "SELECT";
    }
    state.loading = false;
  },
  { immediate: true, deep: true }
);

const { project } = useProjectByName(computed(() => props.projectName));
const hasRequiredFeature = computed(() => hasFeature(props.requiredFeature));

const factorList = computed((): Factor[] => {
  const list: Factor[] = [
    CEL_ATTRIBUTE_RESOURCE_DATABASE,
    CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
    CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME,
  ];
  if (props.includeCloumn) {
    list.push(CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME);
  }
  return list;
});

const factorOperatorOverrideMap = new Map<Factor, Operator[]>([
  [CEL_ATTRIBUTE_RESOURCE_DATABASE, ["_==_", "@in"]],
  [CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME, ["_==_"]],
  [CEL_ATTRIBUTE_RESOURCE_TABLE_NAME, ["_==_", "@in"]],
  [CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME, ["_==_", "@in"]],
]);

const factorOptionConfigMap = computed((): Map<Factor, OptionConfig> => {
  return factorList.value.reduce((map, factor) => {
    if (factor !== CEL_ATTRIBUTE_RESOURCE_DATABASE) {
      map.set(factor, {
        options: [],
      });
    } else {
      map.set(factor, getDatabaseNameOptionConfig(props.projectName));
    }
    return map;
  }, new Map<Factor, OptionConfig>());
});

defineExpose({
  getDatabaseResources: async () => {
    switch (state.radioValue) {
      case "SELECT":
        return state.databaseResources;
      case "EXPRESSION": {
        const resources = await convertToDatabaseResourceList(state.expr);
        return resources;
      }
      default:
        return undefined;
    }
  },
  isValid: computed(() => {
    switch (state.radioValue) {
      case "SELECT":
        return state.databaseResources.length > 0;
      case "EXPRESSION":
        return validateSimpleExpr(state.expr);
      default:
        return true;
    }
  }),
});
</script>
