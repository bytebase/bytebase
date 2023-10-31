import { InjectionKey, Ref, computed, inject, provide, ref } from "vue";
import {
  databaseForTask,
  getLocalSheetByName,
  isGroupingChangeTaskV1,
  sheetNameForSpec,
  specForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { ComposedIssue } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { IssueStatus } from "@/types/proto/v1/issue_service";
import {
  Plan_ChangeDatabaseConfig_Type,
  Plan_Spec,
  Task,
  Task_Status,
  Task_Type,
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

  const allowGhostForDatabase = computed(() => {
    const db = databaseForTask(issue.value, task.value);
    return (
      db.instanceEntity.engine === Engine.MYSQL &&
      semverCompare(
        db.instanceEntity.engineVersion,
        MIN_GHOST_SUPPORT_MYSQL_VERSION,
        "gte"
      )
    );
  });
  const viewType = computed((): GhostUIViewType => {
    if (isGroupingChangeTaskV1(issue.value, task.value)) {
      return "NONE";
    }

    if (
      ![
        Task_Type.DATABASE_SCHEMA_UPDATE,
        Task_Type.DATABASE_SCHEMA_UPDATE_GHOST_SYNC,
      ].includes(task.value.type)
    ) {
      return "NONE";
    }
    if (!allowGhostForDatabase.value) {
      return "NONE";
    }

    const spec = specForTask(issue.value.planEntity, task.value);
    return spec?.changeDatabaseConfig?.type ===
      Plan_ChangeDatabaseConfig_Type.MIGRATE_GHOST
      ? "ON"
      : "OFF";
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
  if (issue.status !== IssueStatus.OPEN) return false;
  return [
    Task_Status.NOT_STARTED,
    Task_Status.FAILED,
    Task_Status.CANCELED,
  ].includes(task.status);
};
