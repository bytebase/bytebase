import { Alert } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import {
  formatPlanCost,
  formatPlanCount,
  isFlaggedFullScan,
  PLAN_FULL_SCAN_HINT,
  type PlanNode,
} from "./plan-model";

interface Props {
  readonly node: PlanNode | undefined;
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex min-w-32 flex-1 flex-col gap-1">
      <dt className="text-xs leading-4 text-control-light">{label}</dt>
      <dd className="text-sm leading-5 text-main tabular-nums">{value}</dd>
    </div>
  );
}

export function QueryPlanNodeDetails({ node }: Props) {
  if (!node) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <p className="text-sm leading-5 text-control-light">
          Select a node to see its estimates.
        </p>
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col overflow-auto p-4 gap-4">
      <div className="flex flex-col gap-1">
        <h2 className="text-base leading-6 font-semibold text-main">
          {node.nodeType}
        </h2>
        {node.subject ? (
          <p className="text-sm leading-5 break-words text-control">
            {node.subject}
          </p>
        ) : null}
        {node.relationship ? (
          <p className="text-xs leading-4 text-control-light">
            {node.relationship}
          </p>
        ) : null}
      </div>

      {/* The diagram flags this with a hover-only icon, which a touch reader
          never reaches. The pane is where the selected node is explained, so
          it carries the warning too. */}
      {isFlaggedFullScan(node) ? (
        <Alert
          variant="warning"
          data-testid="plan-details-full-scan"
          title="Full table scan"
          description={PLAN_FULL_SCAN_HINT}
        />
      ) : null}

      <Separator />

      <dl className="flex flex-wrap gap-4">
        <Metric label="Startup cost" value={formatPlanCost(node.startupCost)} />
        <Metric label="Total cost" value={formatPlanCost(node.totalCost)} />
        <Metric label="Self cost" value={formatPlanCost(node.selfCost)} />
        <Metric label="Estimated rows" value={formatPlanCount(node.rows)} />
        <Metric
          label="Row width"
          value={`${formatPlanCount(node.width)} bytes`}
        />
      </dl>

      {node.properties.length > 0 ? (
        <>
          <Separator />
          <dl className="flex flex-col gap-3">
            {node.properties.map((property) => (
              <div key={property.label} className="flex flex-col gap-1">
                <dt className="text-xs leading-4 text-control-light">
                  {property.label}
                </dt>
                <dd className="font-mono text-xs leading-4 break-words whitespace-pre-wrap text-main">
                  {property.value || "—"}
                </dd>
              </div>
            ))}
          </dl>
        </>
      ) : null}
    </div>
  );
}
