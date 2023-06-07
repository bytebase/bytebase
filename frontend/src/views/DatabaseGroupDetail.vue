<template>
  <div
    v-if="state.isLoaded"
    class="flex-1 overflow-auto focus:outline-none"
    tabindex="0"
    v-bind="$attrs"
  >
    <main class="flex-1 relative overflow-y-auto">
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
                  {{ databaseGroupName }}
                  <BBBadge text="Group" :can-remove="false" class="text-xs" />
                </h1>
              </div>
            </div>
          </div>
          <dl
            class="flex flex-col space-y-1 md:space-y-0 md:flex-row md:flex-wrap"
          >
            <dt class="sr-only">{{ $t("common.environment") }}</dt>
            <dd class="flex items-center text-sm md:mr-4">
              <span class="textlabel"
                >{{ $t("common.environment") }}&nbsp;-&nbsp;</span
              >
              <EnvironmentV1Name
                :environment="environment"
                icon-class="textinfolabel"
              />
            </dd>
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
          <NButton @click="handleEditDatabaseGroup">{{
            $t("common.configure")
          }}</NButton>
          <NButton
            @click="createMigration('bb.issue.database.schema.update')"
            >{{ $t("database.alter-schema") }}</NButton
          >
          <NButton @click="createMigration('bb.issue.database.data.update')">{{
            $t("database.change-data")
          }}</NButton>
        </div>
      </div>

      <hr class="my-4" />

      <div class="w-full px-3 max-w-5xl grid grid-cols-5 gap-x-6">
        <div class="col-span-3">
          <p class="pl-1 text-lg mb-2">
            {{ $t("database-group.condition.self") }}
          </p>
          <ExprEditor
            :expr="state.expr!"
            :allow-admin="false"
            :resource-type="'DATABASE_GROUP'"
          />
        </div>
        <div class="col-span-2">
          <MatchedDatabaseView
            :project="project"
            :environment-id="environment.uid"
            :expr="state.expr!"
            :database-group="databaseGroup"
          />
        </div>
      </div>

      <hr class="mt-8 my-4" />
      <div class="w-full max-w-5xl px-4">
        <div class="w-full flex flex-row justify-between items-center">
          <p class="my-4">{{ $t("database-group.table-group.self") }}</p>
          <div>
            <NButton @click.prevent="handleCreateSchemaGroup">
              {{ $t("database-group.table-group.create") }}
            </NButton>
          </div>
        </div>
        <SchemaGroupTable
          :schema-group-list="schemaGroupList"
          @edit="handleEditSchemaGroup"
        />
      </div>
    </main>
  </div>

  <DatabaseGroupPanel
    v-if="editState.showConfigurePanel"
    :project="project"
    :resource-type="editState.type"
    :database-group="editState.databaseGroup"
    @close="editState.showConfigurePanel = false"
  />
</template>

<script lang="ts" setup>
import { onMounted, reactive, computed, watch } from "vue";
import { useDBGroupStore, useProjectV1Store } from "@/store";
import {
  databaseGroupNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { DatabaseGroup, SchemaGroup } from "@/types/proto/v1/project_service";
import { ConditionGroupExpr } from "@/plugins/cel";
import DatabaseGroupPanel from "@/components/DatabaseGroup/DatabaseGroupPanel.vue";
import ExprEditor from "@/components/DatabaseGroup/common/ExprEditor";
import MatchedDatabaseView from "@/components/DatabaseGroup/MatchedDatabaseView.vue";
import SchemaGroupTable from "@/components/DatabaseGroup/SchemaGroupTable.vue";
import { ResourceType } from "@/components/DatabaseGroup/common/ExprEditor/context";
import { useRouter } from "vue-router";
import { ComposedDatabaseGroup } from "@/types";
import { generateDatabaseGroupIssueRoute } from "@/utils/databaseGroup/issue";
import { NButton } from "naive-ui";

interface LocalState {
  isLoaded: boolean;
  expr?: ConditionGroupExpr;
}

interface EditDatabaseGroupState {
  showConfigurePanel: boolean;
  type: ResourceType;
  databaseGroup?: DatabaseGroup | SchemaGroup;
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
});

const router = useRouter();
const projectStore = useProjectV1Store();
const dbGroupStore = useDBGroupStore();
const state = reactive<LocalState>({
  isLoaded: false,
});
const editState = reactive<EditDatabaseGroupState>({
  showConfigurePanel: false,
  type: "DATABASE_GROUP",
});
const databaseGroupResourceName = computed(() => {
  return `${projectNamePrefix}${props.projectName}/${databaseGroupNamePrefix}${props.databaseGroupName}`;
});
const databaseGroup = computed(() => {
  return dbGroupStore.getDBGroupByName(
    databaseGroupResourceName.value
  ) as ComposedDatabaseGroup;
});
const schemaGroupList = computed(() => {
  return dbGroupStore.getSchemaGroupListByDBGroupName(
    databaseGroupResourceName.value
  );
});
const environment = computed(() => databaseGroup.value.environment);
const project = computed(() => {
  return projectStore.getProjectByName(
    `${projectNamePrefix}${props.projectName}`
  );
});

onMounted(async () => {
  await dbGroupStore.getOrFetchDBGroupByName(databaseGroupResourceName.value);
});

const handleEditDatabaseGroup = () => {
  editState.type = "DATABASE_GROUP";
  editState.databaseGroup = databaseGroup.value;
  editState.showConfigurePanel = true;
};

const handleCreateSchemaGroup = () => {
  editState.type = "SCHEMA_GROUP";
  editState.databaseGroup = undefined;
  editState.showConfigurePanel = true;
};

const handleEditSchemaGroup = (schemaGroup: SchemaGroup) => {
  editState.type = "SCHEMA_GROUP";
  editState.databaseGroup = schemaGroup;
  editState.showConfigurePanel = true;
};

const createMigration = (
  type: "bb.issue.database.schema.update" | "bb.issue.database.data.update"
) => {
  const issueRoute = generateDatabaseGroupIssueRoute(type, databaseGroup.value);
  router.push(issueRoute);
};

watch(
  () => [databaseGroup.value],
  async () => {
    if (!databaseGroup.value) {
      return;
    }

    state.expr = databaseGroup.value.simpleExpr;
    await dbGroupStore.getOrFetchSchemaGroupListByDBGroupName(
      databaseGroup.value.name
    );

    state.isLoaded = true;
  },
  {
    immediate: true,
  }
);
</script>
