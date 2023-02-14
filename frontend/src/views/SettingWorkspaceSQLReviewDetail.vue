<template>
  <FeatureAttention
    v-if="!hasSQLReviewPolicyFeature"
    custom-class="my-5"
    feature="bb.feature.sql-review"
    :description="$t('subscription.features.bb-feature-sql-review.desc')"
  />
  <transition appear name="slide-from-bottom" mode="out-in">
    <SQLReviewCreation
      v-if="state.editMode"
      :policy-id="reviewPolicy.id"
      :name="reviewPolicy.name"
      :selected-environment="reviewPolicy.environment"
      :selected-rule-list="selectedRuleList"
      @cancel="state.editMode = false"
    />
    <div v-else class="my-5">
      <div
        class="flex flex-col items-center space-x-2 justify-center md:flex-row"
      >
        <div class="flex-1 flex space-x-3 items-center justify-start">
          <h1 class="text-xl md:text-3xl font-semibold">
            {{ reviewPolicy.name }}
          </h1>
          <div v-if="reviewPolicy.rowStatus == 'ARCHIVED'">
            <BBBadge
              :text="$t('sql-review.disabled')"
              :can-remove="false"
              :style="'DISABLED'"
            />
          </div>
        </div>
        <div v-if="hasPermission" class="flex space-x-2">
          <button
            v-if="reviewPolicy.rowStatus === 'NORMAL'"
            type="button"
            class="btn-normal py-2 px-4"
            @click.prevent="state.showDisableModal = true"
          >
            {{ $t("common.disable") }}
          </button>
          <button
            v-else
            type="button"
            class="btn-normal py-2 px-4"
            @click.prevent="state.showEnableModal = true"
          >
            {{ $t("common.enable") }}
          </button>
          <button type="button" class="btn-primary" @click="onEdit">
            {{ $t("common.edit") }}
          </button>
        </div>
      </div>
      <div
        v-if="reviewPolicy.environment"
        class="flex items-center flex-wrap gap-x-3 my-5"
      >
        <span class="font-semibold">{{ $t("common.environment") }}</span>
        <router-link
          :to="`/environment/${environmentSlug(reviewPolicy.environment)}`"
          class="col-span-2 font-medium text-main underline"
        >
          {{ environmentName(reviewPolicy.environment) }}
        </router-link>
        <SQLRuleLevelSummary :rule-list="filteredSelectedRuleList" />
      </div>
      <BBAttention
        v-else
        class="my-5"
        :style="`WARN`"
        :title="$t('sql-review.create.basic-info.no-linked-environments')"
      />
      <SQLReviewCategorySummaryFilter
        class="space-y-2 my-5"
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
      <SQLReviewPreview
        :selected-rule-list="filteredSelectedRuleList"
        class="py-5"
      />
      <BBButtonConfirm
        v-if="reviewPolicy.rowStatus === 'ARCHIVED' && hasPermission"
        :style="'DELETE'"
        :button-text="$t('sql-review.delete')"
        :ok-text="$t('common.delete')"
        :confirm-title="$t('common.delete') + ` '${reviewPolicy.name}'?`"
        :require-confirm="true"
        @confirm="onRemove"
      />
    </div>
  </transition>
  <BBAlert
    v-if="state.showDisableModal"
    :style="'CRITICAL'"
    :ok-text="$t('common.disable')"
    :title="$t('common.disable') + ` '${reviewPolicy.name}'?`"
    description=""
    @ok="
      () => {
        state.showDisableModal = false;
        onArchive();
      }
    "
    @cancel="state.showDisableModal = false"
  >
  </BBAlert>
  <BBAlert
    v-if="state.showEnableModal"
    :style="'INFO'"
    :ok-text="$t('common.enable')"
    :title="$t('common.enable') + ` '${reviewPolicy.name}'?`"
    description=""
    @ok="
      () => {
        state.showEnableModal = false;
        onRestore();
      }
    "
    @cancel="state.showEnableModal = false"
  >
  </BBAlert>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { idFromSlug, environmentName, hasWorkspacePermission } from "@/utils";
import {
  unknown,
  RuleLevel,
  RuleTemplate,
  SQLReviewPolicy,
  SchemaRuleEngineType,
  convertToCategoryList,
  RuleType,
  TEMPLATE_LIST,
  convertPolicyRuleToRuleTemplate,
} from "@/types";
import {
  featureToRef,
  useCurrentUser,
  pushNotification,
  useSQLReviewStore,
} from "@/store";
import {
  CategoryFilterItem,
  SQLRuleLevelSummary,
  SQLReviewCategorySummaryFilter,
} from "../components/SQLReview/components";

const props = defineProps({
  sqlReviewPolicySlug: {
    required: true,
    type: String,
  },
});

interface LocalState {
  showDisableModal: boolean;
  showEnableModal: boolean;
  searchText: string;
  selectedCategory?: string;
  editMode: boolean;
  checkedEngine: Set<SchemaRuleEngineType>;
  checkedLevel: Set<RuleLevel>;
}

const { t } = useI18n();
const store = useSQLReviewStore();
const router = useRouter();
const currentUser = useCurrentUser();
const ROUTE_NAME = "setting.workspace.sql-review";

const state = reactive<LocalState>({
  showDisableModal: false,
  showEnableModal: false,
  searchText: "",
  selectedCategory: router.currentRoute.value.query.category
    ? (router.currentRoute.value.query.category as string)
    : undefined,
  editMode: false,
  checkedEngine: new Set<SchemaRuleEngineType>(),
  checkedLevel: new Set<RuleLevel>(),
});

const hasSQLReviewPolicyFeature = featureToRef("bb.feature.sql-review");

const hasPermission = computed(() => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-sql-review-policy",
    currentUser.value.role
  );
});

const reviewPolicy = computed((): SQLReviewPolicy => {
  return (
    store.getReviewPolicyByEnvironmentId(
      idFromSlug(props.sqlReviewPolicySlug)
    ) || (unknown("SQL_REVIEW") as SQLReviewPolicy)
  );
});

const selectedRuleList = computed((): RuleTemplate[] => {
  if (!reviewPolicy.value) {
    return [];
  }

  const ruleTemplateList: RuleTemplate[] = [];
  const ruleTemplateMap: Map<RuleType, RuleTemplate> = TEMPLATE_LIST.reduce(
    (map, template) => {
      for (const rule of template.ruleList) {
        map.set(rule.type, rule);
      }
      return map;
    },
    new Map<RuleType, RuleTemplate>()
  );

  for (const policyRule of reviewPolicy.value.ruleList) {
    const rule = ruleTemplateMap.get(policyRule.type);
    if (!rule) {
      continue;
    }

    const data = convertPolicyRuleToRuleTemplate(policyRule, rule);
    if (data) {
      ruleTemplateList.push(data);
    }
    ruleTemplateMap.delete(policyRule.type);
  }

  for (const rule of ruleTemplateMap.values()) {
    ruleTemplateList.push({
      ...rule,
      level: RuleLevel.DISABLED,
    });
  }

  return ruleTemplateList;
});

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
  return convertToCategoryList(selectedRuleList.value).map((c) => ({
    id: c.id,
    name: t(`sql-review.category.${c.id.toLowerCase()}`),
  }));
});

const selectCategory = (category: string) => {
  state.selectedCategory = category;
  if (category) {
    router.replace({
      name: `${ROUTE_NAME}.detail`,
      query: {
        category,
      },
    });
  } else {
    router.replace({
      name: `${ROUTE_NAME}.detail`,
    });
  }
};

const filteredSelectedRuleList = computed((): RuleTemplate[] => {
  return selectedRuleList.value.filter((selectedRule) => {
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

const onEdit = () => {
  state.editMode = true;
  state.searchText = "";
  state.selectedCategory = undefined;
};

const onArchive = () => {
  store.updateReviewPolicy({
    id: reviewPolicy.value.id,
    rowStatus: "ARCHIVED",
  });
};

const onRestore = () => {
  store.updateReviewPolicy({
    id: reviewPolicy.value.id,
    rowStatus: "NORMAL",
  });
};

const onRemove = () => {
  store.removeReviewPolicy(reviewPolicy.value.id).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("sql-review.policy-removed"),
    });
  });
  router.replace({
    name: ROUTE_NAME,
  });
};
</script>

<style scoped>
.slide-from-bottom-enter-active {
  transition: all 0.2s ease-in-out;
}

.slide-from-bottom-leave-active {
  transition: all 0.2s ease-in-out;
}

.slide-from-bottom-enter-from,
.slide-from-bottom-leave-to {
  transform: translateY(20px);
  opacity: 0;
}
</style>
