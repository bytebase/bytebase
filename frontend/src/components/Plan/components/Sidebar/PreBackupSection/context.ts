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
import { databaseForSpec, isDatabaseChangeSpec } from "@/components/Plan/logic";
import { databaseForTask } from "@/components/Rollout/RolloutDetail";
import { planServiceClient } from "@/grpcweb";
import { useCurrentUserV1, extractUserId } from "@/store";
import { unknownDatabase, type ComposedProject } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { IssueStatus, type Issue } from "@/types/proto/v1/issue_service";
import {
  Plan_ChangeDatabaseConfig_Type,
  type Plan,
  type Plan_Spec,
} from "@/types/proto/v1/plan_service";
import {
  Task,
  Task_Status,
  type Rollout,
} from "@/types/proto/v1/rollout_service";
import {
  flattenSpecList,
  flattenTaskV1List,
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
  "bb.plan.setting.pre-backup"
) as InjectionKey<PreBackupSettingContext>;

export const usePreBackupSettingContext = () => {
  return inject(KEY)!;
};

export const providePreBackupSettingContext = (refs: {
  isCreating: Ref<boolean>;
  project: Ref<ComposedProject>;
  plan: Ref<Plan>;
  selectedSpec: Ref<Plan_Spec | undefined>;
  selectedTask: Ref<Task | undefined>;
  issue?: Ref<Issue | undefined>;
  rollout?: Ref<Rollout | undefined>;
}) => {
  const currentUser = useCurrentUserV1();
  const {
    isCreating,
    project,
    plan,
    selectedSpec,
    selectedTask,
    issue,
    rollout,
  } = refs;

  const events = new Emittery<{
    update: boolean;
  }>();

  const database = computed(() => {
    if (selectedTask.value) {
      return databaseForTask(project.value, selectedTask.value);
    } else if (selectedSpec.value) {
      return databaseForSpec(project.value, selectedSpec.value);
    }
    return unknownDatabase();
  });

  const shouldShow = computed((): boolean => {
    if (
      !selectedSpec.value ||
      selectedSpec.value.changeDatabaseConfig?.type !==
        Plan_ChangeDatabaseConfig_Type.DATA
    ) {
      return false;
    }
    // Always show pre-backup for database change spec.
    if (isDatabaseChangeSpec(selectedSpec.value)) {
      return true;
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

    // If issue is not open, disallow.
    if (issue?.value && issue.value.status !== IssueStatus.OPEN) {
      return false;
    }

    // If task of the spec is running/done/etc..., disallow.
    if (rollout?.value) {
      const tasks = flattenTaskV1List(rollout.value);
      const task = tasks.find((t) => t.specId === selectedSpec.value?.id);
      if (
        task &&
        [
          Task_Status.PENDING,
          Task_Status.RUNNING,
          Task_Status.DONE,
          Task_Status.SKIPPED,
        ].includes(task.status)
      ) {
        return false;
      }
    }

    // Allowed to the plan/issue creator.
    if (currentUser.value.email === extractUserId(unref(plan).creator)) {
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
      selectedSpec.value?.changeDatabaseConfig?.preUpdateBackupDetail?.database;
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
        return s.id === selectedSpec.value?.id;
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
