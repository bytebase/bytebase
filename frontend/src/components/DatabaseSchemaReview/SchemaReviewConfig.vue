<template>
  <div>
    <div v-if="selectRuleList.length === 0">
      <div>
        <p class="textlabel">
          {{ $t("schame-review.create.configure-rule.from-template") }}
        </p>
        <div
          class="flex flex-col sm:flex-row justify-start items-center gap-x-10 gap-y-10 mt-4"
        >
          <div
            v-for="template in templateList"
            :key="template.name"
            class="cursor-pointer bg-transparent border border-gray-300 hover:bg-gray-100 rounded-lg p-6 transition-all flex flex-col justify-center items-center w-full sm:max-w-xs"
            @click="onTemplateApply(template)"
          >
            <img class="h-24" :src="template.image" alt="" />
            <span class="text-lg lg:text-xl mt-4">{{ template.name }}</span>
          </div>
        </div>
      </div>
    </div>
    <div v-else>
      <div
        class="mb-5"
        :class="selectRuleList.length > 0 ? 'border-b-1 border-gray-200' : ''"
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
  </div>
</template>

<script lang="ts" setup>
import { PropType, reactive, computed } from "vue";
import {
  ruleList,
  SelectedRule,
  RulePayload,
  RuleLevel,
} from "../../types/schemaSystem";

interface Template {
  name: string;
  image: string;
  ruleList: SelectedRule[];
}

interface LocalState {
  activeRuleId: string;
}

const props = defineProps({
  selectRuleList: {
    required: true,
    type: Object as PropType<SelectedRule[]>,
  },
});

const emit = defineEmits(["apply-template", "change"]);

const state = reactive<LocalState>({
  activeRuleId: "",
});

const selectedRuleIdList = computed((): string[] => {
  return props.selectRuleList.map((r) => r.id);
});

const getRuleListWithLevel = (
  idList: string[],
  level: RuleLevel
): SelectedRule[] => {
  return idList.reduce((res, id) => {
    const rule = ruleList.find((r) => r.id === id);
    if (!rule) {
      return res;
    }
    res.push({
      ...rule,
      level,
    });
    return res;
  }, [] as SelectedRule[]);
};

const templateList: Template[] = [
  {
    name: "Schema review for Prod",
    image: new URL("../../assets/plan-enterprise.png", import.meta.url).href,
    ruleList: [
      ...getRuleListWithLevel(
        [
          "engine.mysql.use-innodb",
          "table.require-pk",
          "query.select.no-select-all",
          "query.where.require",
          "query.where.no-leading-wildcard-like",
        ],
        RuleLevel.Error
      ),
      ...getRuleListWithLevel(
        [
          "naming.table",
          "naming.column",
          "naming.index.pk",
          "naming.index.uk",
          "naming.index.idx",
          "column.required",
          "column.no-null",
        ],
        RuleLevel.Warning
      ),
    ],
  },
  {
    name: "Schema review for Dev",
    image: new URL("../../assets/plan-free.png", import.meta.url).href,
    ruleList: [
      ...getRuleListWithLevel(
        ["engine.mysql.use-innodb", "table.require-pk"],
        RuleLevel.Error
      ),
      ...getRuleListWithLevel(
        [
          "naming.table",
          "naming.column",
          "naming.index.pk",
          "naming.index.uk",
          "naming.index.idx",
          "column.required",
          "column.no-null",
          "query.select.no-select-all",
          "query.where.require",
          "query.where.no-leading-wildcard-like",
        ],
        RuleLevel.Warning
      ),
    ],
  },
];

const onTemplateApply = (template: Template) => {
  emit("apply-template", template.ruleList);
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
