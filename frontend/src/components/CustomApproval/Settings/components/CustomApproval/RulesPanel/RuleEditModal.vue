<template>
  <NModal
    :show="show"
    :mask-closable="true"
    :close-on-esc="true"
    @update:show="$emit('update:show', $event)"
  >
    <div
      class="w-[calc(100vw-8rem)] lg:max-w-[75vw] 2xl:max-w-[55vw] max-h-[calc(100vh-10rem)] bg-white rounded-lg shadow-lg flex flex-col overflow-hidden"
    >
      <div class="px-6 py-4 border-b flex items-center justify-between">
        <h2 class="text-lg font-medium text-control">
          {{
            mode === "create"
              ? $t("custom-approval.approval-flow.create-approval-flow")
              : $t("custom-approval.approval-flow.edit-rule")
          }}
        </h2>
      </div>

      <div class="flex-1 flex flex-col px-6 py-4 gap-y-4 overflow-y-auto">
        <div
          v-if="isFallback"
          class="text-sm text-amber-600 bg-amber-50 p-3 rounded"
        >
          {{ $t("custom-approval.approval-flow.fallback-rules-hint") }}
        </div>

        <div v-if="mode === 'create'" class="flex flex-col gap-y-2">
          <h3 class="font-medium text-sm text-control">
            {{ $t("custom-approval.approval-flow.template.presets-title") }}
          </h3>
          <div class="flex flex-wrap gap-2">
            <NTooltip
              v-for="(template, index) in templates"
              :key="index"
              to="body"
              :disabled="!template.description()"
            >
              <template #trigger>
                <NButton
                  size="small"
                  :disabled="!allowAdmin"
                  @click="applyTemplate(template)"
                >
                  {{ template.title() }}
                </NButton>
              </template>
              {{ template.description() }}
            </NTooltip>
          </div>
        </div>

        <div class="flex flex-col gap-y-2">
          <h3 class="font-medium text-sm text-control">
            {{ $t("common.title") }} <RequiredStar />
          </h3>
          <NInput
            v-model:value="state.title"
            :placeholder="$t('common.title')"
            :disabled="!allowAdmin"
          />
        </div>

        <div class="flex flex-col gap-y-2">
          <h3 class="font-medium text-sm text-control">
            {{ $t("common.description") }}
          </h3>
          <NInput
            v-model:value="state.description"
            type="textarea"
            :placeholder="$t('common.description')"
            :disabled="!allowAdmin"
            :rows="2"
          />
        </div>

        <div class="flex-1 flex flex-col gap-y-2">
          <h3 class="font-medium text-sm text-control">
            {{ $t("cel.condition.self") }} <RequiredStar />
          </h3>
          <div class="text-sm text-control-light">
            {{ $t("cel.condition.description-tips") }}
          </div>
          <ExprEditor
            :expr="state.conditionExpr"
            :readonly="!allowAdmin"
            :factor-list="factorList"
            :option-config-map="optionConfigMap"
            @update="handleUpdate"
          />
        </div>

        <div class="flex flex-col gap-y-2">
          <h3 class="font-medium text-sm text-control">
            {{ $t("custom-approval.approval-flow.node.nodes") }} <RequiredStar />
          </h3>
          <div class="text-sm text-control-light">
            {{ $t("custom-approval.approval-flow.node.description") }}
          </div>
          <StepsTable
            :flow="state.flow"
            :editable="allowAdmin"
            @update="handleUpdate"
          />
        </div>
      </div>

      <footer
        class="flex items-center justify-end gap-x-2 px-6 py-4 border-t"
      >
        <NButton @click="$emit('update:show', false)">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          type="primary"
          :disabled="!allowSave || !allowAdmin"
          @click="handleSave"
        >
          {{ mode === "create" ? $t("common.create") : $t("common.update") }}
        </NButton>
      </footer>
    </div>
  </NModal>
</template>

<script lang="ts" setup>
import { create as createProto } from "@bufbuild/protobuf";
import { cloneDeep, head } from "lodash-es";
import { NButton, NInput, NModal, NTooltip } from "naive-ui";
import { computed, reactive, watch } from "vue";
import ExprEditor from "@/components/ExprEditor";
import RequiredStar from "@/components/RequiredStar.vue";
import type { ConditionGroupExpr } from "@/plugins/cel";
import {
  buildCELExpr,
  emptySimpleExpr,
  resolveCELExpr,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import type { LocalApprovalRule } from "@/types";
import type { ApprovalFlow } from "@/types/proto-es/v1/issue_service_pb";
import { ApprovalFlowSchema } from "@/types/proto-es/v1/issue_service_pb";
import type { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import {
  batchConvertCELStringToParsedExpr,
  batchConvertParsedExprToCELString,
} from "@/utils";
import {
  getApprovalFactorList,
  getApprovalOptionConfigMap,
} from "../../common/utils";
import { StepsTable } from "../common";
import { useCustomApprovalContext } from "../context";
import {
  type ApprovalRuleTemplate,
  applyTemplateToState,
  filterTemplatesBySource,
  useApprovalRuleTemplates,
} from "./template";

type LocalState = {
  title: string;
  description: string;
  conditionExpr: ConditionGroupExpr;
  flow: ApprovalFlow;
};

const props = defineProps<{
  show: boolean;
  mode: "create" | "edit";
  source: WorkspaceApprovalSetting_Rule_Source;
  rule?: LocalApprovalRule;
  isFallback?: boolean;
}>();

const emit = defineEmits<{
  (event: "update:show", show: boolean): void;
  (event: "save", rule: Partial<LocalApprovalRule>): void;
}>();

const context = useCustomApprovalContext();
const { allowAdmin } = context;
const allTemplates = useApprovalRuleTemplates();
const templates = computed(() =>
  filterTemplatesBySource(allTemplates.value, props.source)
);

const state = reactive<LocalState>({
  title: "",
  description: "",
  conditionExpr: wrapAsGroup(emptySimpleExpr()),
  flow: createProto(ApprovalFlowSchema, { roles: [] }),
});

const factorList = computed(() => getApprovalFactorList(props.source));
const optionConfigMap = computed(() =>
  getApprovalOptionConfigMap(props.source)
);

const allowSave = computed(() => {
  if (!state.title.trim()) return false;
  if (!state.conditionExpr) return false;
  if (!validateSimpleExpr(state.conditionExpr)) return false;
  if (state.flow.roles.length === 0) return false;
  return true;
});

const resolveLocalState = async () => {
  // Reset to empty state immediately to unmount old Condition components.
  // This prevents component reuse issues when switching between rules,
  // where the ValueInput watch would reset values when factor/operator changes.
  state.title = "";
  state.description = "";
  state.conditionExpr = wrapAsGroup(emptySimpleExpr());
  state.flow = createProto(ApprovalFlowSchema, { roles: [] });

  if (props.rule) {
    state.title = props.rule.title || "";
    state.description = props.rule.description || "";
    if (props.rule.condition) {
      const parsedExprs = await batchConvertCELStringToParsedExpr([
        props.rule.condition,
      ]);
      const celExpr = head(parsedExprs);
      if (celExpr) {
        state.conditionExpr = wrapAsGroup(resolveCELExpr(celExpr));
      }
    }
    state.flow = cloneDeep(props.rule.flow);
  }
};

const handleUpdate = () => {
  // Trigger reactivity
};

const applyTemplate = (template: ApprovalRuleTemplate) => {
  const applied = applyTemplateToState(template);
  state.title = applied.title;
  state.conditionExpr = applied.conditionExpr;
  state.flow = applied.flow;
};

const handleSave = async () => {
  if (!context.hasFeature.value) {
    context.showFeatureModal.value = true;
    return;
  }

  const celexpr = await buildCELExpr(state.conditionExpr);
  if (!celexpr) {
    return;
  }

  const expressions = await batchConvertParsedExprToCELString([celexpr]);
  const condition = expressions[0];

  const ruleData: Partial<LocalApprovalRule> = {
    title: state.title,
    description: state.description,
    condition,
    conditionExpr: cloneDeep(state.conditionExpr),
    flow: cloneDeep(state.flow),
    source: props.source,
  };

  if (props.rule) {
    ruleData.uid = props.rule.uid;
  }

  emit("save", ruleData);
};

watch(
  () => props.show,
  (newShow) => {
    if (newShow) {
      resolveLocalState();
    }
  },
  { immediate: true }
);
</script>
