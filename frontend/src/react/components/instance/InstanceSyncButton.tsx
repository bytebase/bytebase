import { ChevronDown } from "lucide-react";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import { pushNotification, useDatabaseV1Store } from "@/store";
import { SyncStatus } from "@/types/proto-es/v1/database_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

interface InstanceSyncButtonProps {
  type?: "default" | "primary";
  size?: "small" | "medium";
  disabled?: boolean;
  instanceName?: string;
  instanceTitle?: string;
  onSyncSchema: (enableFullSync: boolean) => void;
}

export function InstanceSyncButton({
  disabled = false,
  instanceName = "",
  instanceTitle = "",
  onSyncSchema,
}: InstanceSyncButtonProps) {
  const { t } = useTranslation();
  const [syncing, setSyncing] = useState(false);
  const [open, setOpen] = useState(false);

  const hasPermission = hasWorkspacePermissionV2("bb.instances.sync");

  const syncSchema = useCallback(
    async (option: "sync-all" | "sync-new") => {
      setOpen(false);
      try {
        setSyncing(true);
        const displayName = instanceTitle || instanceName || "";
        if (displayName) {
          pushNotification({
            module: "bytebase",
            style: "INFO",
            title: t("db.syncing-databases-for-instance", { 0: displayName }),
          });
        } else {
          pushNotification({
            module: "bytebase",
            style: "INFO",
            title: t("db.start-to-sync-schema"),
          });
        }

        onSyncSchema(option === "sync-all");

        if (instanceName) {
          const displayName = instanceTitle || instanceName;
          let notificationShown = false;

          const checkSyncStatus = async () => {
            const databaseStore = useDatabaseV1Store();
            const { databases } = await databaseStore.fetchDatabases({
              parent: instanceName,
              pageSize: 1000,
              silent: true,
            });
            const stillSyncing = databases.filter(
              (db) => db.syncStatus === SyncStatus.SYNC_STATUS_UNSPECIFIED
            );
            const failed = databases.filter(
              (db) => db.syncStatus === SyncStatus.FAILED
            );
            if (stillSyncing.length > 0) return "syncing";
            if (failed.length > 0) {
              pushNotification({
                module: "bytebase",
                style: "WARN",
                title: t("db.n-databases-had-sync-errors", {
                  0: failed.length,
                }),
              });
            } else {
              pushNotification({
                module: "bytebase",
                style: "SUCCESS",
                title: t("database.sync-complete-for-instance", {
                  0: displayName,
                }),
              });
            }
            notificationShown = true;
            return "complete";
          };

          setTimeout(async () => {
            try {
              await checkSyncStatus();
            } catch {
              /* retry at 30s */
            }
          }, 15000);

          setTimeout(async () => {
            if (notificationShown) return;
            try {
              await checkSyncStatus();
            } catch {
              /* user can check database list */
            }
          }, 30000);
        }
      } catch (error) {
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("db.failed-to-sync-schema"),
          description: (error as Error).message,
        });
      } finally {
        setSyncing(false);
      }
    },
    [instanceName, instanceTitle, onSyncSchema, t]
  );

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger
        render={
          <Button
            variant="outline"
            disabled={!hasPermission || disabled || syncing}
          />
        }
      >
        {syncing ? t("instance.syncing") : t("instance.sync.self")}
        <ChevronDown className="ml-1 size-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-[140px]">
        <DropdownMenuItem
          title={t("instance.sync.sync-all-tip")}
          onClick={() => syncSchema("sync-all")}
        >
          {t("instance.sync.sync-all")}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => syncSchema("sync-new")}>
          {t("instance.sync.sync-new")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
