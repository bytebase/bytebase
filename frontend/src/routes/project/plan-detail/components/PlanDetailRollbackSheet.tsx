import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { DatabaseBackup, Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import {
  issueServiceClientConnect,
  planServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/api";
import { silentContextKey } from "@/api/context-key";
import { router } from "@/app/router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL } from "@/app/router/handles";
import { ReadonlyMonaco } from "@/components/monaco";
import {
  PermissionGuard,
  usePermissionCheck,
} from "@/components/PermissionGuard";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useCurrentUser } from "@/hooks/useAppState";
import { useProjectByName } from "@/hooks/useProjectByName";
import { DraftReviewIssueCreationError } from "@/lib/plan/workflow";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import type { Task, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { PreviewTaskRunRollbackRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractPlanUID,
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
} from "@/utils";
import { PlanTargetDisplay } from "./PlanTargetDisplay";
import { createRollbackDraftReview } from "./rollbackDraft";

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
  const currentUser = useCurrentUser();
  // subscribe to re-render on project cache change
  const projectsByName = useAppStore((s) => s.projectsByName);
  void projectsByName;
  const normalizedProjectName = projectName.startsWith("projects/")
    ? projectName
    : `projects/${projectName}`;
  const project = useProjectByName(normalizedProjectName);
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
  const requestGenerationRef = useRef(0);
  const requestControllersRef = useRef(new Set<AbortController>());
  const itemsRef = useRef(items);
  itemsRef.current = items;
  const itemsIdentity = useMemo(
    () =>
      JSON.stringify(
        items.map(({ task, taskRun }) => [task.name, task.target, taskRun.name])
      ),
    [items]
  );

  const invalidateRequests = useCallback(() => {
    requestGenerationRef.current += 1;
    for (const controller of requestControllersRef.current) {
      controller.abort();
    }
    requestControllersRef.current.clear();
  }, []);

  const requestPreviews = useCallback(
    async (targetItems: Array<{ task: Task; taskRun: TaskRun }>) => {
      invalidateRequests();
      const generation = requestGenerationRef.current;
      const controller = new AbortController();
      requestControllersRef.current.add(controller);
      setLoading(true);
      const nextPreviews = targetItems.map((item) => ({
        task: item.task,
        taskRun: item.taskRun,
        statement: undefined as string | undefined,
        error: undefined as string | undefined,
      }));
      try {
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
                  signal: controller.signal,
                }
              );
            if (requestGenerationRef.current !== generation) return;
            preview.statement = response.statement;
          } catch (error) {
            if (
              controller.signal.aborted ||
              requestGenerationRef.current !== generation
            ) {
              return;
            }
            preview.error = String(error);
          }
        }
        if (requestGenerationRef.current !== generation) return;
        setPreviews(nextPreviews);
        setStep(2);
      } finally {
        requestControllersRef.current.delete(controller);
        if (requestGenerationRef.current === generation) {
          setLoading(false);
        }
      }
    },
    [invalidateRequests]
  );

  const handleOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (!nextOpen) {
        invalidateRequests();
        setLoading(false);
      }
      onOpenChange(nextOpen);
    },
    [invalidateRequests, onOpenChange]
  );

  useEffect(() => {
    invalidateRequests();
    setLoading(false);
    setStep(1);
    setPreviews([]);
    setSelectedTaskRunNames([]);
    if (!open) return;

    const currentItems = itemsRef.current;
    if (currentItems.length === 1) {
      setSelectedTaskRunNames([currentItems[0].taskRun.name]);
      void requestPreviews(currentItems);
    }
    return invalidateRequests;
  }, [invalidateRequests, itemsIdentity, open, requestPreviews]);

  const selectedItems = useMemo(
    () =>
      items.filter((item) => selectedTaskRunNames.includes(item.taskRun.name)),
    [items, selectedTaskRunNames]
  );

  const [canCreateDraftReview, createPermissionReason] = usePermissionCheck(
    ["bb.plans.create", "bb.issues.create"],
    project
  );
  const [canUpdateIssue] = usePermissionCheck(["bb.issues.update"], project);

  const canCreate = useMemo(() => {
    return (
      !loading &&
      previews.length > 0 &&
      previews.every((preview) => !preview.error) &&
      previews.some((preview) => preview.statement) &&
      canCreateDraftReview
    );
  }, [canCreateDraftReview, loading, previews]);

  return (
    <Sheet onOpenChange={handleOpenChange} open={open}>
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
                      className="flex cursor-pointer items-center gap-3 border-b border-control-border px-3 py-2 last:border-b-0 hover:bg-control-bg"
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
                          <PlanTargetDisplay
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
              {!canUpdateIssue && (
                <Alert
                  variant="warning"
                  description={t("plan.draft-update-permission-required")}
                />
              )}
              {loading ? (
                <div className="flex items-center justify-center py-8 text-control-light">
                  <Loader2 className="size-5 animate-spin" />
                </div>
              ) : (
                previews.map((preview) => (
                  <div key={preview.taskRun.name} className="space-y-2">
                    <div className="text-sm font-medium text-main">
                      <PlanTargetDisplay
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
                        className="relative rounded-md border border-control-border"
                        content={preview.statement}
                        language="sql"
                        min={128}
                        max={256}
                      />
                    ) : (
                      <div className="flex items-center justify-center rounded-md border border-control-border bg-control-bg p-8">
                        <div className="flex flex-col items-center gap-y-2 text-center">
                          <DatabaseBackup className="size-6 text-control-placeholder" />
                          <p className="text-sm text-control-light">
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
            <Button
              onClick={() => handleOpenChange(false)}
              appearance="secondary"
            >
              {t("common.cancel")}
            </Button>
          ) : (
            <Button
              onClick={() => {
                invalidateRequests();
                setLoading(false);
                setStep(1);
              }}
              appearance="secondary"
            >
              {t("common.back")}
            </Button>
          )}
          {step === 1 ? (
            <Button
              disabled={selectedItems.length === 0}
              onClick={() => void requestPreviews(selectedItems)}
            >
              {t("common.next")}
            </Button>
          ) : null}
          <PermissionGuard
            permissions={["bb.plans.create", "bb.issues.create"]}
            project={project}
          >
            <Button
              disabled={step !== 2 || !canCreate}
              title={createPermissionReason}
              onClick={async () => {
                const successfulPreviews = previews.flatMap((preview) =>
                  preview.statement
                    ? [
                        {
                          statement: preview.statement,
                          target: preview.task.target,
                        },
                      ]
                    : []
                );
                if (successfulPreviews.length === 0) return;
                const actionGeneration = requestGenerationRef.current;
                try {
                  setLoading(true);
                  const rolloutId = extractPlanUIDFromRolloutName(rolloutName);
                  const { plan: createdPlan } = await createRollbackDraftReview(
                    {
                      createIssue: (request) =>
                        issueServiceClientConnect.createIssue(request),
                      createPlan: (request) =>
                        planServiceClientConnect.createPlan(request),
                      createSheet: (sheet) =>
                        useAppStore
                          .getState()
                          .createSheet(normalizedProjectName, sheet),
                      creator: currentUser.name,
                      newId: uuidv4,
                      parent: normalizedProjectName,
                      previews: successfulPreviews,
                      title: t("plan.rollback.title", { rolloutId }),
                      description: t("plan.rollback.description", {
                        count: successfulPreviews.length,
                        rolloutId,
                      }),
                    }
                  );
                  if (requestGenerationRef.current !== actionGeneration) return;
                  handleOpenChange(false);
                  await router.push({
                    name: PROJECT_V1_ROUTE_PLAN_DETAIL,
                    params: {
                      projectId: extractProjectResourceName(createdPlan.name),
                      planId: extractPlanUID(createdPlan.name),
                    },
                  });
                } catch (err) {
                  if (requestGenerationRef.current !== actionGeneration) return;
                  if (err instanceof DraftReviewIssueCreationError) {
                    handleOpenChange(false);
                    await router.push({
                      name: PROJECT_V1_ROUTE_PLAN_DETAIL,
                      params: {
                        projectId: extractProjectResourceName(err.plan.name),
                        planId: extractPlanUID(err.plan.name),
                      },
                    });
                  }
                  pushNotification({
                    module: "bytebase",
                    style: "CRITICAL",
                    title: t("common.failed"),
                    description: String(
                      err instanceof DraftReviewIssueCreationError
                        ? err.cause
                        : err
                    ),
                  });
                } finally {
                  if (requestGenerationRef.current === actionGeneration) {
                    setLoading(false);
                  }
                }
              }}
            >
              {loading && <Loader2 className="size-4 animate-spin" />}
              {t("common.confirm")}
            </Button>
          </PermissionGuard>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
