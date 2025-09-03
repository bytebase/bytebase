import { create } from "@bufbuild/protobuf";
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
import { isDatabaseChangeSpec, targetsForSpec } from "@/components/Plan/logic";
import { planServiceClientConnect } from "@/grpcweb";
import { useCurrentUserV1, extractUserId, useDatabaseV1Store } from "@/store";
import { isValidDatabaseName } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { IssueStatus, type Issue } from "@/types/proto-es/v1/issue_service_pb";
import { UpdatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import {
  Plan_ChangeDatabaseConfig_Type,
  type Plan,
  type Plan_Spec,
} from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { Task, Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { flattenTaskV1List, hasProjectPermissionV2 } from "@/utils";
import { BACKUP_AVAILABLE_ENGINES } from "./common";

export const PRE_BACKUP_AVAILABLE_ENGINES = [
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
  project: Ref<Project>;
  plan: Ref<Plan>;
  selectedSpec: Ref<Plan_Spec | undefined>;
  selectedTask?: Ref<Task | undefined>;
  issue?: Ref<Issue | undefined>;
  rollout?: Ref<Rollout | undefined>;
  readonly?: Ref<boolean>;
}) => {
  const currentUser = useCurrentUserV1();
  const databaseStore = useDatabaseV1Store();
  const { isCreating, project, plan, selectedSpec, issue, rollout, readonly } =
    refs;

  const events = new Emittery<{
    update: never;
  }>();

  const databases = computed(() => {
    const targets = selectedSpec.value
      ? targetsForSpec(selectedSpec.value)
      : [];
    return targets
      .map((target) => databaseStore.getDatabaseByName(target))
      .filter((db) => isValidDatabaseName(db.name));
  });

  const shouldShow = computed((): boolean => {
    if (
      !selectedSpec.value ||
      selectedSpec.value.config?.case !== "changeDatabaseConfig" ||
      selectedSpec.value.config.value.type !==
        Plan_ChangeDatabaseConfig_Type.DATA
    ) {
      return false;
    }
    if (isDatabaseChangeSpec(selectedSpec.value)) {
      // If any of the databases in the spec is not supported, do not show.
      if (
        !databases.value.every((db) =>
          BACKUP_AVAILABLE_ENGINES.includes(db.instanceResource.engine)
        )
      ) {
        return false;
      }
    }
    return true;
  });

  const allowChange = computed((): boolean => {
    // If readonly mode, disallow changes
    if (readonly?.value) {
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
    if (selectedSpec.value?.config?.case === "changeDatabaseConfig") {
      return Boolean(selectedSpec.value.config.value.enablePriorBackup);
    }
    return false;
  });

  const toggle = async (on: boolean) => {
    if (isCreating.value) {
      if (selectedSpec.value?.config?.case === "changeDatabaseConfig") {
        if (on) {
          selectedSpec.value.config.value.enablePriorBackup = true;
        } else {
          selectedSpec.value.config.value.enablePriorBackup = false;
        }
      }
    } else {
      const planPatch = cloneDeep(unref(plan));
      const spec = planPatch.specs.find((s) => {
        return s.id === selectedSpec.value?.id;
      });
      if (!planPatch || !spec || spec.config.case !== "changeDatabaseConfig") {
        // Should not reach here.
        throw new Error(
          "Plan or spec is not defined. Cannot update pre-backup setting."
        );
      }
      if (on) {
        spec.config.value.enablePriorBackup = true;
      } else {
        spec.config.value.enablePriorBackup = false;
      }

      const request = create(UpdatePlanRequestSchema, {
        plan: planPatch,
        updateMask: { paths: ["specs"] },
      });
      await planServiceClientConnect.updatePlan(request);
    }

    // Emit the update event.
    events.emit("update");
  };

  const context = {
    shouldShow,
    enabled,
    allowChange,
    databases,
    events,
    toggle,
    selectedSpec,
  };

  provide(KEY, context);

  return context;
};

type PreBackupSettingContext = ReturnType<
  typeof providePreBackupSettingContext
>;
