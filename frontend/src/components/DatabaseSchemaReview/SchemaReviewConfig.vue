<template>
  <div>
    <SchemaReviewTemplates
      :required="true"
      v-if="selectRuleList.length === 0"
      :template-list="templateList"
      :title="$t('schema-review.create.configure-rule.from-template')"
      @select="(index) => onTemplateApply(index)"
    />
    <div :class="selectRuleList.length > 0 ? 'border-b-1 border-gray-200' : ''">
      <ul role="list" class="divide-y divide-gray-200">
        <li v-for="rule in selectRuleList" :key="rule.id" class="">
          <SchemaRuleConfig
            :selected-rule="rule"
            :active="rule.id === state.activeRuleId"
            @activate="onRuleActivate"
            @level-change="(level) => onLevelChange(rule, level)"
            @payload-change="(val) => onPayloadChange(rule, val)"
          />
        </li>
      </ul>
    </div>
    <div class="mt-5" v-if="selectRuleList.length > 0">
      <div
        class="flex cursor-pointer items-center px-2 text-indigo-500"
        @click="state.openTemplate = !state.openTemplate"
      >
        <heroicons-solid:chevron-right
          class="w-5 h-5 transform transition-all"
          :class="state.openTemplate ? 'rotate-90' : ''"
        />
        <span class="ml-1 text-sm font-medium">
          {{ $t("schema-review.create.configure-rule.change-template") }}
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
  </div>
</template>

<script lang="ts" setup>
import { PropType, reactive, computed } from "vue";
import {
  SelectedRule,
  RulePayload,
  RuleLevel,
  SchemaReviewTemplate,
} from "../../types/schemaSystem";

interface LocalState {
  activeRuleId: string;
  openTemplate: boolean;
}

const props = defineProps({
  selectRuleList: {
    required: true,
    type: Object as PropType<SelectedRule[]>,
  },
  templateList: {
    required: true,
    type: Object as PropType<SchemaReviewTemplate[]>,
  },
  selectedTemplateIndex: {
    required: true,
    type: Number,
  },
});

const emit = defineEmits(["apply-template", "change"]);

const state = reactive<LocalState>({
  activeRuleId: "",
  openTemplate: false,
});

const selectedRuleIdList = computed((): string[] => {
  return props.selectRuleList.map((r) => r.id);
});

const onTemplateApply = (index: number) => {
  emit("apply-template", index);
  state.activeRuleId = "";
};

const onRuleActivate = (id: string) => {
  if (id === state.activeRuleId) {
    state.activeRuleId = "";
  } else {
    state.activeRuleId = id;
  }
};

const onPayloadChange = (rule: SelectedRule, data: { [val: string]: any }) => {
  if (!rule.payload) {
    return;
  }

  const newRule = {
    ...rule,
    payload: Object.entries(rule.payload).reduce((dict, [key, val]) => {
      dict[key] = { ...val };
      dict[key].value = data[key];
      return dict;
    }, {} as RulePayload),
  };

  emit("change", newRule);
};

const onLevelChange = (rule: SelectedRule, level: RuleLevel) => {
  emit("change", {
    ...rule,
    level,
  });
};
</script>
