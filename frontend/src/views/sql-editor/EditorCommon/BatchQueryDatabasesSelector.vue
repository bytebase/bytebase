<template>
  <NPopover placement="bottom" trigger="click">
    <template #trigger>
      <div
        class="!ml-2 w-6 h-6 p-1 cursor-pointer rounded hover:opacity-80"
        :class="
          selectedLabelsValue.length > 0
            ? 'text-blue-600 bg-blue-100 shadow'
            : 'text-gray-600'
        "
      >
        <Layers class="w-4 h-auto" />
      </div>
    </template>
    <div class="w-128">
      <p class="text-gray-500 mb-1">
        {{ $t("sql-editor.batch-query.description") }}
      </p>
      <div class="w-full grid grid-cols-3 gap-2 py-1">
        <div class="col-span-1">
          <NCheckboxGroup v-model:value="selectedLabelsValue">
            <NCheckbox
              v-for="label in labels"
              :key="`${label.key}-${label.value}`"
              :value="`${label.key}-${label.value}`"
              :label="`${getFormattedLabelKey(label.key)}:${label.value}`"
            >
            </NCheckbox>
          </NCheckboxGroup>
        </div>
        <div class="col-span-2">
          <MatchedDatabaseView
            :hide-title="true"
            :matched-database-list="matchedDatabases"
            :unmatched-database-list="unmatchedDatabases"
          />
        </div>
      </div>
    </div>
  </NPopover>
</template>

<script lang="ts" setup>
import { upperFirst } from "lodash-es";
import { Layers } from "lucide-vue-next";
import { NCheckboxGroup, NCheckbox, NPopover } from "naive-ui";
import { computed, ref, watch } from "vue";
import MatchedDatabaseView from "@/components/DatabaseGroup/MatchedDatabaseView.vue";
import {
  useCurrentUserIamPolicy,
  useDatabaseV1ByUID,
  useDatabaseV1Store,
  useTabStore,
} from "@/store/modules";

const databaseStore = useDatabaseV1Store();
const tabStore = useTabStore();
const currentUserIamPolicy = useCurrentUserIamPolicy();
// Save the stringified label key-value pairs.
const currentTab = computed(() => tabStore.currentTab);
const connection = computed(() => currentTab.value.connection);
const selectedLabelsValue = ref<string[]>([]);

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
      // Only show databases that have at least one label.
      .filter((db) => getFilteredDatabaseLabels(db.labels).length > 0)
  );
});

const labels = computed(() => {
  return filteredDatabaseList.value
    .map((db) => {
      return getFilteredDatabaseLabels(db.labels);
    })
    .flat();
});

const getFilteredDatabaseLabels = (labels: { [key: string]: string }) => {
  // Filter out the environment label.
  const keys = Object.keys(labels).filter((key) => {
    return key !== "bb.environment";
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
        return key !== "bb.environment";
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
  if (labelKey.startsWith("bb.")) {
    return upperFirst(labelKey.substring(3));
  }
  return upperFirst(labelKey);
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
