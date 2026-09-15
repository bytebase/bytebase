import { Alert } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import { formatPlanCost, formatPlanCount, type PlanNode } from "./plan-model";
import { PlanMetric } from "./plan-shared";

interface Props {
  readonly node: PlanNode | undefined;
}

/** The node's estimates as labelled values, leaving out any it does not have. */
function nodeMetrics(node: PlanNode): { label: string; value: string }[] {
  return [
    { label: "Startup cost", value: node.startupCost, format: formatPlanCost },
    { label: "Total cost", value: node.totalCost, format: formatPlanCost },
    { label: "Added cost", value: node.selfCost, format: formatPlanCost },
    { label: "Estimated rows", value: node.rows, format: formatPlanCount },
    {
      label: "Row width",
      value: node.width,
      format: (width: number) => `${formatPlanCount(width)} bytes`,
    },
  ].flatMap(({ label, value, format }) =>
    value === undefined ? [] : [{ label, value: format(value) }]
  );
}

export function QueryPlanNodeDetails({ node }: Props) {
  if (!node) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <p className="text-sm leading-5 text-control-light">
          Select a node to see its details.
        </p>
      </div>
    );
  }

  const metrics = nodeMetrics(node);

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

      {/* The diagram flags these with a hover-only icon, which a touch reader
          never reaches. The pane is where the selected node is explained, so
          it carries the warnings too. */}
      {node.warnings.map((warning, index) => (
        <Alert
          // An engine can report the same warning twice, so its text is not a
          // key; a node's warnings never reorder.
          key={index}
          variant="warning"
          data-testid="plan-details-warning"
          title={warning.title}
          description={warning.detail}
        />
      ))}

      {metrics.length > 0 ? (
        <>
          <Separator />
          <dl className="flex flex-wrap gap-4">
            {metrics.map((metric) => (
              <PlanMetric
                key={metric.label}
                label={metric.label}
                value={metric.value}
              />
            ))}
          </dl>
        </>
      ) : null}

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
