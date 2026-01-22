<template>
  <NDropdown
    :trigger="'click'"
    :options="syncInstnceOptions"
    :render-label="renderDropdownLabel"
    :disabled="disabled || syncingSchema"
    @select="syncSchema"
  >
    <PermissionGuardWrapper
      v-slot="slotProps"
      :permissions="['bb.instances.sync']"
    >
      <NButton
        icon-placement="right"
        :loading="syncingSchema"
        :quaternary="quaternary"
        :disabled="slotProps.disabled || disabled"
        :size="size"
        :type="type"
      >
        <template #icon>
          <ChevronDownIcon class="w-4" />
        </template>
        <template v-if="syncingSchema">
          {{ $t("instance.syncing") }}
        </template>
        <template v-else>
          {{ $t("instance.sync.self") }}
        </template>
      </NButton>
    </PermissionGuardWrapper>
  </NDropdown>
</template>

<script lang="tsx" setup>
import type { ConnectError } from "@connectrpc/connect";
import { ChevronDownIcon } from "lucide-vue-next";
import type { DropdownOption } from "naive-ui";
import { NButton, NDropdown, NTooltip } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import { pushNotification, useDatabaseV1Store } from "@/store";
import { SyncStatus } from "@/types/proto-es/v1/database_service_pb";

const props = withDefaults(
  defineProps<{
    type?: "default" | "primary";
    size?: "small" | "medium";
    disabled?: boolean;
    quaternary?: boolean;
    instanceName?: string;
    instanceTitle?: string;
  }>(),
  {
    type: "default",
    size: "medium",
    disabled: false,
    quaternary: false,
    instanceName: "",
    instanceTitle: "",
  }
);

const emit = defineEmits<{
  (name: "sync-schema", enableFullSync: boolean): void;
}>();

const { t } = useI18n();
const syncingSchema = ref<boolean>(false);

type SyncInstanceOption = "sync-all" | "sync-new";

const renderDropdownLabel = (option: DropdownOption) => {
  if (option.key === "sync-new") {
    return <span>{option.label}</span>;
  }
  return (
    <NTooltip placement="top-start">
      {{
        trigger: () => option.label,
        default: () => (
          <span class="text-sm text-nowrap">
            {t("instance.sync.sync-all-tip")}
          </span>
        ),
      }}
    </NTooltip>
  );
};

const syncInstnceOptions = computed((): DropdownOption[] => {
  return [
    {
      key: "sync-all",
      label: t("instance.sync.sync-all"),
    },
    {
      key: "sync-new",
      label: t("instance.sync.sync-new"),
    },
  ];
});

const syncSchema = async (option: SyncInstanceOption) => {
  try {
    syncingSchema.value = true;

    // Show immediate "syncing" notification (honest - not claiming success yet)
    const displayName = props.instanceTitle || props.instanceName || "";
    if (displayName) {
      pushNotification({
        module: "bytebase",
        style: "INFO",
        title: t("db.syncing-databases-for-instance", [displayName]),
      });
    } else {
      // Fallback for batch sync or when instance info not provided
      pushNotification({
        module: "bytebase",
        style: "INFO",
        title: t("db.start-to-sync-schema"),
      });
    }

    emit("sync-schema", option === "sync-all" /* enable full sync */);

    // Two-phase delayed check for sync status
    if (props.instanceName) {
      const displayName = props.instanceTitle || props.instanceName;
      let notificationShown = false;

      const checkSyncStatus = async (): Promise<"complete" | "syncing"> => {
        const databaseStore = useDatabaseV1Store();
        const { databases } = await databaseStore.fetchDatabases({
          parent: props.instanceName,
          pageSize: 1000,
          silent: true,
        });

        const stillSyncing = databases.filter(
          (db) => db.syncStatus === SyncStatus.SYNC_STATUS_UNSPECIFIED
        );
        const failed = databases.filter(
          (db) => db.syncStatus === SyncStatus.FAILED
        );

        // If any databases still syncing, don't notify yet
        if (stillSyncing.length > 0) {
          return "syncing";
        }

        // All syncs complete - show appropriate notification
        if (failed.length > 0) {
          pushNotification({
            module: "bytebase",
            style: "WARN",
            title: t("db.n-databases-had-sync-errors", [failed.length]),
          });
        } else {
          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t("database.sync-complete-for-instance", [displayName]),
          });
        }
        notificationShown = true;
        return "complete";
      };

      // First check at 15 seconds (fast feedback for small instances)
      setTimeout(async () => {
        try {
          await checkSyncStatus();
        } catch {
          // Silently ignore - will retry at 30s
        }
      }, 15000);

      // Second check at 30 seconds (for larger instances)
      setTimeout(async () => {
        if (notificationShown) return;
        try {
          await checkSyncStatus();
          // If still syncing at 30s, don't notify - user can check database list
        } catch {
          // Silently ignore - user can check database list
        }
      }, 30000);
    }
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("db.failed-to-sync-schema"),
      description: (error as ConnectError).message,
    });
  } finally {
    syncingSchema.value = false;
  }
};
</script>
