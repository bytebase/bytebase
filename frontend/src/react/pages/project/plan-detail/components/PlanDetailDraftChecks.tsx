import { create } from "@bufbuild/protobuf";
import { CircleQuestionMark } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { releaseServiceClientConnect } from "@/connect";
import { PlanCheckSection } from "@/react/components/plan-check/PlanCheckSection";
import { Tooltip } from "@/react/components/ui/tooltip";
import { pushNotification } from "@/store";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import {
  CheckReleaseRequestSchema,
  type CheckReleaseResponse_CheckResult,
  Release_Type,
} from "@/types/proto-es/v1/release_service_pb";
import { extractProjectResourceName } from "@/utils";
import { getSheetStatement } from "@/utils/v1/sheet";
import { usePlanDetailContext } from "../context/PlanDetailContext";
import { getLocalSheetByName } from "../utils/localSheet";
import { transformReleaseCheckResultsToPlanCheckRuns } from "../utils/planCheck";

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

  const statement = useMemo(() => {
    if (selectedSpec.config.case !== "changeDatabaseConfig") return "";
    const sheet = getLocalSheetByName(selectedSpec.config.value.sheet);
    return getSheetStatement(sheet);
  }, [selectedSpec]);

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
      canRun
      isRunning={isRunningChecks}
      onRun={runChecks}
      planCheckRuns={formattedCheckRuns}
      runDisabled={statement.length === 0}
      trailingSummary={trailingSummary}
    />
  );
}
