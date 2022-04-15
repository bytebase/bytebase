<template>
  <div>
    <div>
      <p class="textlabel">
        {{ $t("schame-review.create.configure-rule.from-template") }}
      </p>
      <div
        class="flex flex-col sm:flex-row justify-start items-center gap-x-10 gap-y-10 mt-4"
      >
        <div
          v-for="(template, index) in templateList"
          :key="template.name"
          class="cursor-pointer bg-transparent border border-gray-300 hover:bg-gray-100 rounded-lg p-6 transition-all flex flex-col justify-center items-center w-full sm:max-w-xs"
          @click="onTemplateApply(index)"
        >
          <img class="h-24" :src="template.image" alt="" />
          <span class="text-lg lg:text-xl mt-4">{{ template.name }}</span>
        </div>
      </div>
    </div>
    <div
      :class="
        selectRuleList.length > 0
          ? 'border-b-1 border-gray-200 border-t mt-7'
          : ''
      "
    >
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
});

const emit = defineEmits(["apply-template", "change"]);

const state = reactive<LocalState>({
  activeRuleId: "",
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
