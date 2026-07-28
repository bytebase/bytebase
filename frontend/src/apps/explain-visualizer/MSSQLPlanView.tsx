import { useEffect, useRef } from "react";
import { loadQueryPlanScript } from "./loadQueryPlanScript";

interface Props {
  planXml: string;
}

export function MSSQLPlanView({ planXml }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cancelled = false;
    const container = containerRef.current;
    if (!container) return;
    void (async () => {
      try {
        await loadQueryPlanScript();
        if (cancelled) return;
        if (window.QP && containerRef.current) {
          window.QP.showPlan(containerRef.current, planXml);
        }
      } catch (error) {
        console.error("Failed to load query plan visualizer:", error);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [planXml]);

  return <div ref={containerRef} className="ev-mssql-container" />;
}
