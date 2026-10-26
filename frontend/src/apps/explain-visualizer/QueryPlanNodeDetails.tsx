import { Alert } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import { type QueryPlanTranslate, useQueryPlanTranslation } from "./plan-i18n";
import {
  formatPlanCost,
  formatPlanCount,
  formatPlanDuration,
  PLAN_ROW_ESTIMATE_NOTABLE_FACTOR,
  type PlanNode,
  planRowEstimateFactor,
} from "./plan-model";
import { PlanMetric } from "./plan-shared";

/**
 * How far the row estimate was out, in words, or undefined when it was close
 * enough not to be worth a line.
 */
function rowEstimateNote(
  node: PlanNode,
  translate: QueryPlanTranslate
): string | undefined {
  const factor = planRowEstimateFactor(node);
  if (factor === undefined) return undefined;
  // A node that returned nothing has no multiple to state, and "∞× fewer" is
  // not a number a reader can do anything with.
  if (factor === 0) return translate("metric.rows-none");
  if (factor >= PLAN_ROW_ESTIMATE_NOTABLE_FACTOR) {
    return translate("metric.rows-over", { factor: formatPlanCount(factor) });
  }
  if (factor <= 1 / PLAN_ROW_ESTIMATE_NOTABLE_FACTOR) {
    return translate("metric.rows-under", {
      factor: formatPlanCount(1 / factor),
    });
  }
  return undefined;
}

interface Props {
  readonly node: PlanNode | undefined;
}

/** The node's estimates as labelled values, leaving out any it does not have. */
function nodeMetrics(
  node: PlanNode,
  translate: QueryPlanTranslate
): { label: string; value: string }[] {
  return [
    {
      label: translate("metric.startup-cost"),
      value: node.startupCost,
      format: formatPlanCost,
    },
    {
      label: translate("metric.total-cost"),
      value: node.totalCost,
      format: formatPlanCost,
    },
    {
      label: translate("metric.added-cost"),
      value: node.selfCost,
      format: formatPlanCost,
    },
    {
      label: translate("metric.estimated-rows"),
      value: node.rows,
      format: formatPlanCount,
    },
    {
      label: translate("metric.actual-rows"),
      value: node.actualRows,
      format: formatPlanCount,
    },
    {
      label: translate("metric.loops"),
      value: node.loops,
      format: formatPlanCount,
    },
    {
      label: translate("metric.actual-time"),
      value: node.actualTime,
      format: (time: number) =>
        translate("metric.milliseconds", { count: formatPlanDuration(time) }),
    },
    {
      label: translate("metric.row-width"),
      value: node.width,
      format: (width: number) =>
        translate("metric.bytes", { count: formatPlanCount(width) }),
    },
  ].flatMap(({ label, value, format }) =>
    value === undefined ? [] : [{ label, value: format(value) }]
  );
}

export function QueryPlanNodeDetails({ node }: Props) {
  const translate = useQueryPlanTranslation();
  if (!node) {
    return (
      <p className="text-sm leading-5 text-control-light">
        {translate("details.select-node")}
      </p>
    );
  }

  const metrics = nodeMetrics(node, translate);
  const rowEstimate = rowEstimateNote(node, translate);

  return (
    <div className="flex flex-col gap-4">
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
          {rowEstimate ? (
            <p
              data-testid="plan-details-row-estimate"
              className="text-xs leading-4 text-control"
            >
              {rowEstimate}
            </p>
          ) : null}
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
