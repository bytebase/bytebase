<template>
  <transition appear name="slide-from-bottom" mode="out-in">
    <SchemaGuideCreation
      v-if="state.editMode"
      :id="guide.id"
      :name="guide.name"
      :selectedEnvNameList="
        guide.environmentList.map((id) => environmentNameFromId(id))
      "
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
          Edit
        </button>
      </div>
      <div class="flex flex-wrap gap-x-2 my-5">
        <span>Environments:</span>
        <BBBadge
          v-for="envId in guide.environmentList"
          :key="envId"
          :text="environmentNameFromId(envId)"
          :can-remove="false"
        />
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
import { idFromSlug, environmentName } from "../utils";
import {
  Rule,
  EnvironmentId,
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
}

const { t } = useI18n();
const store = useSchemaSystemStore();
const router = useRouter();
const ROUTE_NAME = "setting.workspace.database-review-guide";

const state = reactive<LocalState>({
  searchText: "",
  selectedCategory: router.currentRoute.value.query.category
    ? (router.currentRoute.value.query.category as string)
    : undefined,
  editMode: false,
});

const environmentNameFromId = function (id: EnvironmentId) {
  const env = useEnvironmentStore().getEnvironmentById(id);

  return environmentName(env);
};

const guide = computed((): DatabaseSchemaGuide => {
  return store.getGuideById(idFromSlug(props.schemaGuideSlug));
});

const ruleMap = ruleList.reduce((map, rule) => {
  map.set(rule.id, rule);
  return map;
}, new Map<string, Rule>());

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
    if (!state.selectedCategory && !state.searchText) {
      // Select "All"
      return true;
    }

    return (
      (!state.selectedCategory ||
        selectedRule.category === state.selectedCategory) &&
      (!state.searchText ||
        selectedRule.id.toLowerCase().includes(state.searchText.toLowerCase()))
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
