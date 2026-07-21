import { ChevronDownIcon, ChevronUpIcon } from "lucide-react";
import { useMemo, useState } from "react";
import type { SpannerChildLink, SpannerPlanNodeData } from "./spanner-types";

interface Props {
  node: SpannerPlanNodeData;
  allNodes: SpannerPlanNodeData[];
  depth: number;
}

const formatValue = (value: unknown): string => {
  if (value === null || value === undefined) return "null";
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
};

const kindClassName = (kind: string): string => {
  switch (kind) {
    case "RELATIONAL":
      return "ev-spanner-kind-relational";
    case "SCALAR":
      return "ev-spanner-kind-scalar";
    default:
      return "ev-spanner-kind-unknown";
  }
};

export function SpannerPlanNode({ node, allNodes, depth }: Props) {
  const [expanded, setExpanded] = useState(true);

  const relationalChildLinks = useMemo<SpannerChildLink[]>(() => {
    if (!node.childLinks) return [];
    return node.childLinks.filter((link) => {
      const childNode = allNodes.find((n) => n.index === link.childIndex);
      return childNode && childNode.kind === "RELATIONAL";
    });
  }, [node.childLinks, allNodes]);

  const hasChildren = relationalChildLinks.length > 0;
  const isScalar = node.kind === "SCALAR";
  const shortDescription = node.shortRepresentation?.description;

  const displayMetadata = useMemo<Record<string, unknown>>(() => {
    if (!node.metadata) return {};
    // Filter out internal metadata keys (prefixed with `_`).
    const filtered: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(node.metadata)) {
      if (!key.startsWith("_")) filtered[key] = value;
    }
    return filtered;
  }, [node.metadata]);

  const hasMetadata = Object.keys(displayMetadata).length > 0;

  const handleToggle = () => {
    if (hasChildren) setExpanded((v) => !v);
  };

  const getChildNode = (index: number): SpannerPlanNodeData => {
    const child = allNodes.find((n) => n.index === index);
    return (
      child ?? {
        index,
        kind: "UNKNOWN",
        displayName: `Unknown Node (${index})`,
      }
    );
  };

  return (
    <div
      className={`ev-spanner-node${isScalar ? " is-scalar" : ""}`}
      data-depth={depth}
    >
      <div className="ev-spanner-row" onClick={handleToggle}>
        {hasChildren ? (
          <span className="ev-spanner-toggle">
            {expanded ? (
              <ChevronDownIcon className="w-4" />
            ) : (
              <ChevronUpIcon className="w-4" />
            )}
          </span>
        ) : (
          <span className="ev-spanner-toggle-placeholder" />
        )}

        <span className={`ev-spanner-kind ${kindClassName(node.kind)}`}>
          {node.kind}
        </span>
        <span className="ev-spanner-name">{node.displayName}</span>

        {shortDescription ? (
          <span className="ev-spanner-desc">{shortDescription}</span>
        ) : null}
      </div>

      {hasMetadata ? (
        <div className="ev-spanner-metadata">
          {Object.entries(displayMetadata).map(([key, value]) => (
            <div key={key} className="ev-spanner-metadata-item">
              <span className="ev-spanner-metadata-key">{key}:</span>
              <span className="ev-spanner-metadata-value">
                {formatValue(value)}
              </span>
            </div>
          ))}
        </div>
      ) : null}

      {expanded && hasChildren ? (
        <div className="ev-spanner-children">
          {relationalChildLinks.map((childLink) => (
            <div key={childLink.childIndex} className="ev-spanner-child">
              {childLink.type ? (
                <div className="ev-spanner-child-type">{childLink.type}</div>
              ) : null}
              <SpannerPlanNode
                node={getChildNode(childLink.childIndex)}
                allNodes={allNodes}
                depth={depth + 1}
              />
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}
