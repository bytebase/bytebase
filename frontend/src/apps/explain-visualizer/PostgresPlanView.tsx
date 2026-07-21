import { Plan as pev2 } from "pev2";
import { useEffect, useRef } from "react";
import { type App, createApp } from "vue";

interface Props {
  planSource: string;
  planQuery?: string;
}

// pev2 is a Vue 3 component; no React port exists. Reimplementing the
// Postgres EXPLAIN renderer is out of scope, so we mount pev2 as a
// short-lived Vue subapp inside this React node and unmount on cleanup.
// Vue is already in this entry's bundle transitively via pev2.
export function PostgresPlanView({ planSource, planQuery }: Props) {
  const hostRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!hostRef.current) return;
    const app: App = createApp(pev2, { planSource, planQuery });
    app.mount(hostRef.current);
    return () => {
      app.unmount();
    };
  }, [planSource, planQuery]);

  return <div ref={hostRef} className="qp-root" />;
}
