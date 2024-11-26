import { uniqBy } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref } from "vue";
import { useI18n } from "vue-i18n";
import {
  databaseForTask,
  getLocalSheetByName,
  isGroupingChangeTaskV1,
  sheetNameForSpec,
  specForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { useCurrentUserV1, useSubscriptionV1Store } from "@/store";
import type { ComposedDatabase, ComposedIssue } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import {
  Plan_ChangeDatabaseConfig_Type,
  type Plan_Spec,
} from "@/types/proto/v1/plan_service";
import type { Task } from "@/types/proto/v1/rollout_service";
import {
  Task_Status,
  task_StatusToJSON,
} from "@/types/proto/v1/rollout_service";
import {
  MIN_GHOST_SUPPORT_MARIADB_VERSION,
  MIN_GHOST_SUPPORT_MYSQL_VERSION,
  extractUserResourceName,
  flattenTaskV1List,
  getSheetStatement,
  hasProjectPermissionV2,
  semverCompare,
} from "@/utils";

export type GhostUIViewType = "NONE" | "OFF" | "ON";

export type IssueGhostContext = {
  viewType: Ref<GhostUIViewType>;
  showFlagsPanel: Ref<boolean>;
  denyEditGhostFlagsReasons: Ref<string[]>;
  showFeatureModal: Ref<boolean>;
  showMissingInstanceLicense: Ref<boolean>;
  toggleGhost: (on: boolean) => Promise<void>;
};

export const KEY = Symbol(
  "bb.issue.context.ghost"
) as InjectionKey<IssueGhostContext>;

export const useIssueGhostContext = () => {
  return inject(KEY)!;
};

export const provideIssueGhostContext = () => {
  const { t } = useI18n();
  const me = useCurrentUserV1();
  const {
    isCreating,
    issue,
    selectedTask: task,
    reInitialize,
  } = useIssueContext();

  const viewType = computed((): GhostUIViewType => {
    return ghostViewTypeForTask(issue.value, task.value);
  });
  const showFlagsPanel = ref(false);

  const denyEditGhostFlagsReasons = computed(() => {
    if (isCreating.value) {
      return [];
    }

    if (issue.value.status !== IssueStatus.OPEN) {
      return [t("issue.error.issue-is-not-open")];
    }

    const errors: string[] = [];

    if (extractUserResourceName(issue.value.creator) !== me.value.email) {
      if (
        !hasProjectPermissionV2(issue.value.projectEntity, "bb.plans.update")
      ) {
        return [t("issue.error.you-don-have-privilege-to-edit-this-issue")];
      }
    }
    if (!allowChangeTaskGhostFlags(issue.value, task.value)) {
      errors.push(
        t("task.online-migration.error.x-status-task-is-not-editable", {
          status: task_StatusToJSON(task.value.status),
        })
      );
    }
    return errors;
  });

  const showFeatureModal = ref(false);
  const showMissingInstanceLicense = computed(() => {
    const instances = uniqBy(
      flattenTaskV1List(issue.value.rolloutEntity).map(
        (task) => databaseForTask(issue.value, task).instanceResource
      ),
      (instance) => instance.name
    );
    const subscriptionStore = useSubscriptionV1Store();
    return instances.some((instance) => {
      return subscriptionStore.instanceMissingLicense(
        "bb.feature.online-migration",
        instance
      );
    });
  });

  const toggleGhost = async (on: boolean) => {
    const overrides: Record<string, string> = {};
    if (on) {
      overrides["ghost"] = "1";
    } else {
      overrides["ghost"] = "";
    }

    // Backup editing statements to `overrides.sqlMap`
    const flattenSpecs = (issue.value.planEntity?.steps ?? []).flatMap(
      (step) => step.specs
    );
    const sqlMap: Record<string, string> = {};
    flattenSpecs.forEach((spec) => {
      const target = spec.changeDatabaseConfig!.target;
      const sheetName = sheetNameForSpec(spec);
      const sheet = getLocalSheetByName(sheetName);
      sqlMap[target] = getSheetStatement(sheet);
    });
    const sqlMapStorageKey = `bb.issues.sql-map.${uuidv4()}`;
    localStorage.setItem(sqlMapStorageKey, JSON.stringify(sqlMap));
    overrides["sqlMapStorageKey"] = sqlMapStorageKey;

    await reInitialize(overrides);
  };

  const context: IssueGhostContext = {
    viewType,
    showFlagsPanel,
    denyEditGhostFlagsReasons,
    showFeatureModal,
    showMissingInstanceLicense,
    toggleGhost,
  };

  provide(KEY, context);

  return context;
};

export const allowChangeTaskGhostFlags = (issue: ComposedIssue, task: Task) => {
  return [
    Task_Status.STATUS_UNSPECIFIED, // Pending create
    Task_Status.NOT_STARTED,
    Task_Status.FAILED,
    Task_Status.CANCELED,
  ].includes(task.status);
};

export const allowGhostForDatabase = (database: ComposedDatabase) => {
  if (
    !useSubscriptionV1Store().hasInstanceFeature(
      "bb.feature.online-migration",
      database.instanceResource
    )
  ) {
    return false;
  }

  return (
    (database.instanceResource.engine === Engine.MYSQL &&
      semverCompare(
        database.instanceResource.engineVersion,
        MIN_GHOST_SUPPORT_MYSQL_VERSION,
        "gte"
      )) ||
    (database.instanceResource.engine === Engine.MARIADB &&
      semverCompare(
        database.instanceResource.engineVersion,
        MIN_GHOST_SUPPORT_MARIADB_VERSION,
        "gte"
      ))
  );
};

export const allowGhostForSpec = (spec: Plan_Spec | undefined) => {
  const config = spec?.changeDatabaseConfig;
  if (!config) return false;

  return [
    Plan_ChangeDatabaseConfig_Type.MIGRATE,
    Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST,
  ].includes(config.type);
};

export const allowGhostForTask = (issue: ComposedIssue, task: Task) => {
  return (
    allowGhostForSpec(specForTask(issue.planEntity, task)) &&
    allowGhostForDatabase(databaseForTask(issue, task))
  );
};

export const ghostViewTypeForTask = (
  issue: ComposedIssue,
  task: Task
): GhostUIViewType => {
  if (isGroupingChangeTaskV1(issue, task)) {
    return "NONE";
  }

  const spec = specForTask(issue.planEntity, task);
  const config = spec?.changeDatabaseConfig;
  if (!config) {
    return "NONE";
  }
  if (config.type === Plan_ChangeDatabaseConfig_Type.MIGRATE) {
    return "OFF";
  }
  if (config.type === Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST) {
    return "ON";
  }
  return "NONE";
};
