<template>
  <transition appear name="slide-from-bottom" mode="out-in">
    <SchemaReviewCreation
      v-if="state.editMode"
      :review-id="review.id"
      :name="review.name"
      :selected-environment-list="environmentList"
      :selected-rule-list="selectedRuleList"
      @cancel="state.editMode = false"
    />
    <div class="my-5" v-else>
      <div class="flex flex-col items-center justify-center md:flex-row">
        <h1 class="text-xl md:text-3xl font-semibold flex-1">
          {{ review.name }}
        </h1>
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
        class="flex flex-wrap gap-x-3 my-5"
        v-if="environmentList.length > 0"
      >
        <span class="font-semibold">{{ $t("common.environments") }}</span>
        <BBBadge
          v-for="env in environmentList"
          :key="env.id"
          :text="env.name"
          :can-remove="false"
        />
      </div>
      <BBAttention
        v-else
        class="my-5"
        :style="`WARN`"
        :title="$t('common.environments')"
        :description="
          $t('schema-review.create.basic-info.no-selected-environments')
        "
      />
      <div class="space-y-2 my-5">
        <span class="font-semibold">{{ $t("schema-review.filter") }}</span>
        <div class="flex flex-wrap gap-x-3">
          <span>{{ $t("schema-review.database") }}:</span>
          <div v-for="db in engineList" :key="db.id" class="flex items-center">
            <input
              type="checkbox"
              :id="db.id"
              :value="db.id"
              :checked="state.checkedEngine.has(db.id)"
              @input="toggleCheckedEngine(db.id)"
              class="h-4 w-4 border-gray-300 rounded text-indigo-600 focus:ring-indigo-500"
            />
            <label :for="db.id" class="ml-2 items-center text-sm text-gray-600">
              {{ db.id }}
              <span
                class="items-center px-2 py-0.5 rounded-full bg-gray-200 text-gray-800"
              >
                {{ db.count }}
              </span>
            </label>
          </div>
        </div>
        <div class="flex flex-wrap gap-x-3">
          <span>{{ $t("schema-review.error-level.name") }}:</span>
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
              {{ $t(`schema-review.error-level.${level.id}`) }}
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
          :placeholder="$t('schema-review.search-rule-name')"
          @change-text="(text) => (state.searchText = text)"
        />
      </div>
      <SchemaReviewPreview
        :selected-rule-list="filteredSelectedRuleList"
        class="py-5"
      />
      <div v-if="environmentList.length > 0" class="textinfolabel">
        {{ $t("schema-review.delete-attention") }}
      </div>
      <BBButtonConfirm
        v-else
        :style="'DELETE'"
        :button-text="$t('schema-review.delete')"
        :ok-text="$t('common.delete')"
        :confirm-title="$t('common.delete') + ` '${review.name}'?`"
        :require-confirm="true"
        @confirm="onRemove"
      />
    </div>
  </transition>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { idFromSlug, environmentName, isOwner, isDBA } from "../utils";
import {
  LEVEL_LIST,
  RuleLevel,
  SchemaRule,
  DatabaseSchemaReviewPolicy,
  SchemaRuleEngineType,
  convertToCategoryList,
  SelectedRule,
  ruleList,
  RulePayload,
  Environment,
} from "../types";
import {
  useCurrentUser,
  pushNotification,
  useEnvironmentStore,
  useSchemaSystemStore,
} from "@/store";
import { CategoryFilterItem } from "../components/DatabaseSchemaReview/components/SchemaReviewCategoryTabFilter.vue";

const props = defineProps({
  schemaReviewSlug: {
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
const envStore = useEnvironmentStore();
const router = useRouter();
const currentUser = useCurrentUser();
const ROUTE_NAME = "setting.workspace.schema-review";

const state = reactive<LocalState>({
  searchText: "",
  selectedCategory: router.currentRoute.value.query.category
    ? (router.currentRoute.value.query.category as string)
    : undefined,
  editMode: false,
  checkedEngine: new Set<SchemaRuleEngineType>(),
  checkedLevel: new Set<RuleLevel>(),
});

const hasPermission = computed(() => {
  return isOwner(currentUser.value.role) || isDBA(currentUser.value.role);
});

const review = computed((): DatabaseSchemaReviewPolicy => {
  return store.getReviewById(idFromSlug(props.schemaReviewSlug));
});

const environmentList = computed((): Environment[] => {
  return review.value.environmentList.map((envId) => {
    const env = envStore.getEnvironmentById(envId);
    return {
      ...env,
      name: environmentName(env),
    };
  });
});

const ruleMap = ruleList.reduce((map, rule) => {
  map.set(rule.id, rule);
  return map;
}, new Map<string, SchemaRule>());

const selectedRuleList = computed((): SelectedRule[] => {
  if (!review.value) {
    return [];
  }

  const res: SelectedRule[] = [];

  for (const selectedRule of review.value.ruleList) {
    const rule = ruleMap.get(selectedRule.id);
    if (!rule) {
      continue;
    }
    res.push({
      ...rule,
      level: selectedRule.level,
      payload: rule.payload
        ? Object.entries(rule.payload).reduce((obj, [key, val]) => {
            obj[key] = {
              ...val,
              value: selectedRule.payload
                ? selectedRule.payload[key]
                : undefined,
            };
            return obj;
          }, {} as RulePayload)
        : undefined,
    });
  }

  return res;
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
    name: c.name,
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

const filteredSelectedRuleList = computed((): SelectedRule[] => {
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
        selectedRule.id
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

const onRemove = () => {
  if (environmentList.value.length > 0) {
    return;
  }
  store.removeReview(review.value.id);
  router.replace({
    name: ROUTE_NAME,
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("schema-review.remove-review"),
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
