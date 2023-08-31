<template>
  <div
    v-if="state.isLoaded && schemaGroup"
    class="flex-1 overflow-auto focus:outline-none"
    tabindex="0"
    v-bind="$attrs"
  >
    <main class="flex-1 relative overflow-y-auto">
      <!-- Highlight Panel -->
      <div
        class="px-4 space-y-2 lg:space-y-0 lg:flex lg:items-center lg:justify-between"
      >
        <div class="flex-1 min-w-0 shrink-0">
          <!-- Summary -->
          <div class="flex items-center">
            <div>
              <div class="flex items-center">
                <h1
                  class="pt-2 pb-2.5 text-xl font-bold leading-6 text-main truncate flex items-center gap-x-3"
                >
                  {{ schemaGroupName }}
                  <BBBadge
                    text="Table Group"
                    :can-remove="false"
                    class="text-xs"
                  />
                </h1>
              </div>
            </div>
          </div>
          <dl
            class="flex flex-col space-y-1 md:space-y-0 md:flex-row md:flex-wrap"
          >
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("common.project") }}&nbsp;-&nbsp;</span
              >
              <ProjectV1Name :project="project" hash="#database-groups" />
            </dd>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("database-group.self") }}&nbsp;-&nbsp;</span
              >
              <DatabaseGroupName :database-group="schemaGroup.databaseGroup" />
            </dd>
          </dl>
        </div>

        <div
          class="flex flex-row justify-end items-center flex-wrap shrink gap-x-2 gap-y-2"
        >
          <button
            type="button"
            class="btn-normal"
            @click.prevent="state.showConfigurePanel = true"
          >
            {{ $t("common.configure") }}
          </button>
        </div>
      </div>

      <hr class="my-4" />

      <FeatureAttentionForInstanceLicense
        v-if="existMatchedUnactivateInstance"
        custom-class="m-4"
        :style="`WARN`"
        feature="bb.feature.database-grouping"
      />

      <div class="w-full px-3 max-w-5xl grid grid-cols-5 gap-x-6">
        <div class="col-span-3">
          <p class="pl-1 text-lg mb-2">
            {{ $t("database-group.condition.self") }}
          </p>
          <ExprEditor
            :expr="state.expr!"
            :allow-admin="false"
            :resource-type="'SCHEMA_GROUP'"
          />
        </div>
        <div class="col-span-2">
          <MatchedTableView
            :loading="false"
            :matched-table-list="matchedTableList"
            :unmatched-table-list="unmatchedTableList"
          />
        </div>
      </div>
    </main>
  </div>

  <DatabaseGroupPanel
    v-if="state.showConfigurePanel"
    :project="project"
    :resource-type="'SCHEMA_GROUP'"
    :database-group="schemaGroup"
    @close="state.showConfigurePanel = false"
  />
</template>

<script lang="ts" setup>
import { useDebounceFn } from "@vueuse/core";
import { reactive, computed, watch, ref } from "vue";
import DatabaseGroupPanel from "@/components/DatabaseGroup/DatabaseGroupPanel.vue";
import MatchedTableView from "@/components/DatabaseGroup/MatchedTableView.vue";
import ExprEditor from "@/components/DatabaseGroup/common/ExprEditor";
import DatabaseGroupName from "@/components/v2/Model/DatabaseGroupName.vue";
import { ConditionGroupExpr } from "@/plugins/cel";
import {
  useDBGroupStore,
  useProjectV1Store,
  useSubscriptionV1Store,
} from "@/store";
import {
  databaseGroupNamePrefix,
  projectNamePrefix,
  schemaGroupNamePrefix,
} from "@/store/modules/v1/common";
import { ComposedSchemaGroupTable } from "@/types";
import { convertCELStringToExpr } from "@/utils/databaseGroup/cel";

interface LocalState {
  isLoaded: boolean;
  showConfigurePanel: boolean;
  expr?: ConditionGroupExpr;
}

const props = defineProps({
  projectName: {
    required: true,
    type: String,
  },
  databaseGroupName: {
    required: true,
    type: String,
  },
  schemaGroupName: {
    required: true,
    type: String,
  },
});

const projectStore = useProjectV1Store();
const dbGroupStore = useDBGroupStore();
const subscriptionV1Store = useSubscriptionV1Store();

const state = reactive<LocalState>({
  isLoaded: false,
  showConfigurePanel: false,
});
const project = computed(() => {
  return projectStore.getProjectByName(
    `${projectNamePrefix}${props.projectName}`
  );
});
const schemaGroup = computed(() => {
  return dbGroupStore.getSchemaGroupByName(
    `${projectNamePrefix}${props.projectName}/${databaseGroupNamePrefix}${props.databaseGroupName}/${schemaGroupNamePrefix}${props.schemaGroupName}`
  );
});

watch(
  () => [props, schemaGroup.value],
  async () => {
    const schemaGroup = await dbGroupStore.getOrFetchSchemaGroupByName(
      `${projectNamePrefix}${props.projectName}/${databaseGroupNamePrefix}${props.databaseGroupName}/${schemaGroupNamePrefix}${props.schemaGroupName}`
    );

    const expression = schemaGroup.tableExpr?.expression ?? "";
    const convertResult = await convertCELStringToExpr(expression);
    state.expr = convertResult;
    state.isLoaded = true;
  },
  {
    immediate: true,
  }
);

const matchedTableList = ref<ComposedSchemaGroupTable[]>([]);
const unmatchedTableList = ref<ComposedSchemaGroupTable[]>([]);
const updateTableMatchingState = useDebounceFn(async () => {
  if (!project.value) {
    return;
  }
  if (!state.expr) {
    return;
  }

  const result = await dbGroupStore.fetchSchemaGroupMatchList({
    projectName: project.value.name,
    databaseGroupName: props.databaseGroupName,
    expr: state.expr,
  });

  matchedTableList.value = result.matchedTableList;
  unmatchedTableList.value = result.unmatchedTableList;
}, 500);

watch([() => project.value, () => state.expr], updateTableMatchingState, {
  immediate: true,
  deep: true,
});

const existMatchedUnactivateInstance = computed(() => {
  return matchedTableList.value.some(
    (tb) =>
      !subscriptionV1Store.hasInstanceFeature(
        "bb.feature.database-grouping",
        tb.databaseEntity.instanceEntity
      )
  );
});
</script>
