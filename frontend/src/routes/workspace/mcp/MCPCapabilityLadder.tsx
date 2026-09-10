import { Check, ChevronRight, Minus } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { MCPCapabilityRow } from "@/components/mcp/mcpCapabilityRows";
import {
  isRowServed,
  MCP_CAPABILITY_ROWS,
  tierClosedBy,
} from "@/components/mcp/mcpCapabilityRows";
import {
  MCP_MODE_PRESENTATION,
  type MCPMode,
} from "@/components/mcp/mcpPolicy";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";

interface Props {
  readonly mode: MCPMode;
  readonly expanded: boolean;
  readonly details: boolean;
  readonly onExpandedChange: (expanded: boolean) => void;
  readonly onDetailsChange: (details: boolean) => void;
}

/**
 * What a mode allows, as eight capability rows in one ordered list of which
 * every mode is a prefix.
 *
 * The served set comes from the mode's tier alone — Read-only serves the read
 * rows, Read-write both — so there is no comparison against a stored mode and
 * the edit state renders exactly what the view will show after saving. Rows a
 * mode does not serve stay visible and muted, which is what lets an admin
 * compare two modes without opening a second surface.
 *
 * Nothing here is fetched: the tier of each row is static, and which tiers a
 * mode serves is the bundle's copy of `mcpServingClasses`.
 */
export function MCPCapabilityLadder({
  mode,
  expanded,
  details,
  onExpandedChange,
  onDetailsChange,
}: Props) {
  const { t } = useTranslation();
  const modeKey = MCP_MODE_PRESENTATION[mode].key;

  return (
    <Collapsible open={expanded} onOpenChange={onExpandedChange}>
      <div className="flex items-center justify-between gap-x-2 border-b border-block-border py-2">
        <CollapsibleTrigger className="min-w-0 flex-1 rounded-xs">
          <ChevronRight
            className={cn(
              "size-4 shrink-0 text-control-light transition-transform",
              expanded && "rotate-90"
            )}
          />
          {/* Collapsed, the trigger is the mode's description; expanded, it
              becomes the list heading. Neither repeats the mode name the chip
              or the selector already shows — except the heading, whose whole
              job is to say whose list this is. */}
          <span
            className={cn(
              "min-w-0 text-sm",
              expanded ? "font-medium text-main" : "text-control-light"
            )}
          >
            {expanded
              ? t("settings.mcp.ladder.heading", {
                  mode: t(`settings.mcp.policy.mode.${modeKey}.title`),
                })
              : t(`settings.mcp.ladder.summary.${modeKey}`)}
          </span>
        </CollapsibleTrigger>
        {expanded && (
          <Button
            appearance="link"
            size="sm"
            className="shrink-0 px-0"
            onClick={() => onDetailsChange(!details)}
          >
            {t(
              details
                ? "settings.mcp.ladder.hide-details"
                : "settings.mcp.ladder.show-details"
            )}
          </Button>
        )}
      </div>

      <CollapsiblePanel>
        <ul className="flex flex-col">
          {MCP_CAPABILITY_ROWS.map((row, index) => (
            <LadderRow
              key={row.id}
              row={row}
              served={isRowServed(mode, row)}
              details={details}
              closesTier={tierClosedBy(index)}
            />
          ))}
        </ul>
        {/* The floor is one line rather than a row: it is what no mode serves,
            so it has no mark and belongs to no tier. */}
        <p className="bg-error/5 px-3 py-2 text-sm text-error">
          <span className="font-medium">
            {t("settings.mcp.ladder.floor.label")}
          </span>{" "}
          {t("settings.mcp.ladder.floor.text")}
        </p>
      </CollapsiblePanel>
    </Collapsible>
  );
}

function LadderRow({
  row,
  served,
  details,
  closesTier,
}: {
  row: MCPCapabilityRow;
  served: boolean;
  details: boolean;
  closesTier: ReturnType<typeof tierClosedBy>;
}) {
  const { t } = useTranslation();
  return (
    <li>
      <div className="flex items-start gap-x-2 border-b border-block-border px-1 py-2">
        <span
          aria-hidden="true"
          className={cn(
            "mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-full",
            served
              ? "bg-success/10 text-success"
              : "bg-control-bg text-control-light"
          )}
        >
          {served ? <Check className="size-3" /> : <Minus className="size-3" />}
        </span>
        <div className="flex min-w-0 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <span
              className={cn(
                "text-sm",
                served ? "font-medium text-main" : "text-control-light"
              )}
            >
              {t(`settings.mcp.ladder.row.${row.id}.title`)}
            </span>
            {/* The tier tag rides the served rows only, so a read-only policy
                never shows the word "write" beside something it allows. */}
            {served && (
              <Badge
                variant={row.tier === "read" ? "success" : "warning"}
                className="px-2 py-0 text-xs"
              >
                {t(`settings.mcp.ladder.tier.${row.tier}`)}
              </Badge>
            )}
          </div>
          {details && (
            <p
              className={cn(
                "text-xs leading-4",
                served ? "text-control" : "text-control-light"
              )}
            >
              {t(`settings.mcp.ladder.row.${row.id}.details`)}
            </p>
          )}
        </div>
      </div>
      {closesTier && (
        <p className="flex items-center gap-x-2 border-b border-block-border py-2 text-xs text-control-light">
          <span className="h-px flex-1 bg-block-border" aria-hidden="true" />
          <span className="uppercase tracking-wide">
            {t(`settings.mcp.ladder.stops.${closesTier}`)}
          </span>
          <span className="h-px flex-1 bg-block-border" aria-hidden="true" />
        </p>
      )}
    </li>
  );
}
