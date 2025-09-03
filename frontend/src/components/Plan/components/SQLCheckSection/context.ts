import type { InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref } from "vue";
import { unknownDatabase } from "@/types";
import type { Plan, Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { CheckReleaseResponse_CheckResult } from "@/types/proto-es/v1/release_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask } from "@/utils";
import { databaseForSpec } from "../../logic";

export const KEY = Symbol(
  "bb.plan.context.sql-checks"
) as InjectionKey<SQLCheckContext>;

export const usePlanSQLCheckContext = () => {
  return inject(KEY)!;
};

export const providePlanSQLCheckContext = (refs: {
  project: Ref<Project>;
  plan: Ref<Plan>;
  selectedSpec: Ref<Plan_Spec>;
  selectedTask?: Ref<Task | undefined>;
}) => {
  const resultMap = ref<Record<string, CheckReleaseResponse_CheckResult>>({});
  const { project, plan, selectedSpec, selectedTask } = refs;

  const database = computed(() => {
    if (selectedTask?.value) {
      return databaseForTask(project.value, selectedTask.value);
    } else if (selectedSpec.value) {
      // TODO(steven): handle db group as target in spec.
      return databaseForSpec(project.value, selectedSpec.value);
    }
    return unknownDatabase();
  });

  const context = {
    project,
    plan,
    database,
    selectedSpec,
    resultMap,
    upsertResult: (key: string, result: CheckReleaseResponse_CheckResult) => {
      resultMap.value[key] = result;
    },
  };

  provide(KEY, context);

  return context;
};

type SQLCheckContext = ReturnType<typeof providePlanSQLCheckContext>;
