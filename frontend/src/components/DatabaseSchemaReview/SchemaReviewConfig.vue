<template>
  <div>
    <SchemaReviewTemplates
      :required="true"
      v-if="selectRuleList.length === 0"
      :template-list="templateList"
      :title="$t('schema-review-policy.create.basic-info.choose-template')"
      @select="(index) => onTemplateApply(index)"
    />
    <div class="mb-5" v-if="selectRuleList.length > 0">
      <div
        class="flex cursor-pointer items-center px-2 text-indigo-500"
        @click="state.openTemplate = !state.openTemplate"
      >
        <heroicons-solid:chevron-right
          class="w-5 h-5 transform transition-all"
          :class="state.openTemplate ? 'rotate-90' : ''"
        />
        <span class="ml-1 text-sm font-medium">
          {{ $t("schema-review-policy.create.configure-rule.change-template") }}
        </span>
      </div>
      <SchemaReviewTemplates
        v-if="state.openTemplate"
        :required="false"
        :template-list="templateList"
        :selected-template-index="selectedTemplateIndex"
        @select="(index) => onTemplateApply(index)"
        class="mx-8 mt-5"
      />
    </div>
    <div :class="selectRuleList.length > 0 ? 'border-b-1 border-gray-200' : ''">
      <ul role="list" class="divide-y divide-gray-200">
        <li v-for="rule in selectRuleList" :key="rule.type">
          <SchemaRuleConfig
            :selected-rule="rule"
            :active="rule.type === state.activeRuleType"
            @activate="onRuleActivate"
            @level-change="(level) => onLevelChange(rule, level)"
            @payload-change="(val) => onPayloadChange(rule, val)"
          />
        </li>
      </ul>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PropType, reactive } from "vue";
import {
  RuleType,
  RuleLevel,
  RuleTemplate,
  RuleTemplatePayload,
  SchemaReviewPolicyTemplate,
} from "../../types/schemaSystem";

interface LocalState {
  activeRuleType: RuleType | null;
  openTemplate: boolean;
}

const props = defineProps({
  selectRuleList: {
    required: true,
    type: Object as PropType<RuleTemplate[]>,
  },
  templateList: {
    required: true,
    type: Object as PropType<SchemaReviewPolicyTemplate[]>,
  },
  selectedTemplateIndex: {
    required: true,
    type: Number,
  },
});

const emit = defineEmits(["apply-template", "change"]);

const state = reactive<LocalState>({
  activeRuleType: null,
  openTemplate: false,
});

const onTemplateApply = (index: number) => {
  emit("apply-template", index);
  state.activeRuleType = null;
};

const onRuleActivate = (type: RuleType) => {
  if (type === state.activeRuleType) {
    state.activeRuleType = null;
  } else {
    state.activeRuleType = type;
  }
};

const onPayloadChange = (rule: RuleTemplate, data: (string | string[])[]) => {
  if (!rule.componentList) {
    return;
  }

  const newRule: RuleTemplate = {
    ...rule,
    componentList: rule.componentList.reduce((list, component, index) => {
      let val = data[index];
      switch (component.payload.type) {
        case "STRING_ARRAY":
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: data[index] as string[],
            },
          });
          break;
        default:
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: data[index] as string,
            },
          });
          break;
      }
      return list;
    }, [] as RuleTemplatePayload[]),
  };

  emit("change", newRule);
};

const onLevelChange = (rule: RuleTemplate, level: RuleLevel) => {
  emit("change", {
    ...rule,
    level,
  });
};
</script>
