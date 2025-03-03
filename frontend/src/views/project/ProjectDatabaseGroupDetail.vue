<template>
  <div
    v-if="databaseGroup"
    class="flex-1 overflow-auto focus:outline-none"
    tabindex="0"
    v-bind="$attrs"
  >
    <main class="flex-1 relative overflow-y-auto space-y-4">
      <div
        class="space-y-2 lg:space-y-0 lg:flex lg:items-center lg:justify-between"
      >
        <div class="flex-1 min-w-0 shrink-0">
          <!-- Summary -->
          <div class="flex items-center">
            <div>
              <div class="flex items-center gap-2">
                <h1
                  class="pt-2 pb-2.5 text-xl font-bold leading-6 text-main truncate flex items-center gap-x-3"
                >
                  {{ databaseGroup.databasePlaceholder }}
                </h1>
                <NTag v-if="databaseGroup.multitenancy" round type="info">
                  <template #icon>
                    <TenantIcon />
                  </template>
                  {{ $t("database-group.multitenancy.self") }}
                </NTag>
              </div>
            </div>
          </div>
        </div>

        <div
          class="flex flex-row justify-end items-center flex-wrap shrink gap-x-2 gap-y-2"
        >
          <NButton v-if="allowEdit" @click="handleEditDatabaseGroup">
            {{ $t("common.configure") }}
          </NButton>
          <NButton
            v-if="hasPermissionToCreateIssue"
            @click="
              previewDatabaseGroupIssue('bb.issue.database.schema.update')
            "
          >
            {{ $t("database.edit-schema") }}
          </NButton>
          <NButton
            v-if="hasPermissionToCreateIssue"
            @click="previewDatabaseGroupIssue('bb.issue.database.data.update')"
          >
            {{ $t("database.change-data") }}
          </NButton>
        </div>
      </div>

      <NDivider />

      <FeatureAttentionForInstanceLicense
        v-if="existMatchedUnactivateInstance"
        type="warning"
        feature="bb.feature.database-grouping"
      />

      <div class="w-full max-w-5xl grid grid-cols-5 gap-x-6">
        <div class="col-span-3">
          <p class="pl-1 text-lg mb-2">
            {{ $t("database-group.condition.self") }}
          </p>
          <ExprEditor
            :expr="databaseGroup.simpleExpr"
            :allow-admin="false"
            :enable-raw-expression="true"
            :factor-list="FactorList"
            :factor-support-dropdown="factorSupportDropdown"
            :option-config-map="getDatabaseGroupOptionConfigMap(project)"
          />
        </div>
        <div class="col-span-2">
          <MatchedDatabaseView
            :loading="false"
            :matched-database-list="matchedDatabaseList"
            :unmatched-database-list="unmatchedDatabaseList"
          />
        </div>
      </div>
    </main>
  </div>

  <DatabaseGroupPanel
    :show="state.showEditPanel"
    :project="project"
    :database-group="databaseGroup"
    @close="state.showEditPanel = false"
  />
</template>

<script lang="ts" setup>
import { NButton, NDivider, NTag } from "naive-ui";
import { reactive, computed, watchEffect } from "vue";
import { useRouter } from "vue-router";
import DatabaseGroupPanel from "@/components/DatabaseGroup/DatabaseGroupPanel.vue";
import MatchedDatabaseView from "@/components/DatabaseGroup/MatchedDatabaseView.vue";
import { FactorList } from "@/components/DatabaseGroup/utils";
import {
  factorSupportDropdown,
  getDatabaseGroupOptionConfigMap,
} from "@/components/DatabaseGroup/utils";
import ExprEditor from "@/components/ExprEditor";
import { FeatureAttentionForInstanceLicense } from "@/components/FeatureGuard";
import TenantIcon from "@/components/TenantIcon.vue";
import {
  useDBGroupStore,
  useDatabaseV1Store,
  useProjectByName,
  useSubscriptionV1Store,
} from "@/store";
import { databaseGroupNamePrefix } from "@/store/modules/v1/common";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { DatabaseGroupView } from "@/types/proto/v1/database_group_service";
import { hasPermissionToCreateChangeDatabaseIssueInProject } from "@/utils";
import { generateDatabaseGroupIssueRoute } from "@/utils/databaseGroup/issue";

interface LocalState {
  showEditPanel: boolean;
}

const props = defineProps<{
  projectId: string;
  databaseGroupName: string;
  allowEdit: boolean;
}>();

const router = useRouter();
const dbGroupStore = useDBGroupStore();
const databaseStore = useDatabaseV1Store();
const subscriptionV1Store = useSubscriptionV1Store();
const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);
const state = reactive<LocalState>({
  showEditPanel: false,
});

const databaseGroupResourceName = computed(() => {
  return `${project.value.name}/${databaseGroupNamePrefix}${props.databaseGroupName}`;
});

const databaseGroup = computed(() => {
  return dbGroupStore.getDBGroupByName(databaseGroupResourceName.value);
});

const hasPermissionToCreateIssue = computed(() => {
  return hasPermissionToCreateChangeDatabaseIssueInProject(project.value);
});

watchEffect(async () => {
  await dbGroupStore.getOrFetchDBGroupByName(databaseGroupResourceName.value, {
    skipCache: true,
    view: DatabaseGroupView.DATABASE_GROUP_VIEW_FULL,
  });
});

const matchedDatabaseList = computed(
  () =>
    databaseGroup.value?.matchedDatabases.map((db) =>
      databaseStore.getDatabaseByName(db.name)
    ) || []
);

const unmatchedDatabaseList = computed(
  () =>
    databaseGroup.value?.unmatchedDatabases.map((db) =>
      databaseStore.getDatabaseByName(db.name)
    ) || []
);

const existMatchedUnactivateInstance = computed(() => {
  return matchedDatabaseList.value?.some(
    (database) =>
      !subscriptionV1Store.hasInstanceFeature(
        "bb.feature.database-grouping",
        database.instanceResource
      )
  );
});

const handleEditDatabaseGroup = () => {
  state.showEditPanel = true;
};

const previewDatabaseGroupIssue = (
  type: "bb.issue.database.schema.update" | "bb.issue.database.data.update"
) => {
  if (!databaseGroup.value) {
    return;
  }
  const issueRoute = generateDatabaseGroupIssueRoute(type, databaseGroup.value);
  router.push(issueRoute);
};
</script>
