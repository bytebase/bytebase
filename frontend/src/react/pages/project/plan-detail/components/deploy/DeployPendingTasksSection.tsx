import { create } from "@bufbuild/protobuf";
import { ChevronDown, ChevronRight, Loader2 } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { rolloutServiceClientConnect } from "@/connect";
import { Button } from "@/react/components/ui/button";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { pushNotification } from "@/store";
import { isValidDatabaseGroupName, isValidDatabaseName } from "@/types";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import {
  CreateRolloutRequestSchema,
  type Rollout,
  type Task,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractEnvironmentResourceName,
  extractPlanNameFromRolloutName,
} from "@/utils";
import { generateRolloutPreview } from "../../utils/rolloutPreview";
import {
  DatabaseGroupTarget,
  DatabaseTarget,
} from "../PlanDetailChangesBranch";
import type { PendingTaskGroup } from "./types";

async function buildPendingTaskGroups(
  plan: Plan,
  rollout: Rollout | undefined,
  projectName: string
): Promise<PendingTaskGroup[]> {
  const preview = await generateRolloutPreview(plan, projectName);
  const existingTasks = new Set<string>();
  for (const stage of rollout?.stages ?? []) {
    for (const task of stage.tasks) {
      existingTasks.add(`${task.target}:${task.specId}`);
    }
  }
  const groups = new Map<string, Task[]>();
  for (const stage of preview.stages) {
    for (const task of stage.tasks) {
      const key = `${task.target}:${task.specId}`;
      if (existingTasks.has(key)) continue;
      const tasks = groups.get(stage.environment) ?? [];
      tasks.push(task);
      groups.set(stage.environment, tasks);
    }
  }
  return Array.from(groups.entries()).map(([environment, tasks]) => ({
    environment,
    tasks,
  }));
}

export function DeployPendingTasksSection({
  onCreated,
  open,
  onOpenChange,
  plan,
  projectName,
  rollout,
}: {
  onCreated: () => Promise<void> | void;
  onOpenChange: (open: boolean) => void;
  open: boolean;
  plan: Plan;
  projectName: string;
  rollout?: Rollout;
}) {
  const { t } = useTranslation();
  const [creatingEnv, setCreatingEnv] = useState<string | null>(null);
  const [expandedEnvs, setExpandedEnvs] = useState<Set<string>>(new Set());
  const [groups, setGroups] = useState<PendingTaskGroup[]>([]);
  const [loading, setLoading] = useState(false);
  const rolloutKey =
    rollout?.stages
      .map(
        (stage) =>
          `${stage.name}:${stage.tasks.map((task) => `${task.target}:${task.specId}`).join(",")}`
      )
      .join("|") ?? "";
  const planKey = `${plan.name}:${plan.specs.map((spec) => spec.id).join(",")}`;

  useEffect(() => {
    if (!open) return;
    let canceled = false;
    const load = async () => {
      setLoading(true);
      try {
        const next = await buildPendingTaskGroups(plan, rollout, projectName);
        if (canceled) return;
        setGroups(next);
        setExpandedEnvs(new Set(next.map((group) => group.environment)));
      } finally {
        if (!canceled) setLoading(false);
      }
    };
    void load();
    return () => {
      canceled = true;
    };
  }, [open, planKey, projectName, rolloutKey]);

  const hasPendingTasks = groups.length > 0;
  const rolloutParent = rollout
    ? extractPlanNameFromRolloutName(rollout.name)
    : plan.name;

  const toggleEnv = (env: string) => {
    setExpandedEnvs((prev) => {
      const next = new Set(prev);
      if (next.has(env)) next.delete(env);
      else next.add(env);
      return next;
    });
  };

  return (
    <Sheet onOpenChange={onOpenChange} open={open}>
      <SheetContent
        className="w-[25rem] max-w-[calc(100vw-2rem)]"
        width="standard"
      >
        <SheetHeader>
          <SheetTitle>{t("rollout.pending-tasks-preview.title")}</SheetTitle>
        </SheetHeader>
        <SheetBody className="gap-y-4">
          <p className="text-sm text-control-light">
            {t("rollout.pending-tasks-preview.description")}
          </p>

          {loading ? (
            <div className="flex justify-center py-8 text-control-light">
              <Loader2 className="h-5 w-5 animate-spin" />
            </div>
          ) : !hasPendingTasks ? (
            <p className="py-8 text-center text-control-light">
              {t("rollout.pending-tasks-preview.no-pending-tasks")}
            </p>
          ) : (
            <div className="space-y-4">
              {groups.map((group) => (
                <div key={group.environment} className="rounded-lg border">
                  <div className="flex items-center gap-2 bg-gray-50 px-3 py-2">
                    <button
                      className="flex flex-1 items-center gap-2 text-left"
                      onClick={() => toggleEnv(group.environment)}
                      type="button"
                    >
                      {expandedEnvs.has(group.environment) ? (
                        <ChevronDown className="h-4 w-4 text-gray-500" />
                      ) : (
                        <ChevronRight className="h-4 w-4 text-gray-500" />
                      )}
                      <span className="font-medium">
                        {extractEnvironmentResourceName(group.environment)}
                      </span>
                      <span className="text-xs text-control-light">
                        {t("rollout.pending-tasks-preview.task-count", {
                          count: group.tasks.length,
                        })}
                      </span>
                    </button>
                    <Button
                      disabled={Boolean(creatingEnv)}
                      onClick={async () => {
                        try {
                          setCreatingEnv(group.environment);
                          await rolloutServiceClientConnect.createRollout(
                            create(CreateRolloutRequestSchema, {
                              parent: rolloutParent,
                              target: group.environment,
                            })
                          );
                          pushNotification({
                            module: "bytebase",
                            style: "SUCCESS",
                            title: t("common.success"),
                          });
                          await onCreated();
                          setGroups((prev) =>
                            prev.filter(
                              (item) => item.environment !== group.environment
                            )
                          );
                        } catch (error) {
                          pushNotification({
                            module: "bytebase",
                            style: "CRITICAL",
                            title: t("common.error"),
                            description: String(error),
                          });
                        } finally {
                          setCreatingEnv(null);
                        }
                      }}
                      size="xs"
                    >
                      {creatingEnv === group.environment && (
                        <Loader2 className="h-3.5 w-3.5 animate-spin" />
                      )}
                      {t("common.create")}
                    </Button>
                  </div>
                  {expandedEnvs.has(group.environment) && (
                    <ul className="space-y-1 px-3 py-2">
                      {group.tasks.map((task) => (
                        <li key={`${task.target}:${task.specId}`}>
                          {isValidDatabaseName(task.target) ? (
                            <DatabaseTarget
                              showEnvironment
                              target={task.target}
                            />
                          ) : isValidDatabaseGroupName(task.target) ? (
                            <DatabaseGroupTarget target={task.target} />
                          ) : (
                            <span className="text-sm text-control">
                              {task.target}
                            </span>
                          )}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              ))}
            </div>
          )}
        </SheetBody>
      </SheetContent>
    </Sheet>
  );
}
