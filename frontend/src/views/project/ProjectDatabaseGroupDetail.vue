<template>
  <div
    v-if="databaseGroup"
    v-bind="$attrs"
    class="min-h-full flex-1 relative flex flex-col gap-y-4 pt-4"
  >
    <FeatureAttention
      :feature="PlanFeature.FEATURE_DATABASE_GROUPS"
      class="mb-4"
    />

    <div
      v-if="hasDatabaseGroupFeature && !state.editing"
      class="flex flex-row justify-end items-center flex-wrap shrink gap-x-2 gap-y-2"
    >
      <PermissionGuardWrapper
        v-slot="slotProps"
        :project="project"
        :permissions="[
          ...PERMISSIONS_FOR_DATABASE_CHANGE_ISSUE
        ]"
      >
        <NTooltip
          :disabled="slotProps.disabled || hasMatchedDatabases"
        >
          <template #trigger>
            <NButton
              :disabled="slotProps.disabled || !hasMatchedDatabases"
              @click="() => {
                preCreateIssue(project.name, [databaseGroupResourceName])
              }"
            >
              {{ $t("database.change-database") }}
            </NButton>
          </template>
          {{ $t("database-group.no-matched-databases") }}
        </NTooltip>
      </PermissionGuardWrapper>

      <PermissionGuardWrapper
        v-slot="slotProps"
        :project="project"
        :permissions="[
          'bb.databaseGroups.update',
        ]"
      >
        <NButton type="primary" :disabled="slotProps.disabled" @click="state.editing = true">
          <template #icon>
            <EditIcon class="w-4 h-4" />
          </template>
          {{ $t("common.configure") }}
        </NButton>
      </PermissionGuardWrapper>

      <NDropdown
        v-if="dropdownOptions.length > 0"
        trigger="click"
        :options="dropdownOptions"
        @select="handleDropdownSelect"
      >
        <NButton size="small" quaternary class="px-1!">
          <template #icon>
            <EllipsisVerticalIcon class="w-4 h-4" />
          </template>
        </NButton>
      </NDropdown>
    </div>

    <DatabaseGroupForm
      :readonly="!state.editing"
      :project="project"
      :database-group="databaseGroup"
      @dismiss="state.editing = false"
    />
  </div>
</template>

<script lang="ts" setup>
import { EditIcon, EllipsisVerticalIcon } from "lucide-vue-next";
import type { DropdownOption } from "naive-ui";
import { NButton, NDropdown, NTooltip, useDialog } from "naive-ui";
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import DatabaseGroupForm from "@/components/DatabaseGroup/DatabaseGroupForm.vue";
import FeatureAttention from "@/components/FeatureGuard/FeatureAttention.vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { preCreateIssue } from "@/components/Plan/logic/issue";
import { useBodyLayoutContext } from "@/layouts/common";
import { PROJECT_V1_ROUTE_DATABASE_GROUPS } from "@/router/dashboard/projectV1";
import {
  featureToRef,
  useDBGroupStore,
  useGracefulRequest,
  useProjectByName,
} from "@/store";
import {
  databaseGroupNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { DatabaseGroupView } from "@/types/proto-es/v1/database_group_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  hasProjectPermissionV2,
  PERMISSIONS_FOR_DATABASE_CHANGE_ISSUE,
} from "@/utils";

interface LocalState {
  editing: boolean;
}

const props = defineProps<{
  projectId: string;
  databaseGroupName: string;
  allowEdit: boolean;
}>();

const dbGroupStore = useDBGroupStore();
const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);
const { t } = useI18n();
const dialog = useDialog();
const router = useRouter();
const state = reactive<LocalState>({
  editing: false,
});

const databaseGroupResourceName = computed(() => {
  return `${project.value.name}/${databaseGroupNamePrefix}${props.databaseGroupName}`;
});

const databaseGroup = computed(() => {
  return dbGroupStore.getDBGroupByName(databaseGroupResourceName.value);
});

const hasMatchedDatabases = computed(
  () => (databaseGroup.value?.matchedDatabases.length ?? 0) > 0
);

const hasDatabaseGroupFeature = featureToRef(
  PlanFeature.FEATURE_DATABASE_GROUPS
);

const allowDelete = computed(() => {
  return hasProjectPermissionV2(project.value, "bb.databaseGroups.delete");
});

const dropdownOptions = computed((): DropdownOption[] => {
  if (!allowDelete.value) {
    return [];
  }
  return [
    {
      key: "delete",
      label: t("common.delete"),
    },
  ];
});

const handleDropdownSelect = (key: string) => {
  if (key === "delete" && databaseGroup.value) {
    dialog.warning({
      title: t("database-group.delete-group", {
        name: databaseGroup.value.title,
      }),
      content: t("common.cannot-undo-this-action"),
      negativeText: t("common.cancel"),
      positiveText: t("common.delete"),
      onPositiveClick: () => {
        useGracefulRequest(async () => {
          await dbGroupStore.deleteDatabaseGroup(
            databaseGroupResourceName.value
          );
          router.push({
            name: PROJECT_V1_ROUTE_DATABASE_GROUPS,
          });
        });
      },
    });
  }
};

watchEffect(async () => {
  await dbGroupStore.getOrFetchDBGroupByName(databaseGroupResourceName.value, {
    skipCache: true,
    view: DatabaseGroupView.FULL,
  });
});

const { overrideMainContainerClass } = useBodyLayoutContext();

overrideMainContainerClass("py-0!");
</script>
