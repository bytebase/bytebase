import Emittery from "emittery";
import { cloneDeep } from "lodash-es";
import {
  computed,
  inject,
  provide,
  unref,
  type InjectionKey,
  type Ref,
} from "vue";
import { useI18n } from "vue-i18n";
import { databaseForSpec, isDatabaseChangeSpec } from "@/components/Plan/logic";
import { planServiceClient } from "@/grpcweb";
import { pushNotification, useCurrentUserV1, extractUserId } from "@/store";
import type { ComposedProject } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import type { Issue } from "@/types/proto/v1/issue_service";
import {
  Plan_ChangeDatabaseConfig_Type,
  type Plan,
  type Plan_Spec,
} from "@/types/proto/v1/plan_service";
import {
  flattenSpecList,
  hasProjectPermissionV2,
  isNullOrUndefined,
} from "@/utils";
import { getArchiveDatabase } from "./utils";

const PRE_BACKUP_AVAILABLE_ENGINES = [
  Engine.MYSQL,
  Engine.TIDB,
  Engine.MSSQL,
  Engine.ORACLE,
  Engine.POSTGRES,
];

const KEY = Symbol(
  "bb.changelist.dashboard"
) as InjectionKey<PreBackupSettingContext>;

export const usePreBackupSettingContext = () => {
  return inject(KEY)!;
};

export const providePreBackupSettingContext = (refs: {
  project: Ref<ComposedProject>;
  plan: Ref<Plan>;
  selectedSpec: Ref<Plan_Spec>;
  isCreating: Ref<boolean>;
  issue?: Ref<Issue>;
}) => {
  const { t } = useI18n();
  const currentUserV1 = useCurrentUserV1();
  const { project, plan, selectedSpec, isCreating } = refs;

  const events = new Emittery<{
    update: boolean;
  }>();

  const database = computed(() =>
    databaseForSpec(project.value, selectedSpec.value)
  );

  const shouldShow = computed((): boolean => {
    if (!isDatabaseChangeSpec(selectedSpec.value)) {
      return false;
    }
    if (
      selectedSpec.value.changeDatabaseConfig?.type !==
      Plan_ChangeDatabaseConfig_Type.DATA
    ) {
      return false;
    }
    const { engine } = database.value.instanceResource;
    if (!PRE_BACKUP_AVAILABLE_ENGINES.includes(engine)) {
      return false;
    }
    return true;
  });

  const allowChange = computed((): boolean => {
    // Disallow pre-backup if no backup available for the target database.
    if (!database.value.backupAvailable) {
      return false;
    }

    // Allow toggle pre-backup when creating.
    if (isCreating.value) {
      return true;
    }

    // Allowed to the plan/issue creator.
    if (currentUserV1.value.email === extractUserId(unref(plan).creator)) {
      return true;
    }

    // Allowed to the permission holder.
    if (hasProjectPermissionV2(project.value, "bb.plans.update")) {
      return true;
    }

    return false;
  });

  const enabled = computed((): boolean => {
    const preBackupDatabase =
      selectedSpec.value.changeDatabaseConfig?.preUpdateBackupDetail?.database;
    return !isNullOrUndefined(preBackupDatabase) && preBackupDatabase !== "";
  });

  const archiveDatabase = computed((): string =>
    getArchiveDatabase(database.value.instanceResource.engine)
  );

  const toggle = async (on: boolean) => {
    if (isCreating.value) {
      if (selectedSpec.value && selectedSpec.value.changeDatabaseConfig) {
        if (on) {
          selectedSpec.value.changeDatabaseConfig.preUpdateBackupDetail = {
            database:
              database.value.instance + "/databases/" + archiveDatabase.value,
          };
        } else {
          selectedSpec.value.changeDatabaseConfig.preUpdateBackupDetail =
            undefined;
        }
      }
    } else {
      const planPatch = cloneDeep(unref(plan));
      const spec = flattenSpecList(planPatch).find((s) => {
        return s.id === selectedSpec.value.id;
      });
      if (!planPatch || !spec || !spec.changeDatabaseConfig) {
        // Should not reach here.
        throw new Error(
          "Plan or spec is not defined. Cannot update pre-backup setting."
        );
      }
      if (on) {
        spec.changeDatabaseConfig.preUpdateBackupDetail = {
          database:
            database.value.instance + "/databases/" + archiveDatabase.value,
        };
      } else {
        spec.changeDatabaseConfig.preUpdateBackupDetail = undefined;
      }

      await planServiceClient.updatePlan({
        plan: planPatch,
        updateMask: ["steps"],
      });

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    }

    // Emit the update event.
    events.emit("update", on);
  };

  const context = {
    shouldShow,
    enabled,
    allowChange,
    database,
    events,
    toggle,
  };

  provide(KEY, context);

  return context;
};

type PreBackupSettingContext = ReturnType<
  typeof providePreBackupSettingContext
>;
