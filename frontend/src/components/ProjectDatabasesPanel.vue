<template>
  <div class="space-y-2">
    <div
      class="text-lg font-medium leading-7 text-main flex flex-col lg:flex-row items-start lg:items-center justify-between space-y-2"
    >
      <EnvironmentTabFilter
        :environment="state.environment"
        :include-all="true"
        @update:environment="state.environment = $event"
      />
      <NInputGroup style="width: auto">
        <DatabaseLabelFilter
          v-model:selected="state.selectedLabels"
          :database-list="databaseList"
        />
        <InstanceSelect
          class="!w-48"
          :instance="state.instance"
          :include-all="true"
          :filter="filterInstance"
          :environment="environment?.uid"
          @update:instance="
            state.instance = $event ? String($event) : String(UNKNOWN_ID)
          "
        />
        <div class="hidden md:block">
          <SearchBox
            :value="state.keyword"
            :placeholder="$t('common.filter-by-name')"
            @update:value="state.keyword = $event"
          />
        </div>
      </NInputGroup>
    </div>

    <DatabaseOperations
      v-if="selectedDatabases.length > 0"
      :databases="selectedDatabases"
      @dismiss="state.selectedDatabaseIds.clear()"
    />

    <template v-if="databaseList.length > 0">
      <DatabaseV1Table
        mode="PROJECT"
        table-class="border"
        :show-selection-column="true"
        :database-list="filteredDatabaseList"
      >
        <template #selection-all="{ databaseList: selectedDatabaseList }">
          <input
            v-if="selectedDatabaseList.length > 0"
            type="checkbox"
            class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
            v-bind="getAllSelectionState(selectedDatabaseList as ComposedDatabase[])"
            @input="
              toggleDatabasesSelection(
                selectedDatabaseList as ComposedDatabase[],
                ($event.target as HTMLInputElement).checked
              )
            "
          />
        </template>
        <template #selection="{ database }">
          <input
            type="checkbox"
            class="h-4 w-4 text-accent rounded disabled:cursor-not-allowed border-control-border focus:ring-accent"
            :checked="isDatabaseSelected(database as ComposedDatabase)"
            @click.stop="
              toggleDatabasesSelection(
                [database as ComposedDatabase],
                ($event.target as HTMLInputElement).checked
              )
            "
          />
        </template>
      </DatabaseV1Table>
    </template>
    <div v-else class="text-center textinfolabel">
      <i18n-t keypath="project.overview.no-db-prompt" tag="p">
        <template #newDb>
          <span class="text-main">{{ $t("quick-action.new-db") }}</span>
        </template>
        <template #transferInDb>
          <span class="text-main">{{ $t("quick-action.transfer-in-db") }}</span>
        </template>
      </i18n-t>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { uniqBy } from "lodash-es";
import { NInputGroup } from "naive-ui";
import { reactive, PropType, computed, ref, watchEffect } from "vue";
import {
  useCurrentUserV1,
  usePolicyV1Store,
  useEnvironmentV1Store,
} from "@/store";
import {
  Policy,
  PolicyResourceType,
  PolicyType,
} from "@/types/proto/v1/org_policy_service";
import { filterDatabaseV1ByKeyword, isDatabaseV1Accessible } from "@/utils";
import {
  ComposedDatabase,
  ComposedInstance,
  UNKNOWN_ID,
  UNKNOWN_ENVIRONMENT_NAME,
} from "../types";
import {
  DatabaseV1Table,
  DatabaseOperations,
  DatabaseLabelFilter,
  EnvironmentTabFilter,
  InstanceSelect,
  SearchBox,
} from "./v2";

interface LocalState {
  environment: string;
  instance: string;
  keyword: string;
  selectedDatabaseIds: Set<string>;
  selectedLabels: { key: string; value: string }[];
}

const props = defineProps({
  databaseList: {
    required: true,
    type: Object as PropType<ComposedDatabase[]>,
  },
});

const currentUserV1 = useCurrentUserV1();
const environmentV1Store = useEnvironmentV1Store();

const state = reactive<LocalState>({
  environment: UNKNOWN_ENVIRONMENT_NAME,
  instance: String(UNKNOWN_ID),
  keyword: "",
  selectedDatabaseIds: new Set(),
  selectedLabels: [],
});
const policyList = ref<Policy[]>([]);

const preparePolicyList = () => {
  usePolicyV1Store()
    .fetchPolicies({
      policyType: PolicyType.WORKSPACE_IAM,
      resourceType: PolicyResourceType.WORKSPACE,
    })
    .then((list) => (policyList.value = list));
};

watchEffect(preparePolicyList);

const filteredDatabaseList = computed(() => {
  let list = [...props.databaseList].filter((database) =>
    isDatabaseV1Accessible(database, currentUserV1.value)
  );
  if (state.environment !== UNKNOWN_ENVIRONMENT_NAME) {
    list = list.filter(
      (db) => db.effectiveEnvironmentEntity.name === state.environment
    );
  }
  if (state.instance !== String(UNKNOWN_ID)) {
    list = list.filter((db) => db.instanceEntity.uid === state.instance);
  }
  const keyword = state.keyword.trim().toLowerCase();
  if (keyword) {
    list = list.filter((db) =>
      filterDatabaseV1ByKeyword(db, keyword, [
        "name",
        "environment",
        "instance",
      ])
    );
  }
  const labels = state.selectedLabels;
  if (labels.length > 0) {
    list = list.filter((db) => {
      return labels.some((kv) => db.labels[kv.key] === kv.value);
    });
  }
  return list;
});

const instanceList = computed(() => {
  return uniqBy(
    props.databaseList.map((db) => db.instanceEntity),
    (instance) => instance.uid
  );
});

const filterInstance = (instance: ComposedInstance) => {
  return instanceList.value.findIndex((inst) => inst.uid === instance.uid) >= 0;
};

const environment = computed(() => {
  return environmentV1Store.getEnvironmentByName(state.environment);
});

const getAllSelectionState = (
  databaseList: ComposedDatabase[]
): { checked: boolean; indeterminate: boolean } => {
  const checked =
    state.selectedDatabaseIds.size > 0 &&
    databaseList.every((db) => state.selectedDatabaseIds.has(db.uid));
  const indeterminate =
    !checked &&
    databaseList.some((db) => state.selectedDatabaseIds.has(db.uid));

  return {
    checked,
    indeterminate,
  };
};

const toggleDatabasesSelection = (
  databaseList: ComposedDatabase[],
  on: boolean
): void => {
  if (on) {
    databaseList.forEach((db) => {
      state.selectedDatabaseIds.add(db.uid);
    });
  } else {
    databaseList.forEach((db) => {
      state.selectedDatabaseIds.delete(db.uid);
    });
  }
};

const isDatabaseSelected = (database: ComposedDatabase): boolean => {
  return state.selectedDatabaseIds.has((database as ComposedDatabase).uid);
};

const selectedDatabases = computed((): ComposedDatabase[] => {
  return filteredDatabaseList.value.filter((db) =>
    state.selectedDatabaseIds.has(db.uid)
  );
});
</script>
