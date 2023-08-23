<template>
  <div class="space-y-2">
    <div
      class="text-lg font-medium leading-7 text-main flex items-center justify-between"
    >
      <div class="flex items-center">
        <EnvironmentTabFilter
          :environment="state.environment"
          :include-all="true"
          @update:environment="state.environment = $event"
        />
      </div>
      <NInputGroup style="width: auto">
        <InstanceSelect
          :instance="state.instance"
          :include-all="true"
          :filter="filterInstance"
          :environment="environment?.uid"
          @update:instance="
            state.instance = $event ? String($event) : String(UNKNOWN_ID)
          "
        />
        <SearchBox
          :value="state.keyword"
          :placeholder="$t('database.search-database')"
          @update:value="state.keyword = $event"
        />
      </NInputGroup>
    </div>

    <template v-if="databaseList.length > 0">
      <DatabaseV1Table
        mode="PROJECT"
        table-class="border"
        :database-list="filteredDatabaseList"
      />
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
import { DatabaseV1Table } from "./v2";
import { EnvironmentTabFilter, InstanceSelect, SearchBox } from "./v2";

interface LocalState {
  environment: string;
  instance: string;
  keyword: string;
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
</script>
