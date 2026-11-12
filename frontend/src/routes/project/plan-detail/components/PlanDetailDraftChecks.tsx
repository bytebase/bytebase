import { create } from "@bufbuild/protobuf";
import { SHA256 } from "crypto-js";
import { CircleQuestionMark } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { releaseServiceClientConnect } from "@/api";
import { PlanCheckSection } from "@/components/plan-check/PlanCheckSection";
import { Tooltip } from "@/components/ui/tooltip";
import {
  expandSpecTargets,
  transformReleaseCheckResultsToPlanCheckRuns,
} from "@/lib/plan/check";
import { AgentDiagnosticHelpButton } from "@/modules/agent/components/AgentDiagnosticHelpButton";
import { getPlanCheckDiagnosticContext } from "@/modules/agent/logic/planCheckDiagnostics";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import {
  CheckReleaseRequestSchema,
  type CheckReleaseResponse_CheckResult,
  Release_Type,
} from "@/types/proto-es/v1/release_service_pb";
import { extractProjectResourceName } from "@/utils";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import {
  getSpecStatementContent,
  isSameStatementContent,
  useLocalSheetsVersion,
} from "../utils/localSheet";
import { PlanTargetDisplay } from "./PlanTargetDisplay";

export function PlanDetailDraftChecks({
  checkedContent,
  checkResults,
  content: contentOverride,
  canDiagnose,
  onCheckResultsChange,
  selectedSpec,
  title,
}: {
  checkedContent?: Uint8Array;
  checkResults?: CheckReleaseResponse_CheckResult[];
  content?: Uint8Array;
  canDiagnose?: boolean;
  onCheckResultsChange: (
    content: Uint8Array | undefined,
    results: CheckReleaseResponse_CheckResult[] | undefined
  ) => void;
  selectedSpec: Plan_Spec;
  title?: string;
}) {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const [isRunningChecks, setIsRunningChecks] = useState(false);
  const databasesByName = useAppStore((state) => state.databasesByName);
  const dbGroupsByName = useAppStore((state) => state.dbGroupsByName);

  const localSheetsVersion = useLocalSheetsVersion();
  // An editor override is authoritative, including an empty statement.
  const content = useMemo(
    () => contentOverride ?? getSpecStatementContent(selectedSpec),
    [contentOverride, selectedSpec, localSheetsVersion]
  );
  const checkedSql = useMemo(() => {
    if (
      !checkResults?.length ||
      !checkedContent?.length ||
      !isSameStatementContent(checkedContent, content)
    )
      return undefined;
    try {
      // Retain the BOM so the hash identifies the exact checked bytes.
      const text = new TextDecoder("utf-8", {
        fatal: true,
        ignoreBOM: true,
      }).decode(checkedContent);
      return { text, sha256: SHA256(text).toString() };
    } catch {
      return undefined;
    }
  }, [checkResults, checkedContent, content]);
  const checkedTargets = expandSpecTargets(selectedSpec, (target) =>
    dbGroupsByName[target]?.matchedDatabases.map((database) => database.name)
  );

  const formattedCheckRuns = useMemo(
    () => transformReleaseCheckResultsToPlanCheckRuns(checkResults ?? []),
    [checkResults]
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
                statement: content ?? new Uint8Array(),
              },
            ],
          },
          targets: selectedSpec.config.value.targets ?? [],
        })
      );
      onCheckResultsChange(content, response.results || []);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("plan.checks.completed"),
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

  const trailingSummary =
    checkResults && affectedRows > 0n ? (
      <Tooltip content={t("task.check-type.affected-rows.description")}>
        <span className="flex items-center gap-1 text-sm text-control-light">
          <span>{t("task.check-type.affected-rows.self")}</span>
          <span>{String(affectedRows)}</span>
          <CircleQuestionMark className="h-3.5 w-3.5 opacity-80" />
        </span>
      </Tooltip>
    ) : null;

  return (
    <PlanCheckSection
      title={title}
      canRun
      isRunning={isRunningChecks}
      onRun={runChecks}
      planCheckRuns={formattedCheckRuns}
      renderResultAction={(result) => {
        const project = `projects/${page.projectId}`;
        const database = databasesByName[result.target];
        if (
          !(canDiagnose ?? page.isCreating) ||
          !checkedSql ||
          !database ||
          database.name !== result.target ||
          database.project !== project ||
          !checkedTargets.includes(result.target)
        )
          return null;
        const context = getPlanCheckDiagnosticContext({
          resource: {
            kind: "database",
            name: database.name,
            project,
            label: database.name,
          },
          result,
          specId: selectedSpec.id,
          sourceId: `${project}/draftChecks/${selectedSpec.id}/${checkedSql.sha256}`,
          sql: {
            source: "draft",
            ...checkedSql,
            sheet:
              selectedSpec.config.case === "changeDatabaseConfig"
                ? selectedSpec.config.value.sheet
                : undefined,
          },
          facts: [{ label: "Spec", value: selectedSpec.id }],
        });
        return context ? <AgentDiagnosticHelpButton context={context} /> : null;
      }}
      renderTarget={(target) => (
        <PlanTargetDisplay showEnvironment target={target} />
      )}
      runDisabled={(content?.length ?? 0) === 0}
      trailingSummary={trailingSummary}
    />
  );
}
