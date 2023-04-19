<template>
  <BBGrid
    :column-list="INSTANCE_COLUMNS"
    :data-source="composedSlowQueryPolicyList"
    :row-clickable="false"
    :is-row-expanded="(item: ComposedSlowQueryPolicy) => item.databaseList.length > 0"
    row-key="id"
    class="border"
    expanded-row-class="!p-0"
  >
    <template #item="{ item }: ComposedSlowQueryPolicyRow">
      <div class="bb-grid-cell">
        <InstanceName :instance="item.instance" />
      </div>

      <div class="bb-grid-cell">
        <EnvironmentName :environment="item.instance.environment" />
      </div>
      <div class="bb-grid-cell">
        <SpinnerSwitch
          v-if="item.type === 'INSTANCE'"
          :value="item.active"
          :disabled="!allowAdmin"
          :on-toggle="(active) => toggleInstanceActive(item.instance, active)"
        />
      </div>
    </template>

    <template
      #expanded-item="{
        item: { databaseList, instance },
      }: ComposedSlowQueryPolicyRow"
    >
      <BBGrid
        :column-list="DATABASE_COLUMNS"
        :data-source="databaseList"
        :show-header="false"
        class="w-full ml-9 border-l"
      >
        <template
          #item="{
            item: { database, active },
          }: ComposedDatabaseSlowQueryPolicyRow"
        >
          <div class="bb-grid-cell">
            <DatabaseName :database="database" />
          </div>
          <div class="bb-grid-cell">
            <SpinnerSwitch
              :value="active"
              :disabled="!allowAdmin"
              :on-toggle="
                (active) => toggleDatabaseActive(instance, database, active)
              "
            />
          </div>
        </template>
      </BBGrid>
    </template>
  </BBGrid>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import { type BBGridColumn, BBGrid, BBGridRow } from "@/bbkit";
import type { Database, Instance, Policy } from "@/types";
import {
  InstanceName,
  EnvironmentName,
  SpinnerSwitch,
  DatabaseName,
} from "@/components/v2/";
import { useCurrentUser, useDatabaseStore } from "@/store";
import {
  hasWorkspacePermission,
  slowQueryTypeOfInstance,
  extractSlowQueryPolicyPayload,
} from "@/utils";

export type ComposedDatabaseSlowQueryPolicy = {
  database: Database;
  active: boolean;
};
export type ComposedSlowQueryPolicy = {
  instance: Instance;
  type: "INSTANCE" | "DATABASE";
  active: boolean;
  databaseList: ComposedDatabaseSlowQueryPolicy[];
};

export type ComposedSlowQueryPolicyRow = BBGridRow<ComposedSlowQueryPolicy>;
export type ComposedDatabaseSlowQueryPolicyRow =
  BBGridRow<ComposedDatabaseSlowQueryPolicy>;

const props = defineProps<{
  instanceList: Instance[];
  policyList: Policy[];
  toggleInstanceActive: (instance: Instance, active: boolean) => Promise<void>;
  toggleDatabaseActive: (
    instance: Instance,
    database: Database,
    active: boolean
  ) => Promise<void>;
}>();

const { t } = useI18n();
const currentUser = useCurrentUser();
const databaseStore = useDatabaseStore();

const INSTANCE_COLUMNS = computed((): BBGridColumn[] => {
  return [
    {
      title: t("common.instance"),
      width: "2fr",
    },
    {
      title: t("common.environment"),
      width: "minmax(auto, 1fr)",
    },
    {
      title: t("common.active"),
      width: "minmax(auto, 6rem)",
    },
  ];
});

const DATABASE_COLUMNS = computed((): BBGridColumn[] => {
  return [
    {
      title: t("common.database"),
      width: "1fr",
    },
    {
      title: t("common.active"),
      width: "minmax(auto, 6rem)",
    },
  ];
});

const allowAdmin = computed(() => {
  return hasWorkspacePermission(
    "bb.permission.workspace.manage-slow-query",
    currentUser.value.role
  );
});

const composeDatabaseList = (
  instance: Instance,
  policy: Policy | undefined
) => {
  const databaseList = databaseStore.getDatabaseListByInstanceId(instance.id);
  const payload = extractSlowQueryPolicyPayload(policy);
  return databaseList.map<ComposedDatabaseSlowQueryPolicy>((database) => {
    const active =
      payload.active && payload.databaseList?.includes(database.name)
        ? true
        : false;
    return {
      database,
      active,
    };
  });
};

const composedSlowQueryPolicyList = computed(() => {
  return props.instanceList.map<ComposedSlowQueryPolicy>((instance) => {
    const policy = props.policyList.find(
      (policy) => policy.resourceId === instance.id
    );
    const type = slowQueryTypeOfInstance(instance)!;
    const active = extractSlowQueryPolicyPayload(policy).active;
    const databaseList =
      type === "DATABASE" ? composeDatabaseList(instance, policy) : [];
    return {
      instance,
      type,
      active,
      databaseList,
    };
  });
});
</script>
