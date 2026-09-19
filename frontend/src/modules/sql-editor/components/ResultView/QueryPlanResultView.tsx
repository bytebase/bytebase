import { LoaderCircle } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { QueryPlanView } from "@/apps/explain-visualizer/QueryPlanView";
import { formatPlanSource } from "@/apps/explain-visualizer/QueryPlanViewer";
import { Alert } from "@/components/ui/alert";
import { CopyButton } from "@/components/ui/copy-button";
import { Tabs, TabsList, TabsPanel, TabsTrigger } from "@/components/ui/tabs";
import type { VisualizerEngine } from "@/utils/explainToken";

export interface InlineQueryPlan {
  source: string;
  statement: string;
}

type LoadPlan = () => Promise<InlineQueryPlan | undefined>;

interface Props {
  rawPlan: string;
  initialPlan?: InlineQueryPlan;
  engine?: VisualizerEngine;
  loadPlan?: LoadPlan;
}

interface LoadRequest {
  loader: LoadPlan;
  rawPlan: string;
  promise: ReturnType<LoadPlan>;
}

function TextPlan({ source }: { source: string }) {
  const formatted = useMemo(() => formatPlanSource(source), [source]);
  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex shrink-0 justify-end px-4 py-2">
        <CopyButton content={formatted} size="sm" appearance="outline" />
      </div>
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
}: Props) {
  const { t } = useTranslation();
  const [plan, setPlan] = useState(initialPlan);
  const [loading, setLoading] = useState(
    engine !== undefined && !initialPlan && !!loadPlan
  );
  const [failed, setFailed] = useState(false);
  const loadRequest = useRef<LoadRequest | undefined>(undefined);

  useEffect(() => {
    let cancelled = false;
    setPlan(initialPlan);
    setFailed(false);
    if (initialPlan || !engine || !loadPlan) {
      setLoading(false);
      return;
    }

    setLoading(true);
    if (
      loadRequest.current?.loader !== loadPlan ||
      loadRequest.current.rawPlan !== rawPlan
    ) {
      loadRequest.current = {
        loader: loadPlan,
        rawPlan,
        promise: loadPlan(),
      };
    }
    void loadRequest.current.promise
      .then((loaded) => {
        if (cancelled) return;
        if (loaded) setPlan(loaded);
        else setFailed(true);
      })
      .catch(() => {
        if (!cancelled) setFailed(true);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [engine, initialPlan, loadPlan, rawPlan]);

  if (plan && engine) {
    return (
      <QueryPlanView
        engine={engine}
        planSource={plan.source}
        textPlanSource={rawPlan}
        textTabLabel={t("sql-editor.query-plan-text")}
        planQuery={plan.statement}
      />
    );
  }

  if (loading) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center gap-2 text-sm text-control-light">
        <LoaderCircle className="size-4 animate-spin" />
        {t("common.loading")}
      </div>
    );
  }

  if (failed) {
    return (
      <div className="flex min-h-0 flex-1 flex-col gap-2 overflow-hidden p-4">
        <Alert variant="error" title={t("sql-editor.query-plan-load-failed")} />
        <TextPlan source={rawPlan} />
      </div>
    );
  }

  return (
    <Tabs value="plan" className="flex min-h-0 flex-1 flex-col">
      <TabsList className="shrink-0 px-2 pt-2">
        <TabsTrigger value="plan">{t("common.plan")}</TabsTrigger>
      </TabsList>

      <TabsPanel
        value="plan"
        className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden"
      >
        <TextPlan source={rawPlan} />
      </TabsPanel>
    </Tabs>
  );
}
