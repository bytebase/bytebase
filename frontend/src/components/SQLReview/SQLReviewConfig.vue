<template>
  <div>
    <div>
      <SQLReviewCategorySummaryFilter
        class="space-y-2 mb-5"
        :rule-list="selectedRuleList"
        :is-checked-engine="(engine) => state.checkedEngine.has(engine)"
        :is-checked-level="(level) => state.checkedLevel.has(level)"
        @toggle-checked-engine="(engine) => toggleCheckedEngine(engine)"
        @toggle-checked-level="(level) => toggleCheckedLevel(level)"
      />
      <div class="py-2 flex justify-between items-center mt-5">
        <SQLReviewCategoryTabFilter
          :selected="state.selectedCategory"
          :category-list="categoryFilterList"
          @select="selectCategory"
        />
        <BBTableSearch
          ref="searchField"
          :placeholder="$t('sql-review.search-rule-name')"
          @change-text="(text: string) => (state.searchText = text)"
        />
      </div>
    </div>
    <div class="flex gap-x-20">
      <SQLReviewSidebar :selected-rule-list="filteredSelectedRuleList" />
      <div class="flex-1">
        <SQLReviewTemplates
          v-if="selectedRuleList.length === 0"
          :required="true"
          :template-list="templateList"
          :title="$t('sql-review.create.basic-info.choose-template')"
          @select="(index) => onTemplateApply(index)"
        />
        <div v-if="selectedRuleList.length > 0" class="mb-5">
          <div
            class="flex cursor-pointer items-center text-indigo-500"
            @click="state.openTemplate = !state.openTemplate"
          >
            <heroicons-solid:chevron-right
              class="w-5 h-5 transform transition-all"
              :class="state.openTemplate ? 'rotate-90' : ''"
            />
            <span class="ml-1 text-sm font-medium">
              {{ $t("sql-review.create.configure-rule.change-template") }}
            </span>
          </div>
          <SQLReviewTemplates
            v-if="state.openTemplate"
            :required="false"
            :template-list="templateList"
            :selected-template-index="selectedTemplateIndex"
            class="mx-8 mt-5"
            @select="(index) => onTemplateApply(index)"
          />
        </div>
        <div
          :class="[
            'space-y-5',
            filteredSelectedRuleList.length > 0
              ? 'border-b-1 border-gray-200'
              : '',
          ]"
        >
          <div v-for="(category, index) in categoryList" :key="index">
            <div class="block text-2xl text-indigo-600 font-semibold px-2 mb-3">
              {{ $t(`sql-review.category.${category.id.toLowerCase()}`) }}
            </div>
            <ul role="list" class="divide-y divide-gray-200">
              <li v-for="rule in category.ruleList" :key="rule.type">
                <SQLRuleConfig
                  :selected-rule="rule"
                  :active="rule.type === state.activeRuleType"
                  :disabled="
                    !ruleIsAvailableInSubscription(
                      rule.type,
                      subscriptionStore.currentPlan
                    )
                  "
                  @activate="onRuleActivate"
                  @level-change="(level) => onLevelChange(rule, level)"
                  @payload-change="(val) => onPayloadChange(rule, val)"
                />
              </li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { PropType, reactive, computed } from "vue";
import {
  RuleType,
  RuleLevel,
  RuleTemplate,
  RuleConfigComponent,
  SQLReviewPolicyTemplate,
  convertToCategoryList,
  ruleIsAvailableInSubscription,
  SchemaRuleEngineType,
} from "@/types/sqlReview";
import { useSubscriptionStore } from "@/store";
import {
  type CategoryFilterItem,
  SQLReviewCategorySummaryFilter,
  SQLReviewSidebar,
  SQLReviewTemplates,
  SQLReviewCategoryTabFilter,
  SQLRuleConfig,
} from "./components/";
import { useI18n } from "vue-i18n";

interface LocalState {
  activeRuleType: RuleType | null;
  openTemplate: boolean;

  searchText: string;
  selectedCategory?: string;
  checkedEngine: Set<SchemaRuleEngineType>;
  checkedLevel: Set<RuleLevel>;
}

const props = defineProps({
  selectedRuleList: {
    required: true,
    type: Object as PropType<RuleTemplate[]>,
  },
  templateList: {
    required: true,
    type: Object as PropType<SQLReviewPolicyTemplate[]>,
  },
  selectedTemplateIndex: {
    required: true,
    type: Number,
  },
});

const emit = defineEmits(["apply-template", "change"]);
const { t } = useI18n();

const subscriptionStore = useSubscriptionStore();

const state = reactive<LocalState>({
  activeRuleType: null,
  openTemplate: false,
  searchText: "",
  selectedCategory: undefined,
  checkedEngine: new Set<SchemaRuleEngineType>(),
  checkedLevel: new Set<RuleLevel>(),
});

const categoryList = computed(() => {
  return convertToCategoryList(filteredSelectedRuleList.value);
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

const onPayloadChange = (
  rule: RuleTemplate,
  data: (string | number | boolean | string[])[]
) => {
  if (!rule.componentList) {
    return;
  }

  const newRule: RuleTemplate = {
    ...rule,
    componentList: rule.componentList.reduce((list, component, index) => {
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
        case "NUMBER":
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: data[index] as number,
            },
          });
          break;
        case "BOOLEAN":
          list.push({
            ...component,
            payload: {
              ...component.payload,
              value: data[index] as boolean,
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
    }, [] as RuleConfigComponent[]),
  };

  emit("change", newRule);
};

const onLevelChange = (rule: RuleTemplate, level: RuleLevel) => {
  emit("change", {
    ...rule,
    level,
  });
};

const toggleCheckedEngine = (engine: SchemaRuleEngineType) => {
  if (state.checkedEngine.has(engine)) {
    state.checkedEngine.delete(engine);
  } else {
    state.checkedEngine.add(engine);
  }
};

const toggleCheckedLevel = (level: RuleLevel) => {
  if (state.checkedLevel.has(level)) {
    state.checkedLevel.delete(level);
  } else {
    state.checkedLevel.add(level);
  }
};

const categoryFilterList = computed((): CategoryFilterItem[] => {
  return convertToCategoryList(props.selectedRuleList).map((c) => ({
    id: c.id,
    name: t(`sql-review.category.${c.id.toLowerCase()}`),
  }));
});

const selectCategory = (category: string) => {
  state.selectedCategory = category;
};

const filteredSelectedRuleList = computed((): RuleTemplate[] => {
  return props.selectedRuleList.filter((selectedRule) => {
    if (
      !state.selectedCategory &&
      !state.searchText &&
      state.checkedEngine.size === 0 &&
      state.checkedLevel.size === 0
    ) {
      // Select "All"
      return true;
    }

    return (
      (!state.selectedCategory ||
        selectedRule.category === state.selectedCategory) &&
      (!state.searchText ||
        selectedRule.type
          .toLowerCase()
          .includes(state.searchText.toLowerCase())) &&
      (state.checkedEngine.size === 0 ||
        selectedRule.engineList.some((engine) =>
          state.checkedEngine.has(engine)
        )) &&
      (state.checkedLevel.size === 0 ||
        state.checkedLevel.has(selectedRule.level))
    );
  });
});
</script>
