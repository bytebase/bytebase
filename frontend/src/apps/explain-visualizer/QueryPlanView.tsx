import { useMemo } from "react";
import { Alert } from "@/components/ui/alert";
import { cn } from "@/lib/utils";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { VisualizerEngine } from "@/utils/explainToken";
import {
  MSSQL_PLAN_EMPTY_MESSAGE,
  MSSQL_PLAN_INVALID_XML_MESSAGE,
  MSSQL_PLAN_NO_PLAN_MESSAGE,
  parseMssqlPlan,
} from "./mssql-plan";
import {
  QueryPlanI18nProvider,
  type QueryPlanTranslate,
  type QueryPlanTranslationKey,
  useQueryPlanTranslation,
} from "./plan-i18n";
import { PLAN_TOO_DEEP_MESSAGE, type PlanParseResult } from "./plan-model";
import {
  POSTGRES_PLAN_EMPTY_MESSAGE,
  POSTGRES_PLAN_INVALID_JSON_MESSAGE,
  POSTGRES_PLAN_NO_PLAN_MESSAGE,
  parsePostgresPlan,
} from "./postgres-plan";
import { QueryPlanViewer } from "./QueryPlanViewer";
import {
  parseSpannerPlan,
  SPANNER_PLAN_EMPTY_MESSAGE,
  SPANNER_PLAN_EMULATOR_MESSAGE,
  SPANNER_PLAN_INVALID_JSON_MESSAGE,
  SPANNER_PLAN_NO_PLAN_MESSAGE,
} from "./spanner-plan";

const PARSERS: Record<VisualizerEngine, (source: string) => PlanParseResult> = {
  [Engine.POSTGRES]: parsePostgresPlan,
  [Engine.MSSQL]: parseMssqlPlan,
  [Engine.SPANNER]: parseSpannerPlan,
};

const STATUS_CLASS =
  "flex h-full min-h-0 w-full flex-col gap-4 overflow-auto bg-background p-4";

const PARSE_ERROR_KEYS = new Map<string, QueryPlanTranslationKey>([
  [PLAN_TOO_DEEP_MESSAGE, "error.too-deep"],
  [POSTGRES_PLAN_EMPTY_MESSAGE, "error.postgres-empty"],
  [POSTGRES_PLAN_INVALID_JSON_MESSAGE, "error.postgres-invalid"],
  [POSTGRES_PLAN_NO_PLAN_MESSAGE, "error.postgres-no-plan"],
  [MSSQL_PLAN_EMPTY_MESSAGE, "error.mssql-empty"],
  [MSSQL_PLAN_INVALID_XML_MESSAGE, "error.mssql-invalid"],
  [MSSQL_PLAN_NO_PLAN_MESSAGE, "error.mssql-no-plan"],
  [SPANNER_PLAN_EMPTY_MESSAGE, "error.spanner-empty"],
  [SPANNER_PLAN_INVALID_JSON_MESSAGE, "error.spanner-invalid"],
  [SPANNER_PLAN_NO_PLAN_MESSAGE, "error.spanner-no-plan"],
  [SPANNER_PLAN_EMULATOR_MESSAGE, "error.spanner-emulator"],
]);

interface Props {
  engine: VisualizerEngine;
  planSource: string;
  textPlanSource?: string;
  planQuery?: string;
  translate?: QueryPlanTranslate;
  disallowCopyingData?: boolean;
  syncSelectionWithHash?: boolean;
}

function QueryPlanViewContent({
  engine,
  planSource,
  textPlanSource,
  planQuery,
  disallowCopyingData = false,
  syncSelectionWithHash,
}: Props) {
  const translate = useQueryPlanTranslation();
  const result = useMemo(
    () => PARSERS[engine](planSource),
    [engine, planSource]
  );

  if (!result.ok) {
    const errorKey = PARSE_ERROR_KEYS.get(result.message);
    return (
      <div className={cn(STATUS_CLASS, disallowCopyingData && "select-none")}>
        <Alert
          variant="error"
          title={translate("error.unreadable")}
          description={errorKey ? translate(errorKey) : result.message}
        />
        {planSource.trim() ? (
          <pre className="font-mono text-xs leading-4 break-words whitespace-pre-wrap text-control">
            {planSource}
          </pre>
        ) : null}
      </div>
    );
  }

  return (
    <QueryPlanViewer
      tree={result.tree}
      rawPlan={planSource}
      textPlan={textPlanSource}
      query={planQuery}
      disallowCopyingData={disallowCopyingData}
      syncSelectionWithHash={syncSelectionWithHash}
    />
  );
}

export function QueryPlanView({ translate, ...props }: Props) {
  return (
    <QueryPlanI18nProvider translate={translate}>
      <QueryPlanViewContent {...props} />
    </QueryPlanI18nProvider>
  );
}
