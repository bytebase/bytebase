<template>
  <transition appear name="slide-from-bottom" mode="out-in">
    <SchemaGuideCreation
      v-if="state.editMode"
      :id="guide.id"
      :name="guide.name"
      :selectedEnvNameList="envNameList"
      :selectedRuleList="selectedRuleList"
      @cancel="state.editMode = false"
    />
    <div class="my-5" v-else>
      <div class="flex flex-col items-center justify-center md:flex-row">
        <h1 class="text-xl md:text-3xl font-semibold flex-1">
          {{ guide.name }}
        </h1>

        <BBButtonConfirm
          :style="'DELETE'"
          :button-text="$t('common.delete')"
          :ok-text="$t('common.delete')"
          :confirm-title="$t('common.delete') + ` '${guide.name}'?`"
          :require-confirm="true"
          @confirm="onRemove"
        />
        <button type="button" class="btn-primary ml-5" @click="onEdit">
          {{ $t("common.edit") }}
        </button>
      </div>
      <div class="flex flex-wrap gap-x-3 my-5">
        <span>{{ $t("common.environments") }}:</span>
        <BBBadge
          v-for="envName in envNameList"
          :key="envName"
          :text="envName"
          :can-remove="false"
        />
      </div>
      <div class="flex flex-wrap gap-x-3 my-5">
        <span>{{ $t("database-review-guide.filter-by-database") }}:</span>
        <div v-for="db in databaseList" :key="db" class="flex items-center">
          <input
            type="checkbox"
            :id="db"
            :value="db"
            :checked="state.checkedDatabase.has(db)"
            @input="toggleCheckedDatabase(db)"
            class="h-4 w-4 border-gray-300 rounded text-indigo-600 focus:ring-indigo-500"
          />
          <label :for="db" class="ml-2 items-center text-sm text-gray-600">
            {{ db }}
          </label>
        </div>
      </div>
      <div class="py-2 flex justify-between items-center mt-5">
        <SchemaGuideCategoryTabFilter
          :selected="state.selectedCategory"
          :category-list="categoryFilterList"
          @select="selectCategory"
        />
        <BBTableSearch
          ref="searchField"
          :placeholder="$t('database-review-guide.search-rule-name')"
          @change-text="(text) => (state.searchText = text)"
        />
      </div>
      <SchemaGuidePreview :rule-list="filteredSelectedRuleList" class="py-5" />
    </div>
  </transition>
</template>

<script lang="ts" setup>
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { idFromSlug } from "../utils";
import {
  DatabaseType,
  SchemaRule,
  DatabaseSchemaGuide,
  convertToCategoryList,
  SelectedRule,
  ruleList,
  RulePayload,
} from "../types";
import {
  pushNotification,
  useEnvironmentStore,
  useSchemaSystemStore,
} from "@/store";
import { CategoryFilterItem } from "../components/DatabaseSchemaGuide/SchemaGuideCategoryTabFilter.vue";

const props = defineProps({
  schemaGuideSlug: {
    required: true,
    type: String,
  },
});

interface LocalState {
  searchText: string;
  selectedCategory?: string;
  editMode: boolean;
  checkedDatabase: Set<DatabaseType>;
}

const { t } = useI18n();
const store = useSchemaSystemStore();
const envStore = useEnvironmentStore();
const router = useRouter();
const ROUTE_NAME = "setting.workspace.database-review-guide";

const state = reactive<LocalState>({
  searchText: "",
  selectedCategory: router.currentRoute.value.query.category
    ? (router.currentRoute.value.query.category as string)
    : undefined,
  editMode: false,
  checkedDatabase: new Set<DatabaseType>(),
});

const guide = computed((): DatabaseSchemaGuide => {
  return store.getGuideById(idFromSlug(props.schemaGuideSlug));
});

const envNameList = computed((): string[] => {
  return guide.value.environmentList.map((envId) =>
    envStore.getEnvironmentNameById(envId)
  );
});

const ruleMap = ruleList.reduce((map, rule) => {
  map.set(rule.id, rule);
  return map;
}, new Map<string, SchemaRule>());

const selectedRuleList = computed((): SelectedRule[] => {
  if (!guide.value) {
    return [];
  }

  const res: SelectedRule[] = [];

  for (const selectedRule of guide.value.ruleList) {
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

const databaseList = computed((): DatabaseType[] => {
  return [
    ...new Set(
      selectedRuleList.value.reduce((res, rule) => {
        res.push(...rule.database);
        return res;
      }, [] as DatabaseType[])
    ),
  ];
});

const toggleCheckedDatabase = (db: DatabaseType) => {
  if (state.checkedDatabase.has(db)) {
    state.checkedDatabase.delete(db);
  } else {
    state.checkedDatabase.add(db);
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
      state.checkedDatabase.size === 0
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
      (state.checkedDatabase.size === 0 ||
        selectedRule.database.some((db) => state.checkedDatabase.has(db)))
    );
  });
});

const onEdit = () => {
  state.editMode = true;
  state.searchText = "";
  state.selectedCategory = undefined;
};

const onRemove = () => {
  store.removeGuideline(guide.value.id);
  router.replace({
    name: ROUTE_NAME,
  });
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("database-review-guide.remove-guideline"),
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
