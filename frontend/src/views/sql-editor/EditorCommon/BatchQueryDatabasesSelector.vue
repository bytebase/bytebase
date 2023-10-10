<template>
  <NPopover
    placement="bottom"
    :disabled="!hasBatchQueryFeature"
    trigger="click"
  >
    <template #trigger>
      <div
        class="flex flex-row justify-start items-center gap-1"
        @click="handleTriggerClick"
      >
        <div
          class="!ml-2 w-auto h-6 px-2 border border-gray-400 flex flex-row justify-center items-center gap-1 cursor-pointer rounded hover:opacity-80"
          :class="
            matchedDatabases.length > 0 && hasBatchQueryFeature
              ? 'text-accent bg-blue-50 shadow !border-accent'
              : 'text-gray-600'
          "
        >
          <span>{{ $t("sql-editor.batch-query.batch") }}</span>
          <span v-if="matchedDatabases.length > 0">
            ({{ matchedDatabases.length }})
          </span>
          <FeatureBadge feature="bb.feature.batch-query" />
        </div>
      </div>
    </template>
    <div class="w-128 max-h-128 overflow-y-auto p-1 pb-2">
      <p class="text-gray-500 mb-1 w-full leading-4">
        <span class="mr-1">{{
          $t("sql-editor.batch-query.description", {
            count: matchedDatabases.length,
          })
        }}</span>
        <LearnMoreLink
          url="https://www.bytebase.com/docs/sql-editor/batch-query?source=console"
          class="text-sm"
        />
      </p>
      <div class="w-full flex flex-col justify-start items-start">
        <div class="w-full">
          <p class="font-medium">
            {{
              labels.length > 0
                ? $t("sql-editor.batch-query.database-labels", {
                    database: selectedDatabase.databaseName,
                  })
                : $t("sql-editor.batch-query.database-has-no-label", {
                    database: selectedDatabase.databaseName,
                  })
            }}
          </p>
          <NCheckboxGroup v-model:value="selectedLabelsValue">
            <NSpace class="flex">
              <NCheckbox
                v-for="label in labels"
                :key="`${label.key}-${label.value}`"
                :value="`${label.key}-${label.value}`"
                :label="`${getFormattedLabelKey(label.key)}:${label.value}`"
              >
              </NCheckbox>
            </NSpace>
          </NCheckboxGroup>
        </div>
        <NDivider class="!my-3" />
        <div class="w-full">
          <MatchedDatabaseView
            :hide-title="true"
            :matched-database-list="matchedDatabases"
            :unmatched-database-list="unmatchedDatabases"
          />
        </div>
      </div>
    </div>
  </NPopover>

  <FeatureModal
    feature="bb.feature.batch-query"
    :open="state.showFeatureModal"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { isEqual } from "lodash-es";
import {
  NCheckboxGroup,
  NCheckbox,
  NPopover,
  NSpace,
  NDivider,
} from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import MatchedDatabaseView from "@/components/DatabaseGroup/MatchedDatabaseView.vue";
import {
  hasFeature,
  useCurrentUserIamPolicy,
  useDatabaseV1ByUID,
  useDatabaseV1Store,
  useTabStore,
} from "@/store/modules";
import { displayLabelKey } from "@/utils";

interface LocalState {
  showFeatureModal: boolean;
}

const databaseStore = useDatabaseV1Store();
const tabStore = useTabStore();
const currentUserIamPolicy = useCurrentUserIamPolicy();
const state = reactive<LocalState>({
  showFeatureModal: false,
});
// Save the stringified label key-value pairs.
const currentTab = computed(() => tabStore.currentTab);
const connection = computed(() => currentTab.value.connection);
const selectedLabelsValue = ref<string[]>([]);

const hasBatchQueryFeature = hasFeature("bb.feature.batch-query");

const { database: selectedDatabase } = useDatabaseV1ByUID(
  computed(() => String(connection.value.databaseId))
);

const project = computed(() => selectedDatabase.value.projectEntity);

const databases = computed(() => {
  return databaseStore.databaseListByProject(project.value.name);
});

const filteredDatabaseList = computed(() => {
  return (
    databases.value
      // Don't show the currently selected database.
      .filter((db) => db.uid !== selectedDatabase.value.uid)
      // Only show databases that the user has permission to query.
      .filter((db) => currentUserIamPolicy.allowToQueryDatabaseV1(db))
      // Only show databases with same engine.
      .filter(
        (db) =>
          db.instanceEntity.engine ===
          selectedDatabase.value.instanceEntity.engine
      )
      // Only show databases that have at least one same label.
      .filter((db) =>
        getFilteredDatabaseLabels(db.labels).some((label) =>
          labels.value.find((raw) => isEqual(raw, label))
        )
      )
  );
});

const labels = computed(() => {
  return getFilteredDatabaseLabels(selectedDatabase.value.labels);
});

const handleTriggerClick = () => {
  if (!hasBatchQueryFeature) {
    state.showFeatureModal = true;
  }
};

const getFilteredDatabaseLabels = (labels: { [key: string]: string }) => {
  // Filter out the environment label.
  const keys = Object.keys(labels).filter((key) => {
    return key !== "environment";
  });
  return keys.map((key) => {
    return {
      key,
      value: labels[key],
    };
  });
};

const matchedDatabases = computed(() => {
  return filteredDatabaseList.value.filter((db) => {
    return selectedLabelsValue.value.find((labelString) => {
      // Filter out the environment label.
      const keys = Object.keys(db.labels).filter((key) => {
        return key !== "environment";
      });
      return keys
        .map((key) => {
          return {
            key,
            value: db.labels[key],
          };
        })
        .find((label) => {
          return `${label.key}-${label.value}` === labelString;
        });
    });
  });
});

const unmatchedDatabases = computed(() => {
  return filteredDatabaseList.value.filter((db) => {
    return matchedDatabases.value.indexOf(db) === -1;
  });
});

const getFormattedLabelKey = (labelKey: string) => {
  return displayLabelKey(labelKey);
};

watch(selectedLabelsValue, () => {
  tabStore.updateCurrentTab({
    batchQueryContext: {
      selectedLabels: selectedLabelsValue.value,
    },
  });
});

watch(
  () => currentTab.value.batchQueryContext?.selectedLabels,
  () => {
    selectedLabelsValue.value =
      currentTab.value.batchQueryContext?.selectedLabels || [];
  },
  {
    immediate: true,
  }
);
</script>
