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
      <div class="border-t border-gray-200 mt-8 pt-8">
        <p class="textlabel">
          {{ $t("schame-review.create.configure-rule.from-scratch") }}
        </p>
        <button
          type="button"
          class="btn-primary inline-flex justify-center w-64 mt-4"
          @click="state.openModal = true"
        >
          <div class="flex">
            <heroicons-solid:plus-circle class="w-5 h-5 mr-2" />
            {{ $t("schame-review.create.configure-rule.add-rule") }}
          </div>
        </button>
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
              @remove="(val) => $emit('remove', val)"
              @level-change="(level) => onLevelChange(rule, level)"
              @payload-change="(val) => onPayloadChange(rule, val)"
            />
          </li>
        </ul>
      </div>

      <button
        type="button"
        class="btn-normal inline-flex justify-center w-64"
        @click="state.openModal = true"
      >
        <div class="flex">
          <heroicons-solid:plus-circle class="w-5 h-5 mr-2" />
          {{ $t("schame-review.create.configure-rule.add-rule") }}
        </div>
      </button>
    </div>
    <BBModal
      :title="$t('schame-review.create.configure-rule.select-rule')"
      v-if="state.openModal"
      @close="state.openModal = false"
    >
      <SchemaRuleSelection
        @select="onRuleSelect"
        :selected-rule-id-list="selectedRuleIdList"
      />
    </BBModal>
  </div>
</template>

<script lang="ts" setup>
import { PropType, reactive, computed } from "vue";
import {
  SchemaRule,
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
  openModal: boolean;
  activeRuleId: string;
}

const props = defineProps({
  selectRuleList: {
    required: true,
    type: Object as PropType<SelectedRule[]>,
  },
});

const emit = defineEmits(["apply-template", "select", "remove", "change"]);

const state = reactive<LocalState>({
  openModal: false,
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

const onRuleSelect = (rule: SchemaRule) => {
  state.openModal = false;
  state.activeRuleId = rule.id;
  emit("select", rule);
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
