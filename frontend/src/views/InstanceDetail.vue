<template>
  <div class="flex flex-col gap-y-2 px-6" v-bind="$attrs">
    <ArchiveBanner v-if="instance.state === State.DELETED" />
    <BBAttention
      v-if="!instance.environment"
      class="w-full mb-4"
      :type="'warning'"
    >
      {{ $t("instance.no-environment") }}
    </BBAttention>

    <div class="flex items-center justify-between">
      <div class="flex items-center gap-x-2">
        <EngineIcon :engine="instance.engine" custom-class="h-6!" />
        <span class="text-lg font-medium">{{ instanceV1Name(instance) }}</span>
      </div>
    </div>

    <NTabs :value="state.selectedTab" @update:value="onTabChange">
      <template #suffix>
        <div v-if="instance.state === State.ACTIVE" class="flex items-center gap-x-2">
          <InstanceSyncButton
            @sync-schema="syncSchema"
          />
          <PermissionGuardWrapper
            v-if="allowCreateDatabase"
            v-slot="slotProps"
            :permissions="['bb.issues.create', 'bb.plans.create']"
          >
            <NButton
              type="primary"
              :disabled="slotProps.disabled"
              @click.prevent="createDatabase"
            >
              <template #icon>
                <PlusIcon class="h-4 w-4" />
              </template>
              {{ $t("instance.new-database") }}
            </NButton>
          </PermissionGuardWrapper>
        </div>
      </template>
      <NTabPane name="overview" :tab="$t('common.overview')">
        <InstanceForm ref="instanceFormRef" class="-mt-2" :instance="instance">
          <InstanceFormBody :hide-archive-restore="hideArchiveRestore" />
          <InstanceFormButtons class="sticky bottom-0 z-10" />
        </InstanceForm>
      </NTabPane>
      <NTabPane name="databases" :tab="$t('common.databases')">
        <div class="flex flex-col gap-y-2">
          <div
            class="w-full flex flex-col sm:flex-row items-start sm:items-end justify-between gap-2"
          >
            <AdvancedSearch
              v-model:params="state.params"
              class="flex-1"
              :autofocus="false"
              :placeholder="$t('database.filter-database')"
              :scope-options="scopeOptions"
            />
          </div>
          <DatabaseOperations
            :databases="selectedDatabases"
            @refresh="() => pagedDatabaseTableRef?.refresh()"
            @update="
              (databases) => pagedDatabaseTableRef?.updateCache(databases)
            "
          />
          <PagedDatabaseTable
            ref="pagedDatabaseTableRef"
            mode="INSTANCE"
            :footer-class="'pb-4'"
            :show-selection="true"
            :filter="filter"
            :parent="instance.name"
            v-model:selected-database-names="state.selectedDatabaseNameList"
          />
        </div>
      </NTabPane>
      <NTabPane name="users" :tab="$t('instance.users')">
        <InstanceRoleTable :instance-role-list="instanceRoleList" />
      </NTabPane>
    </NTabs>
  </div>

  <Drawer
    v-model:show="state.showCreateDatabaseModal"
    :title="$t('quick-action.create-db')"
  >
    <CreateDatabasePrepPanel
      :environment-name="formatEnvironmentName(environment.id)"
      :instance-name="instance.name"
      @dismiss="state.showCreateDatabaseModal = false"
    />
  </Drawer>
</template>

<script lang="tsx" setup>
import { useTitle } from "@vueuse/core";
import { cloneDeep } from "lodash-es";
import { PlusIcon } from "lucide-vue-next";
import { NButton, NTabPane, NTabs } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute, useRouter } from "vue-router";
import { BBAttention } from "@/bbkit";
import AdvancedSearch from "@/components/AdvancedSearch";
import { useCommonSearchScopeOptions } from "@/components/AdvancedSearch/useCommonSearchScopeOptions";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import { CreateDatabasePrepPanel } from "@/components/CreateDatabasePrepForm";
import { EngineIcon } from "@/components/Icon";
import InstanceSyncButton from "@/components/Instance/InstanceSyncButton.vue";
import {
  InstanceForm,
  Form as InstanceFormBody,
  Buttons as InstanceFormButtons,
} from "@/components/InstanceForm/";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { Drawer, InstanceRoleTable } from "@/components/v2";
import {
  DatabaseOperations,
  PagedDatabaseTable,
} from "@/components/v2/Model/DatabaseV1Table";
import { useRouteChangeGuard } from "@/composables/useRouteChangeGuard";
import { useBodyLayoutContext } from "@/layouts/common";
import {
  pushNotification,
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useInstanceV1Store,
} from "@/store";
import {
  environmentNamePrefix,
  instanceNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import {
  type ComposedDatabase,
  formatEnvironmentName,
  isValidDatabaseName,
} from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { SearchParams, SearchScope } from "@/utils";
import {
  CommonFilterScopeIdList,
  getValueFromSearchParams,
  getValuesFromSearchParams,
  instanceV1HasCreateDatabase,
  instanceV1Name,
} from "@/utils";

const instanceHashList = ["overview", "databases", "users"] as const;
export type InstanceHash = (typeof instanceHashList)[number];
const isInstanceHash = (x: unknown): x is InstanceHash =>
  instanceHashList.includes(x as InstanceHash);

interface LocalState {
  showCreateDatabaseModal: boolean;
  syncingSchema: boolean;
  selectedDatabaseNameList: string[];
  params: SearchParams;
  selectedTab: InstanceHash;
}

const props = defineProps<{
  instanceId: string;
  hideArchiveRestore?: boolean;
}>();

defineOptions({
  inheritAttrs: false,
});

const { overrideMainContainerClass } = useBodyLayoutContext();
overrideMainContainerClass("pb-0!");

const { t } = useI18n();
const router = useRouter();
const instanceV1Store = useInstanceV1Store();
const databaseStore = useDatabaseV1Store();
const pagedDatabaseTableRef = ref<InstanceType<typeof PagedDatabaseTable>>();
const instanceFormRef = ref<InstanceType<typeof InstanceForm>>();

const onTabChange = (tab: InstanceHash) => {
  if (instanceFormRef.value?.isEditing) {
    if (!window.confirm(t("common.leave-without-saving"))) {
      return;
    }
  }
  state.selectedTab = tab;
};

useRouteChangeGuard(computed(() => instanceFormRef.value?.isEditing ?? false));

const readonlyScopes = computed((): SearchScope[] => [
  { id: "instance", value: props.instanceId, readonly: true },
]);

const state = reactive<LocalState>({
  showCreateDatabaseModal: false,
  syncingSchema: false,
  selectedDatabaseNameList: [],
  params: {
    query: "",
    scopes: [...readonlyScopes.value],
  },
  selectedTab: "overview",
});

const route = useRoute();

const scopeOptions = useCommonSearchScopeOptions([
  ...CommonFilterScopeIdList,
  "project",
  "label",
]);

watch(
  () => route.hash,
  (hash) => {
    const targetHash = hash.replace(/^#?/g, "") as InstanceHash;
    if (isInstanceHash(targetHash)) {
      state.selectedTab = targetHash;
    }
  },
  { immediate: true }
);

watch(
  () => state.selectedTab,
  (tab) => {
    const query = cloneDeep(route.query);
    delete query["qs"];
    router.replace({
      query,
      hash: `#${tab}`,
    });
  },
  { immediate: true }
);

const instance = computed(() => {
  return instanceV1Store.getInstanceByName(
    `${instanceNamePrefix}${props.instanceId}`
  );
});

const environment = computed(() => {
  return useEnvironmentV1Store().getEnvironmentByName(
    instance.value.environment ?? ""
  );
});

const selectedEnvironment = computed(() => {
  return getValueFromSearchParams(
    state.params,
    "environment",
    environmentNamePrefix
  );
});

const selectedProject = computed(() => {
  return getValueFromSearchParams(state.params, "project", projectNamePrefix);
});

const selectedLabels = computed(() => {
  return getValuesFromSearchParams(state.params, "label");
});

const filter = computed(() => ({
  environment: selectedEnvironment.value,
  project: selectedProject.value,
  query: state.params.query,
  labels: selectedLabels.value,
}));

const instanceRoleList = computed(() => {
  return instance.value.roles;
});

const allowCreateDatabase = computed(() => {
  return instanceV1HasCreateDatabase(instance.value);
});

const syncSchema = async (enableFullSync: boolean) => {
  await instanceV1Store.syncInstance(instance.value.name, enableFullSync);
  // Remove the database list cache for the instance.
  databaseStore.removeCacheByInstance(instance.value.name);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t(
      "instance.successfully-synced-schema-for-instance-instance-value-name",
      [instance.value.title]
    ),
  });
};

const createDatabase = () => {
  state.showCreateDatabaseModal = true;
};

useTitle(computed(() => instance.value.title));

const selectedDatabases = computed((): ComposedDatabase[] => {
  return state.selectedDatabaseNameList
    .map((databaseName) => databaseStore.getDatabaseByName(databaseName))
    .filter((database) => isValidDatabaseName(database.name));
});
</script>
