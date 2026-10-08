import { Check, Minus } from "lucide-react";
import { Fragment } from "react";
import { useTranslation } from "react-i18next";
import type {
  MCPCapabilityRow,
  MCPCapabilityTier,
} from "@/components/mcp/mcpCapabilityRows";
import {
  isRowServed,
  MCP_CAPABILITY_TIERS,
  mcpRowKey,
  mcpTierKey,
  rowsInTier,
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
  readonly details: boolean;
  readonly tierDividers?: boolean;
  readonly floor?: boolean;
}

/**
 * The capability rows shared by policy configuration and consent.
 *
 * Every consumer shows the same capability boundary: what the selected mode
 * serves, where it stops, and what it refuses.
 */
export function MCPCapabilityList({
  mode,
  details,
  tierDividers = false,
  floor = false,
}: Props) {
  const { t } = useTranslation();
  return (
    <>
      <ul role="list" className="flex flex-col gap-y-2">
        {MCP_CAPABILITY_TIERS.map((tier) => (
          <Fragment key={tier}>
            {rowsInTier(tier).map((row) => (
              <CapabilityRow
                key={row.id}
                row={row}
                served={isRowServed(mode, row)}
                details={details}
              />
            ))}
            {tierDividers && <TierDivider tier={tier} />}
          </Fragment>
        ))}
      </ul>
      {floor && (
        <p className="bg-error/5 px-3 py-2 text-sm text-error my-2">
          <span className="font-medium">
            {t("settings.mcp.ladder.floor.label")}
          </span>{" "}
          {t("settings.mcp.ladder.floor.text")}
        </p>
      )}
    </>
  );
}

function CapabilityRow({
  row,
  served,
  details,
}: {
  row: MCPCapabilityRow;
  served: boolean;
  details: boolean;
}) {
  const { t } = useTranslation();
  return (
    <li className="flex items-start gap-x-2 px-1 py-1">
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
      <div className="flex min-w-0 flex-col gap-1">
        <div className="flex flex-wrap items-center gap-2">
          <span
            className={cn("text-sm", served ? "text-main" : "text-control")}
          >
            {t(mcpRowKey(row, "title"))}
          </span>
          {served && (
            <Badge
              variant={TIER_VARIANT[row.tier]}
              className="px-2 py-0 text-xs"
            >
              {t(mcpTierKey(row.tier, "tier"))}
            </Badge>
          )}
        </div>
        {details && (
          <p className="text-xs leading-5 text-control-light">
            {t(mcpRowKey(row, "details"))}
          </p>
        )}
      </div>
    </li>
  );
}

function TierDivider({ tier }: { tier: MCPCapabilityTier }) {
  const { t } = useTranslation();
  return (
    <li
      role="presentation"
      className="flex items-center gap-x-2 py-2 text-xs text-control-light"
    >
      <Separator className="flex-1" />
      <span className="uppercase font-medium text-main tracking-wide">
        {t(mcpTierKey(tier, "stops"))}
      </span>
      <Separator className="flex-1" />
    </li>
  );
}
