<template>
  <FeatureAttention
    v-if="!hasSchemaReviewPolicyFeature"
    custom-class="my-5"
    feature="bb.feature.schema-review-policy"
    :description="
      $t('subscription.features.bb-feature-schema-review-policy.desc')
    "
  />
  <transition appear name="slide-from-bottom" mode="out-in">
    <SchemaReviewCreation
      v-if="state.editMode"
      :policy-id="reviewPolicy.id"
      :name="reviewPolicy.name"
      :selected-environment="reviewPolicy.environment"
      :selected-rule-list="selectedRuleList"
      @cancel="state.editMode = false"
    />
    <div class="my-5" v-else>
      <div class="flex flex-col items-center justify-center md:flex-row">
        <div class="flex-1 flex space-x-3 items-center justify-start">
          <h1 class="text-xl md:text-3xl font-semibold">
            {{ reviewPolicy.name }}
          </h1>
          <div v-if="reviewPolicy.rowStatus == 'ARCHIVED'">
            <BBBadge
              :text="$t('schema-review-policy.disabled')"
              :can-remove="false"
              :style="'DISABLED'"
            />
          </div>
        </div>
        <button
          v-if="hasPermission"
          type="button"
          class="btn-primary ml-5"
          @click="onEdit"
        >
          {{ $t("common.edit") }}
        </button>
      </div>
      <div
        class="flex items-center flex-wrap gap-x-3 my-5"
        v-if="reviewPolicy.environment"
      >
        <span class="font-semibold">{{ $t("common.environment") }}</span>
        <router-link
          :to="`/environment/${environmentSlug(reviewPolicy.environment)}`"
          class="col-span-2 font-medium text-main underline"
        >
          {{ environmentName(reviewPolicy.environment) }}
        </router-link>
      </div>
      <BBAttention
        v-else
        class="my-5"
        :style="`WARN`"
        :title="
          $t('schema-review-policy.create.basic-info.no-linked-environments')
        "
      />
      <div class="space-y-2 my-5">
        <span class="font-semibold">{{
          $t("schema-review-policy.filter")
        }}</span>
        <div class="flex flex-wrap gap-x-3">
          <span>{{ $t("common.database") }}:</span>
          <div
            v-for="engine in engineList"
            :key="engine.id"
            class="flex items-center"
          >
            <input
              type="checkbox"
              :id="engine.id"
              :value="engine.id"
              :checked="state.checkedEngine.has(engine.id)"
              @input="toggleCheckedEngine(engine.id)"
              class="h-4 w-4 border-gray-300 rounded text-indigo-600 focus:ring-indigo-500"
            />
            <label
              :for="engine.id"
              class="ml-2 items-center text-sm text-gray-600"
            >
              {{ $t(`engine.${engine.id.toLowerCase()}`) }}
              <span
                class="items-center px-2 py-0.5 rounded-full bg-gray-200 text-gray-800"
              >
                {{ engine.count }}
              </span>
            </label>
          </div>
        </div>
        <div class="flex flex-wrap gap-x-3">
          <span>{{ $t("schema-review-policy.error-level.name") }}:</span>
          <div
            v-for="level in errorLevelList"
            :key="level.id"
            class="flex items-center"
          >
            <input
              type="checkbox"
              :id="level.id"
              :value="level.id"
              :checked="state.checkedLevel.has(level.id)"
              @input="toggleCheckedLevel(level.id)"
              class="h-4 w-4 border-gray-300 rounded text-indigo-600 focus:ring-indigo-500"
            />
            <label
              :for="level.id"
              class="ml-2 items-center text-sm text-gray-600"
            >
              {{
                $t(`schema-review-policy.error-level.${level.id.toLowerCase()}`)
              }}
              <span
                class="items-center px-2 py-0.5 rounded-full bg-gray-200 text-gray-800"
              >
                {{ level.count }}
              </span>
            </label>
          </div>
        </div>
      </div>
      <div class="py-2 flex justify-between items-center mt-5">
        <SchemaReviewCategoryTabFilter
          :selected="state.selectedCategory"
          :category-list="categoryFilterList"
          @select="selectCategory"
        />
        <BBTableSearch
          ref="searchField"
          :placeholder="$t('schema-review-policy.search-rule-name')"
          @change-text="(text) => (state.searchText = text)"
        />
      </div>
      <SchemaReviewPreview
        :selected-rule-list="filteredSelectedRuleList"
        class="py-5"
      />
      <BBButtonConfirm
        v-if="reviewPolicy.rowStatus === 'NORMAL'"
        :style="'ARCHIVE'"
        :button-text="$t('schema-review-policy.disable')"
        confirm-description=""
        :ok-text="$t('common.disable')"
        :confirm-title="$t('common.disable') + ` '${reviewPolicy.name}'?`"
        :require-confirm="true"
        @confirm="onArchive"
      />
      <div class="flex gap-x-5" v-else>
        <BBButtonConfirm
          :style="'RESTORE'"
          :button-text="$t('schema-review-policy.enable')"
          confirm-description=""
          :ok-text="$t('common.enable')"
          :confirm-title="$t('common.enable') + ` '${reviewPolicy.name}'?`"
          :require-confirm="true"
          @confirm="onRestore"
        />
        <BBButtonConfirm
          :style="'DELETE'"
          :button-text="$t('schema-review-policy.delete')"
          :ok-text="$t('common.delete')"
          :confirm-title="$t('common.delete') + ` '${reviewPolicy.name}'?`"
          :require-confirm="true"
          @confirm="onRemove"
        />
      </div>
    </div>
  </transition>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { idFromSlug, environmentName, isDBAOrOwner } from "@/utils";
import {
  unknown,
  RuleType,
  LEVEL_LIST,
  RuleLevel,
  RuleTemplate,
  DatabaseSchemaReviewPolicy,
  SchemaRuleEngineType,
  convertToCategoryList,
  ruleTemplateList,
  convertPolicyRuleToRuleTemplate,
} from "@/types";
import {
  featureToRef,
  useCurrentUser,
  pushNotification,
  useSchemaSystemStore,
} from "@/store";
import { CategoryFilterItem } from "../components/DatabaseSchemaReview/components/SchemaReviewCategoryTabFilter.vue";

const props = defineProps({
  schemaReviewPolicySlug: {
    required: true,
    type: String,
  },
});

interface LocalState {
  searchText: string;
  selectedCategory?: string;
  editMode: boolean;
  checkedEngine: Set<SchemaRuleEngineType>;
  checkedLevel: Set<RuleLevel>;
}

const { t } = useI18n();
const store = useSchemaSystemStore();
const router = useRouter();
const currentUser = useCurrentUser();
const ROUTE_NAME = "setting.workspace.schema-review-policy";

const state = reactive<LocalState>({
  searchText: "",
  selectedCategory: router.currentRoute.value.query.category
    ? (router.currentRoute.value.query.category as string)
    : undefined,
  editMode: false,
  checkedEngine: new Set<SchemaRuleEngineType>(),
  checkedLevel: new Set<RuleLevel>(),
});

const hasSchemaReviewPolicyFeature = featureToRef(
  "bb.feature.schema-review-policy"
);

const hasPermission = computed(() => {
  return isDBAOrOwner(currentUser.value.role);
});

const reviewPolicy = computed((): DatabaseSchemaReviewPolicy => {
  return (
    store.getReviewPolicyByEnvironmentId(
      idFromSlug(props.schemaReviewPolicySlug)
    ) || (unknown("SCHEMA_REVIEW") as DatabaseSchemaReviewPolicy)
  );
});

const ruleMap = ruleTemplateList.reduce((map, rule) => {
  map.set(rule.type, rule);
  return map;
}, new Map<RuleType, RuleTemplate>());

const selectedRuleList = computed((): RuleTemplate[] => {
  if (!reviewPolicy.value) {
    return [];
  }

  const ruleTemplateList: RuleTemplate[] = [];

  for (const policyRule of reviewPolicy.value.ruleList) {
    const rule = ruleMap.get(policyRule.type);
    if (!rule) {
      continue;
    }

    const data = convertPolicyRuleToRuleTemplate(policyRule, rule);
    if (data) {
      ruleTemplateList.push(data);
    }
  }

  return ruleTemplateList;
});

const engineList = computed(
  (): { id: SchemaRuleEngineType; count: number }[] => {
    const tmp = selectedRuleList.value.reduce((dict, rule) => {
      if (!dict[rule.engine]) {
        dict[rule.engine] = {
          id: rule.engine,
          count: 0,
        };
      }
      dict[rule.engine].count += 1;
      return dict;
    }, {} as { [id: string]: { id: SchemaRuleEngineType; count: number } });

    return Object.values(tmp);
  }
);

const errorLevelList = computed((): { id: RuleLevel; count: number }[] => {
  return LEVEL_LIST.map((level) => ({
    id: level,
    count: selectedRuleList.value.filter((rule) => rule.level === level).length,
  }));
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
    name: t(`schema-review-policy.category.${c.id.toLowerCase()}`),
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
        state.checkedEngine.has(selectedRule.engine)) &&
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
  store.removeReviewPolicy(reviewPolicy.value.id);
  router.replace({
    name: ROUTE_NAME,
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("schema-review-policy.policy-removed"),
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
