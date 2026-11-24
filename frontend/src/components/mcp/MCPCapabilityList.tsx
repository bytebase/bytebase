import { Check, Minus } from "lucide-react";
import { useTranslation } from "react-i18next";
import type {
  MCPCapabilityRow,
  MCPCapabilityTier,
} from "@/components/mcp/mcpCapabilityRows";
import {
  isRowServed,
  MCP_CAPABILITY_ROWS,
  mcpRowKey,
  mcpTierKey,
} from "@/components/mcp/mcpCapabilityRows";
import type { MCPServingMode } from "@/components/mcp/mcpPolicy";
import type { BadgeProps } from "@/components/ui/badge";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";

const TIER_VARIANT: Record<MCPCapabilityTier, BadgeProps["variant"]> = {
  read: "success",
  write: "warning",
};

interface Props {
  readonly mode: MCPServingMode;
  /** Omitted where the caller renders the first heading itself. */
  readonly allowedHeading?: string;
  readonly refusedHeading: string;
  /** The level that nests the group headings under the caller's own. */
  readonly headingLevel: 2 | 4;
}

/**
 * The capability rows shared by policy configuration and consent: what the
 * mode serves, then what it refuses, ending with what no mode serves.
 */
export function MCPCapabilityList({
  mode,
  allowedHeading,
  refusedHeading,
  headingLevel,
}: Props) {
  const Heading = headingLevel === 2 ? "h2" : "h4";
  const allowed = MCP_CAPABILITY_ROWS.filter((row) => isRowServed(mode, row));
  const refused = MCP_CAPABILITY_ROWS.filter((row) => !isRowServed(mode, row));
  return (
    <div className="flex flex-col gap-y-3">
      {allowedHeading && (
        <Heading className={GROUP_HEADING_CLASS}>{allowedHeading}</Heading>
      )}
      <ul role="list" className="flex flex-col gap-y-2">
        {allowed.map((row) => (
          <CapabilityRow key={row.id} row={row} served />
        ))}
      </ul>
      <Separator />
      <Heading className={GROUP_HEADING_CLASS}>{refusedHeading}</Heading>
      <ul role="list" className="flex flex-col gap-y-2">
        {refused.map((row) => (
          <CapabilityRow key={row.id} row={row} served={false} />
        ))}
        <FloorRow />
      </ul>
    </div>
  );
}

const GROUP_HEADING_CLASS = "text-base font-semibold text-main";

function CapabilityRow({
  row,
  served,
}: {
  row: MCPCapabilityRow;
  served: boolean;
}) {
  const { t } = useTranslation();
  return (
    <li className="flex items-start gap-x-2 px-1 py-1">
      <Mark served={served} />
      <div className="flex min-w-0 flex-wrap items-center gap-2">
        <span className={cn("text-sm", served ? "text-main" : "text-control")}>
          {t(mcpRowKey(row))}
        </span>
        {served && (
          <Badge variant={TIER_VARIANT[row.tier]} className="px-2 py-0 text-xs">
            {t(mcpTierKey(row.tier))}
          </Badge>
        )}
      </div>
    </li>
  );
}

function Mark({ served }: { served: boolean }) {
  const { t } = useTranslation();
  return (
    <span
      className={cn(
        "mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-full",
        served ? "bg-success/10 text-success" : "bg-error/10 text-error"
      )}
    >
      {served ? (
        <Check className="size-3" aria-hidden="true" />
      ) : (
        <Minus className="size-3" aria-hidden="true" />
      )}
      <span className="sr-only">
        {served
          ? t("settings.mcp.ladder.mark.allowed")
          : t("settings.mcp.ladder.mark.refused")}
      </span>
    </span>
  );
}

function FloorRow() {
  const { t } = useTranslation();
  return (
    <li className="flex items-start gap-x-2 px-1 py-1">
      <Mark served={false} />
      <span className="text-sm text-control">
        {t("settings.mcp.ladder.floor.text")}
      </span>
    </li>
  );
}
