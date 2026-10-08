import { ChevronRight } from "lucide-react";
import { useTranslation } from "react-i18next";
import { MCPCapabilityList } from "@/components/mcp/MCPCapabilityList";
import {
  type MCPServingMode,
  mcpModeKey,
  mcpSummaryKey,
} from "@/components/mcp/mcpPolicy";
import { Button } from "@/components/ui/button";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";

interface Props {
  readonly mode: MCPServingMode;
  readonly expanded: boolean;
  readonly details: boolean;
  readonly onExpandedChange: (expanded: boolean) => void;
  readonly onDetailsChange: (details: boolean) => void;
}

/**
 * What a mode allows, as capability rows in one ordered list of which every
 * mode is a prefix.
 *
 * The served set comes from the mode's tier alone, so there is no comparison
 * against a stored mode and the edit state renders exactly what the view will
 * show after saving. Rows a mode does not serve stay visible and muted, which is
 * what lets an admin compare two modes without opening a second surface.
 */
export function MCPCapabilityLadder({
  mode,
  expanded,
  details,
  onExpandedChange,
  onDetailsChange,
}: Props) {
  const { t } = useTranslation();

  return (
    <Collapsible open={expanded} onOpenChange={onExpandedChange}>
      <div className="flex items-center gap-x-2 py-2">
        <CollapsibleTrigger
          className={cn(
            "min-w-0 rounded-xs",
            expanded ? "w-auto shrink-0" : "flex-1"
          )}
          data-testid="mcp-ladder-trigger"
        >
          <ChevronRight
            className={cn(
              "size-4 shrink-0 text-control-light transition-transform",
              expanded && "rotate-90"
            )}
          />
          {/* Neither label repeats the mode name the chip or the selector
              already shows, except the heading, whose job is to say whose list
              this is. */}
          <span
            className={cn(
              "min-w-0",
              expanded
                ? "text-base font-semibold text-main"
                : "text-sm text-control-light"
            )}
          >
            {expanded
              ? t("settings.mcp.ladder.heading", {
                  mode: t(mcpModeKey(mode, "title")),
                })
              : t(mcpSummaryKey(mode))}
          </span>
        </CollapsibleTrigger>
        {expanded && (
          <Button
            appearance="link"
            size="sm"
            className="shrink-0"
            onClick={() => onDetailsChange(!details)}
          >
            {details
              ? t("settings.mcp.ladder.hide-details")
              : t("settings.mcp.ladder.show-details")}
          </Button>
        )}
      </div>

      <CollapsiblePanel>
        <MCPCapabilityList mode={mode} details={details} tierDividers floor />
      </CollapsiblePanel>
    </Collapsible>
  );
}
