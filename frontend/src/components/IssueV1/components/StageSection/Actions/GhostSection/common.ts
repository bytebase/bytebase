import { InjectionKey, Ref, computed, inject, provide, ref } from "vue";
import {
  databaseForTask,
  getLocalSheetByName,
  isGroupingChangeTaskV1,
  sheetNameForSpec,
  specForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { ComposedDatabase, ComposedIssue } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import {
  Plan_ChangeDatabaseConfig_Type,
  Plan_Spec,
  Task,
  Task_Status,
} from "@/types/proto/v1/rollout_service";
import {
  MIN_GHOST_SUPPORT_MYSQL_VERSION,
  getSheetStatement,
  semverCompare,
} from "@/utils";

export type GhostUIViewType = "NONE" | "OFF" | "ON";

export type IssueGhostContext = {
  viewType: Ref<GhostUIViewType>;
  showFlagsPanel: Ref<boolean>;
  toggleGhost: (spec: Plan_Spec, on: boolean) => Promise<void>;
};

export const KEY = Symbol(
  "bb.issue.context.ghost"
) as InjectionKey<IssueGhostContext>;

export const useIssueGhostContext = () => {
  return inject(KEY)!;
};

export const provideIssueGhostContext = () => {
  const { issue, selectedTask: task, reInitialize } = useIssueContext();

  const viewType = computed((): GhostUIViewType => {
    return ghostViewTypeForTask(issue.value, task.value);
  });
  const showFlagsPanel = ref(false);

  const toggleGhost = async (spec: Plan_Spec, on: boolean) => {
    const overrides: Record<string, string> = {};
    if (on) {
      overrides["ghost"] = "1";
    }

    // Backup editing statements to `overrides.sqlList`
    const flattenSpecs = (issue.value.planEntity?.steps ?? []).flatMap(
      (step) => step.specs
    );
    const sqlList: string[] = [];
    flattenSpecs.forEach((spec, i) => {
      const sheetName = sheetNameForSpec(spec);
      const sheet = getLocalSheetByName(sheetName);
      sqlList[i] = getSheetStatement(sheet);
    });
    overrides["sqlList"] = JSON.stringify(sqlList);

    await reInitialize(overrides);
  };

  const context: IssueGhostContext = {
    viewType,
    showFlagsPanel,
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
  return (
    database.instanceEntity.engine === Engine.MYSQL &&
    semverCompare(
      database.instanceEntity.engineVersion,
      MIN_GHOST_SUPPORT_MYSQL_VERSION,
      "gte"
    )
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
