import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { DatabaseBackup, Loader2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import {
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { ReadonlyMonaco } from "@/react/components/monaco";
import { Alert } from "@/react/components/ui/alert";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL } from "@/router/dashboard/projectV1";
import { pushNotification, useProjectV1Store, useSheetV1Store } from "@/store";
import {
  CreatePlanRequestSchema,
  Plan_ChangeDatabaseConfigSchema,
  Plan_SpecSchema,
  PlanSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  PreviewTaskRunRollbackRequestSchema,
  type Task,
  type TaskRun,
} from "@/types/proto-es/v1/rollout_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import {
  extractPlanUID,
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
  hasProjectPermissionV2,
} from "@/utils";
import { DatabaseTarget } from "./PlanDetailChangesBranch";

export function PlanDetailRollbackSheet({
  open,
  onOpenChange,
  projectName,
  rolloutName,
  items,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  projectName: string;
  rolloutName: string;
  items: Array<{
    task: Task;
    taskRun: TaskRun;
  }>;
}) {
  const { t } = useTranslation();
  const sheetStore = useSheetV1Store();
  const projectStore = useProjectV1Store();
  const normalizedProjectName = projectName.startsWith("projects/")
    ? projectName
    : `projects/${projectName}`;
  const project = useVueState(() =>
    projectStore.getProjectByName(normalizedProjectName)
  );
  const [loading, setLoading] = useState(false);
  const [selectedTaskRunNames, setSelectedTaskRunNames] = useState<string[]>(
    []
  );
  const [previews, setPreviews] = useState<
    Array<{
      task: Task;
      taskRun: TaskRun;
      statement?: string;
      error?: string;
    }>
  >([]);
  const [step, setStep] = useState<1 | 2>(1);

  useEffect(() => {
    if (!open) {
      setStep(1);
      setPreviews([]);
      setSelectedTaskRunNames([]);
      return;
    }
    if (items.length === 1) {
      setSelectedTaskRunNames([items[0].taskRun.name]);
    }
  }, [items, open]);

  useEffect(() => {
    if (!open || items.length !== 1 || step !== 1) {
      return;
    }
    const only = items[0];
    void (async () => {
      try {
        setLoading(true);
        const response =
          await rolloutServiceClientConnect.previewTaskRunRollback(
            create(PreviewTaskRunRollbackRequestSchema, {
              name: only.taskRun.name,
            }),
            {
              contextValues: createContextValues().set(silentContextKey, true),
            }
          );
        setPreviews([
          {
            task: only.task,
            taskRun: only.taskRun,
            statement: response.statement,
          },
        ]);
        setStep(2);
      } catch (error) {
        setPreviews([
          {
            task: only.task,
            taskRun: only.taskRun,
            error: String(error),
          },
        ]);
        setStep(2);
      } finally {
        setLoading(false);
      }
    })();
  }, [items, open, step]);

  const selectedItems = useMemo(
    () =>
      items.filter((item) => selectedTaskRunNames.includes(item.taskRun.name)),
    [items, selectedTaskRunNames]
  );

  const canCreate = useMemo(() => {
    return (
      !loading &&
      previews.length > 0 &&
      previews.every((preview) => !preview.error) &&
      previews.some((preview) => preview.statement) &&
      hasProjectPermissionV2(project, "bb.plans.create")
    );
  }, [loading, previews, project]);

  return (
    <Sheet onOpenChange={onOpenChange} open={open}>
      <SheetContent width="wide">
        <SheetHeader>
          <SheetTitle>{t("common.rollback")}</SheetTitle>
        </SheetHeader>
        <SheetBody className="gap-y-4">
          <div className="flex items-center gap-x-2">
            <Badge variant={step === 1 ? "secondary" : "default"}>
              {t("task.select-task")}
            </Badge>
            <Badge variant={step === 2 ? "secondary" : "default"}>
              {t("task-run.rollback.preview-statement.description")}
            </Badge>
          </div>
          {step === 1 ? (
            <div className="space-y-3">
              <div className="text-sm text-control-light">
                {t("task.select-task")}
              </div>
              <div className="overflow-hidden rounded-sm border">
                {items.map((item) => {
                  const checked = selectedTaskRunNames.includes(
                    item.taskRun.name
                  );
                  return (
                    <label
                      key={item.taskRun.name}
                      className="flex cursor-pointer items-center gap-3 border-b border-control-border px-3 py-2 last:border-b-0 hover:bg-gray-50"
                    >
                      <Checkbox
                        checked={checked}
                        onCheckedChange={(checked) => {
                          setSelectedTaskRunNames((prev) => {
                            if (checked) {
                              return [...prev, item.taskRun.name];
                            }
                            return prev.filter(
                              (name) => name !== item.taskRun.name
                            );
                          });
                        }}
                      />
                      <div className="min-w-0 space-y-1">
                        <div className="min-w-0 text-sm font-medium text-main">
                          <DatabaseTarget
                            showEnvironment
                            target={item.task.target}
                          />
                        </div>
                        <div className="text-xs text-control-light">
                          {item.taskRun.name}
                        </div>
                      </div>
                    </label>
                  );
                })}
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              <Alert
                variant="info"
                description={t(
                  "task-run.rollback.preview-statement.description"
                )}
              />
              {loading ? (
                <div className="flex items-center justify-center py-8 text-control-light">
                  <Loader2 className="h-5 w-5 animate-spin" />
                </div>
              ) : (
                previews.map((preview) => (
                  <div key={preview.taskRun.name} className="space-y-2">
                    <div className="text-sm font-medium text-main">
                      <DatabaseTarget
                        showEnvironment
                        target={preview.task.target}
                      />
                    </div>
                    {preview.error ? (
                      <div className="rounded-md border border-error/30 bg-error/5 p-3 text-sm text-error">
                        {preview.error}
                      </div>
                    ) : preview.statement ? (
                      <ReadonlyMonaco
                        className="relative h-auto max-h-[320px] min-h-[120px] overflow-hidden rounded-md border border-control-border"
                        content={preview.statement}
                        language="sql"
                      />
                    ) : (
                      <div className="flex items-center justify-center rounded-md border bg-gray-50 p-8">
                        <div className="flex flex-col items-center gap-y-2 text-center">
                          <DatabaseBackup className="h-6 w-6 text-gray-400" />
                          <p className="text-sm text-gray-500">
                            {t("task-run.rollback.no-statement-generated")}
                          </p>
                        </div>
                      </div>
                    )}
                  </div>
                ))
              )}
            </div>
          )}
        </SheetBody>
        <SheetFooter>
          {step === 1 ? (
            <Button onClick={() => onOpenChange(false)} variant="ghost">
              {t("common.cancel")}
            </Button>
          ) : (
            <Button onClick={() => setStep(1)} variant="ghost">
              {t("common.back")}
            </Button>
          )}
          {step === 1 ? (
            <Button
              disabled={selectedItems.length === 0}
              onClick={async () => {
                try {
                  setLoading(true);
                  const nextPreviews: Array<{
                    task: Task;
                    taskRun: TaskRun;
                    statement?: string;
                    error?: string;
                  }> = selectedItems.map((item) => ({
                    task: item.task,
                    taskRun: item.taskRun,
                  }));
                  for (const preview of nextPreviews) {
                    try {
                      const response =
                        await rolloutServiceClientConnect.previewTaskRunRollback(
                          create(PreviewTaskRunRollbackRequestSchema, {
                            name: preview.taskRun.name,
                          }),
                          {
                            contextValues: createContextValues().set(
                              silentContextKey,
                              true
                            ),
                          }
                        );
                      preview.statement = response.statement;
                    } catch (err) {
                      preview.error = String(err);
                    }
                  }
                  setPreviews(nextPreviews);
                  setStep(2);
                } finally {
                  setLoading(false);
                }
              }}
            >
              {t("common.next")}
            </Button>
          ) : null}
          <Button
            disabled={step !== 2 || !canCreate}
            onClick={async () => {
              if (previews.length === 0) return;
              try {
                setLoading(true);
                const specs = [];
                for (const preview of previews) {
                  if (!preview.statement) continue;
                  const sheet = await sheetStore.createSheet(
                    projectName,
                    create(SheetSchema, {
                      name: `${projectName}/sheets/${uuidv4()}`,
                      content: new TextEncoder().encode(preview.statement),
                    })
                  );
                  specs.push(
                    create(Plan_SpecSchema, {
                      id: uuidv4(),
                      config: {
                        case: "changeDatabaseConfig",
                        value: create(Plan_ChangeDatabaseConfigSchema, {
                          targets: [preview.task.target],
                          sheet: sheet.name,
                        }),
                      },
                    })
                  );
                }
                const plan = create(PlanSchema, {
                  name: `${projectName}/plans/${uuidv4()}`,
                  title: `Rollback for rollout#${extractPlanUIDFromRolloutName(
                    rolloutName
                  )}`,
                  description: `This plan is created to rollback ${previews.length} task(s) in rollout #${extractPlanUIDFromRolloutName(rolloutName)}`,
                  specs,
                });
                const createdPlan = await planServiceClientConnect.createPlan(
                  create(CreatePlanRequestSchema, {
                    parent: projectName,
                    plan,
                  })
                );
                void router.push({
                  name: PROJECT_V1_ROUTE_PLAN_DETAIL,
                  params: {
                    projectId: extractProjectResourceName(projectName),
                    planId: extractPlanUID(createdPlan.name),
                  },
                });
              } catch (err) {
                pushNotification({
                  module: "bytebase",
                  style: "CRITICAL",
                  title: t("common.failed"),
                  description: String(err),
                });
              } finally {
                setLoading(false);
              }
            }}
          >
            {loading && <Loader2 className="h-4 w-4 animate-spin" />}
            {t("common.confirm")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
