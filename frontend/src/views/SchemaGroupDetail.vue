<template>
  <div
    v-if="state.isLoaded"
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
                  <BBBadge text="Group" :can-remove="false" class="text-xs" />
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
            Configure
          </button>
          <button
            type="button"
            class="btn-normal"
            @click.prevent="createMigration('bb.issue.database.schema.update')"
          >
            Alter Schema
          </button>
          <button
            type="button"
            class="btn-normal"
            @click.prevent="createMigration('bb.issue.database.data.update')"
          >
            Change Data
          </button>
        </div>
      </div>

      <hr class="my-4" />

      <div class="w-full px-3 max-w-5xl grid grid-cols-5 gap-x-6">
        <div class="col-span-3">
          <p class="pl-1 text-lg mb-2">Condition</p>
          <ExprEditor
            :expr="state.expr!"
            :allow-admin="false"
            :resource-type="'SCHEMA_GROUP'"
          />
        </div>
        <div class="col-span-2">
          <MatchedTableView
            :project="project"
            :database-group-name="databaseGroupName"
            :schema-group="schemaGroup"
            :expr="state.expr!"
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
import { reactive, computed, watch } from "vue";
import { useDBGroupStore, useProjectV1Store } from "@/store";
import {
  databaseGroupNamePrefix,
  projectNamePrefix,
  schemaGroupNamePrefix,
} from "@/store/modules/v1/common";
import { convertCELStringToExpr } from "@/utils/databaseGroup/cel";
import { ConditionGroupExpr } from "@/plugins/cel";
import DatabaseGroupPanel from "@/components/DatabaseGroup/DatabaseGroupPanel.vue";
import ExprEditor from "@/components/DatabaseGroup/common/ExprEditor";
import MatchedTableView from "@/components/DatabaseGroup/MatchedTableView.vue";
import { generateIssueRoute } from "@/utils/databaseGroup/issue";
import { useRouter } from "vue-router";

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

const router = useRouter();
const projectStore = useProjectV1Store();
const dbGroupStore = useDBGroupStore();
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

const createMigration = (
  type: "bb.issue.database.schema.update" | "bb.issue.database.data.update"
) => {
  if (!schemaGroup.value) {
    return;
  }
  const issueRoute = generateIssueRoute(type, schemaGroup.value.databaseGroup, [
    schemaGroup.value.name,
  ]);
  router.push(issueRoute);
};

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
</script>
