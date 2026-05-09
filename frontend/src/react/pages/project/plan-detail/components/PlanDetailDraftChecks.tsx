import { create } from "@bufbuild/protobuf";
import {
  AlertCircle,
  CheckCircle,
  CircleQuestionMark,
  FileCode,
  Play,
  SearchCode,
  XCircle,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { releaseServiceClientConnect } from "@/connect";
import { Badge } from "@/react/components/ui/badge";
import { Button } from "@/react/components/ui/button";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/react/components/ui/sheet";
import { Tooltip } from "@/react/components/ui/tooltip";
import { pushNotification } from "@/store";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import {
  type PlanCheckRun,
  type PlanCheckRun_Result,
  PlanCheckRun_Result_Type,
} from "@/types/proto-es/v1/plan_service_pb";
import {
  CheckReleaseRequestSchema,
  type CheckReleaseResponse_CheckResult,
  Release_Type,
} from "@/types/proto-es/v1/release_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import { extractProjectResourceName } from "@/utils";
import { getSheetStatement } from "@/utils/v1/sheet";
import { usePlanDetailContext } from "../context/PlanDetailContext";
import { getLocalSheetByName } from "../utils/localSheet";
import {
  getFilteredResultGroups,
  getPlanCheckSummary,
  transformReleaseCheckResultsToPlanCheckRuns,
} from "../utils/planCheck";

export function PlanDetailDraftChecks({
  checkResults,
  onCheckResultsChange,
  selectedSpec,
}: {
  checkResults?: CheckReleaseResponse_CheckResult[];
  onCheckResultsChange: (
    results: CheckReleaseResponse_CheckResult[] | undefined
  ) => void;
  selectedSpec: Plan_Spec;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const [isRunningChecks, setIsRunningChecks] = useState(false);
  const [selectedResultStatus, setSelectedResultStatus] = useState<
    Advice_Level | undefined
  >();

  const statement = useMemo(() => {
    if (selectedSpec.config.case !== "changeDatabaseConfig") return "";
    const sheet = getLocalSheetByName(selectedSpec.config.value.sheet);
    return getSheetStatement(sheet);
  }, [selectedSpec]);

  const formattedCheckRuns = useMemo(
    () => transformReleaseCheckResultsToPlanCheckRuns(checkResults ?? []),
    [checkResults]
  );
  const summary = useMemo(
    () => getPlanCheckSummary(formattedCheckRuns),
    [formattedCheckRuns]
  );
  const affectedRows = useMemo(() => {
    return (checkResults ?? []).reduce(
      (acc, result) => acc + result.affectedRows,
      0n
    );
  }, [checkResults]);

  const runChecks = async () => {
    if (selectedSpec.config.case !== "changeDatabaseConfig") return;
    setIsRunningChecks(true);
    try {
      const response = await releaseServiceClientConnect.checkRelease(
        create(CheckReleaseRequestSchema, {
          parent: `projects/${extractProjectResourceName(page.plan.name)}`,
          release: {
            type: Release_Type.VERSIONED,
            files: [
              {
                version: "0",
                statement: new TextEncoder().encode(statement),
              },
            ],
          },
          targets: selectedSpec.config.value.targets ?? [],
        })
      );
      onCheckResultsChange(response.results || []);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("plan.checks.started"),
      });
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("plan.checks.failed-to-run"),
        description: String(error),
      });
    } finally {
      setIsRunningChecks(false);
    }
  };

  return (
    <div className="flex flex-col gap-y-1">
      <div className="flex items-center justify-between gap-2">
        <h3 className="textlabel uppercase">{t("plan.checks.self")}</h3>
        <Button
          disabled={statement.length === 0 || isRunningChecks}
          onClick={() => void runChecks()}
          size="xs"
          variant="outline"
        >
          <Play className="h-3.5 w-3.5" />
          {t("plan.run")}
        </Button>
      </div>

      <div className="flex min-w-0 flex-wrap items-center gap-3 text-sm">
        {summary.error > 0 && (
          <button
            className="flex items-center gap-1 text-error hover:opacity-80"
            onClick={() => setSelectedResultStatus(Advice_Level.ERROR)}
            type="button"
          >
            <XCircle className="h-4 w-4 text-error" />
            <span>{t("common.error")}</span>
            <span className="font-medium">{summary.error}</span>
          </button>
        )}
        {summary.warning > 0 && (
          <button
            className="flex items-center gap-1 text-warning hover:opacity-80"
            onClick={() => setSelectedResultStatus(Advice_Level.WARNING)}
            type="button"
          >
            <AlertCircle className="h-4 w-4 text-warning" />
            <span>{t("common.warning")}</span>
            <span className="font-medium">{summary.warning}</span>
          </button>
        )}
        {!checkResults ? null : summary.error === 0 && summary.warning === 0 ? (
          <span className="flex items-center gap-1 text-success">
            <CheckCircle className="h-4 w-4 text-success" />
            <span>{t("common.success")}</span>
          </span>
        ) : null}
        {checkResults && affectedRows > 0 && (
          <Tooltip content={t("task.check-type.affected-rows.description")}>
            <span className="flex items-center gap-1 text-sm text-control-light">
              <span>{t("task.check-type.affected-rows.self")}</span>
              <span>{String(affectedRows)}</span>
              <CircleQuestionMark className="h-3.5 w-3.5 opacity-80" />
            </span>
          </Tooltip>
        )}
      </div>

      <DraftChecksDialog
        onClose={() => setSelectedResultStatus(undefined)}
        open={selectedResultStatus !== undefined}
        planCheckRuns={formattedCheckRuns}
        status={selectedResultStatus}
      />
    </div>
  );
}

function DraftChecksDialog({
  onClose,
  open,
  planCheckRuns,
  status,
}: {
  onClose: () => void;
  open: boolean;
  planCheckRuns: PlanCheckRun[];
  status?: Advice_Level;
}) {
  const { t } = useTranslation();
  const [selectedStatus, setSelectedStatus] = useState<
    Advice_Level | undefined
  >(status);
  const summary = useMemo(
    () => getPlanCheckSummary(planCheckRuns),
    [planCheckRuns]
  );

  useEffect(() => {
    if (!open) return;
    setSelectedStatus(status);
  }, [open, status]);

  const filteredResultGroups = useMemo(
    () => getFilteredResultGroups({ planCheckRuns, selectedStatus }),
    [planCheckRuns, selectedStatus]
  );

  return (
    <Sheet
      onOpenChange={(nextOpen) => {
        if (!nextOpen) onClose();
      }}
      open={open}
    >
      <SheetContent width="wide">
        <SheetHeader>
          <SheetTitle>{t("plan.navigator.checks")}</SheetTitle>
        </SheetHeader>
        <SheetBody className="gap-y-4">
          {summary.total > 0 && (
            <div className="flex min-w-0 flex-wrap items-center gap-3 text-sm">
              {summary.error > 0 && (
                <button
                  className={
                    selectedStatus === Advice_Level.ERROR
                      ? "flex items-center gap-1 rounded-lg bg-gray-100 px-2 py-1 text-error"
                      : "flex items-center gap-1 text-error"
                  }
                  onClick={() =>
                    setSelectedStatus((current) =>
                      current === Advice_Level.ERROR
                        ? undefined
                        : Advice_Level.ERROR
                    )
                  }
                  type="button"
                >
                  <XCircle className="h-5 w-5" />
                  <span>{t("common.error")}</span>
                  <span>{summary.error}</span>
                </button>
              )}
              {summary.warning > 0 && (
                <button
                  className={
                    selectedStatus === Advice_Level.WARNING
                      ? "flex items-center gap-1 rounded-lg bg-gray-100 px-2 py-1 text-warning"
                      : "flex items-center gap-1 text-warning"
                  }
                  onClick={() =>
                    setSelectedStatus((current) =>
                      current === Advice_Level.WARNING
                        ? undefined
                        : Advice_Level.WARNING
                    )
                  }
                  type="button"
                >
                  <AlertCircle className="h-5 w-5" />
                  <span>{t("common.warning")}</span>
                  <span>{summary.warning}</span>
                </button>
              )}
              {summary.success > 0 && (
                <button
                  className={
                    selectedStatus === Advice_Level.SUCCESS
                      ? "flex items-center gap-1 rounded-lg bg-gray-100 px-2 py-1 text-success"
                      : "flex items-center gap-1 text-success"
                  }
                  onClick={() =>
                    setSelectedStatus((current) =>
                      current === Advice_Level.SUCCESS
                        ? undefined
                        : Advice_Level.SUCCESS
                    )
                  }
                  type="button"
                >
                  <CheckCircle className="h-5 w-5" />
                  <span>{t("common.success")}</span>
                  <span>{summary.success}</span>
                </button>
              )}
            </div>
          )}
          {filteredResultGroups.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-control-light">
              <CheckCircle className="mb-4 h-12 w-12 opacity-50" />
              <div className="text-lg">{t("plan.checks.no-check-results")}</div>
            </div>
          ) : (
            <div className="divide-y">
              {filteredResultGroups.map((group) => (
                <div key={group.key} className="px-2 py-4">
                  <div className="mb-2 flex flex-wrap items-start justify-between gap-2">
                    <div className="flex items-center gap-3">
                      {group.type ===
                      PlanCheckRun_Result_Type.STATEMENT_ADVISE ? (
                        <SearchCode className="h-5 w-5 text-control-light" />
                      ) : (
                        <FileCode className="h-5 w-5 text-control-light" />
                      )}
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-medium">
                          {group.type ===
                          PlanCheckRun_Result_Type.STATEMENT_ADVISE
                            ? t("task.check-type.sql-review.self")
                            : t("task.check-type.affected-rows.self")}
                        </span>
                        {group.type ===
                          PlanCheckRun_Result_Type.STATEMENT_SUMMARY_REPORT && (
                          <Tooltip
                            content={t(
                              "task.check-type.affected-rows.description"
                            )}
                          >
                            <CircleQuestionMark className="h-4 w-4 text-control-light" />
                          </Tooltip>
                        )}
                        {group.createTime && (
                          <span className="text-xs text-control-light">
                            {group.createTime.seconds
                              ? new Date(
                                  Number(group.createTime.seconds) * 1000
                                ).toLocaleString()
                              : ""}
                          </span>
                        )}
                      </div>
                    </div>
                    <div className="min-w-0 max-w-[50%] text-sm text-control-light">
                      <span className="truncate">{group.target}</span>
                    </div>
                  </div>
                  <div className="flex flex-col gap-y-2 pl-8">
                    {group.results.map((result, index) => (
                      <DraftCheckResultCard
                        key={`${group.key}-${index}`}
                        result={result}
                      />
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </SheetBody>
      </SheetContent>
    </Sheet>
  );
}

function DraftCheckResultCard({ result }: { result: PlanCheckRun_Result }) {
  const { t } = useTranslation();
  const position =
    result.report?.case === "sqlReviewReport"
      ? result.report.value.startPosition
      : undefined;
  const affectedRows =
    result.report?.case === "sqlSummaryReport"
      ? result.report.value.affectedRows
      : undefined;

  return (
    <div className="flex items-start gap-3 rounded-lg border border-control-border bg-control-bg px-3 py-2">
      <div className="mt-0.5 shrink-0">
        {result.status === Advice_Level.ERROR ? (
          <XCircle className="h-5 w-5 text-error" />
        ) : result.status === Advice_Level.WARNING ? (
          <AlertCircle className="h-5 w-5 text-warning" />
        ) : (
          <CheckCircle className="h-5 w-5 text-success" />
        )}
      </div>
      <div className="flex min-w-0 flex-1 flex-col gap-y-1">
        <div className="text-sm font-medium text-main">{result.title}</div>
        {result.content && (
          <div className="text-sm text-control">{result.content}</div>
        )}
        {position && (position.line > 0 || position.column > 0) && (
          <div className="mt-1 text-sm text-control-light">
            {position.line > 0 ? `${t("common.line")} ${position.line}` : ""}
            {position.line > 0 && position.column > 0 ? ", " : ""}
            {position.column > 0
              ? `${t("common.column")} ${position.column}`
              : ""}
          </div>
        )}
        {affectedRows !== undefined && (
          <div className="mt-1 flex items-center gap-1 text-sm">
            <Badge className="px-2 py-0 text-xs font-medium" variant="default">
              {t("task.check-type.affected-rows.self")}
            </Badge>
            <span>{String(affectedRows)}</span>
            <span className="text-control opacity-80">
              ({t("task.check-type.affected-rows.description")})
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
