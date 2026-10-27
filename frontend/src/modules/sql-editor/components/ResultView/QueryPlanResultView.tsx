import { LoaderCircle } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { QueryPlanTranslate } from "@/apps/explain-visualizer/plan-i18n";
import type { PlanTableLookup } from "@/apps/explain-visualizer/plan-model";
import { QueryPlanView } from "@/apps/explain-visualizer/QueryPlanView";
import { formatPlanSource } from "@/apps/explain-visualizer/QueryPlanViewer";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import { Tabs, TabsList, TabsPanel, TabsTrigger } from "@/components/ui/tabs";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import type { VisualizerEngine } from "@/utils/queryPlanEngines";

export interface InlineQueryPlan {
  source: string;
}

/** Fetches the plan; the signal fires when the reader cancels the load. */
type LoadPlan = (signal: AbortSignal) => Promise<InlineQueryPlan | undefined>;

interface Props {
  rawPlan: string;
  initialPlan?: InlineQueryPlan;
  engine?: VisualizerEngine;
  loadPlan?: LoadPlan;
  // The plan was made on demand for a result that already ran, so it is an
  // estimate of now, not a record of that run.
  estimated?: boolean;
  describeTable?: PlanTableLookup;
  onCancel?: () => void;
  disallowCopyingData?: boolean;
}

interface LoadRequest {
  rawPlan: string;
  promise: ReturnType<LoadPlan>;
  abort: () => void;
}

function TextPlan({
  source,
  disallowCopyingData,
}: {
  source: string;
  disallowCopyingData: boolean;
}) {
  const formatted = useMemo(() => formatPlanSource(source), [source]);
  return (
    <div
      className={cn(
        "flex h-full min-h-0 flex-col",
        disallowCopyingData && "select-none"
      )}
    >
      {disallowCopyingData ? null : (
        <div className="flex shrink-0 justify-end px-4 py-2">
          <CopyButton content={formatted} size="sm" appearance="outline" />
        </div>
      )}
      <pre className="min-h-0 flex-1 overflow-auto px-4 pb-4 font-mono text-xs leading-4 break-words whitespace-pre-wrap text-main">
        {formatted}
      </pre>
    </div>
  );
}

export function QueryPlanResultView({
  rawPlan,
  initialPlan,
  engine,
  loadPlan,
  estimated = false,
  describeTable,
  onCancel,
  disallowCopyingData = false,
}: Props) {
  const { t } = useTranslation();
  const [plan, setPlan] = useState(initialPlan);
  const [loading, setLoading] = useState(!initialPlan && !!loadPlan);
  // null while nothing failed; the engine's message, or "" when there is none.
  const [failure, setFailure] = useState<string | null>(null);
  const loadRequest = useRef<LoadRequest | undefined>(undefined);
  // The loader's identity changes with unrelated state (e.g. language); only
  // the plan inputs decide whether to load again.
  const loadPlanRef = useRef(loadPlan);
  loadPlanRef.current = loadPlan;
  const canLoad = !!loadPlan;
  const translatePlan = useCallback<QueryPlanTranslate>(
    (key, values) => t(`sql-editor.query-plan-viewer.${key}`, values),
    [t]
  );

  useEffect(() => {
    let cancelled = false;
    setPlan(initialPlan);
    setFailure(null);
    if (initialPlan || !canLoad) {
      setLoading(false);
      return;
    }

    setLoading(true);
    if (loadRequest.current?.rawPlan !== rawPlan) {
      const controller = new AbortController();
      loadRequest.current = {
        rawPlan,
        promise: loadPlanRef.current!(controller.signal),
        abort: () => controller.abort(),
      };
    }
    void loadRequest.current.promise
      .then((loaded) => {
        if (cancelled) return;
        if (loaded) setPlan(loaded);
        else setFailure("");
      })
      .catch((error: unknown) => {
        if (cancelled) return;
        setFailure(error instanceof Error ? error.message : String(error));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [initialPlan, canLoad, rawPlan]);

  const cancel = () => {
    loadRequest.current?.abort();
    loadRequest.current = undefined;
    onCancel?.();
  };

  // One word in the tab row; the reason it matters is a hover away.
  const estimateNote = estimated ? (
    <Tooltip content={t("sql-editor.query-plan-estimated")}>
      <Badge variant="default">{t("sql-editor.plan-estimated")}</Badge>
    </Tooltip>
  ) : undefined;

  if (plan && engine) {
    return (
      <QueryPlanView
        engine={engine}
        planSource={plan.source}
        textPlanSource={rawPlan || undefined}
        note={estimateNote}
        describeTable={describeTable}
        translate={translatePlan}
        disallowCopyingData={disallowCopyingData}
      />
    );
  }

  if (loading) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center gap-2 text-sm text-control-light">
        <LoaderCircle className="size-4 animate-spin" />
        {t("common.loading")}
        {onCancel ? (
          <Button size="sm" appearance="secondary" onClick={cancel}>
            {t("common.cancel")}
          </Button>
        ) : null}
      </div>
    );
  }

  if (failure !== null) {
    return (
      <div className="flex min-h-0 flex-1 flex-col gap-2 overflow-hidden p-4">
        <Alert
          variant="error"
          title={t("sql-editor.query-plan-load-failed")}
          description={failure || undefined}
        />
        {rawPlan ? (
          <TextPlan
            source={rawPlan}
            disallowCopyingData={disallowCopyingData}
          />
        ) : null}
      </div>
    );
  }

  return (
    <Tabs value="plan" className="flex min-h-0 flex-1 flex-col">
      <TabsList className="shrink-0 px-2 pt-2">
        <TabsTrigger value="plan">{t("common.plan")}</TabsTrigger>
        {estimateNote ? (
          <span className="ml-auto self-center text-sm text-control-light">
            {estimateNote}
          </span>
        ) : null}
      </TabsList>

      <TabsPanel
        value="plan"
        className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <TextPlan
          source={plan?.source ?? rawPlan}
          disallowCopyingData={disallowCopyingData}
        />
      </TabsPanel>
    </Tabs>
  );
}
