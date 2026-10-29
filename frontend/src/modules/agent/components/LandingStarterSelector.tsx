import { ChevronDown } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Tooltip } from "@/components/ui/tooltip";
import type { LandingStarterId } from "../logic/landingStarters";

interface LandingStarterSelectorProps {
  starterIds: LandingStarterId[];
  disabled: boolean;
  onSelect: (starterId: LandingStarterId) => void;
}

export function LandingStarterSelector({
  starterIds,
  disabled,
  onSelect,
}: LandingStarterSelectorProps) {
  const { t } = useTranslation();
  const [primary, ...alternatives] = starterIds;

  if (!primary) {
    return null;
  }

  const labelFor = (starterId: LandingStarterId) =>
    t(`agent.landing-starters.${starterId}`);

  return (
    <div className="flex min-w-0 items-center gap-1">
      <Button
        type="button"
        appearance="link"
        size="xs"
        className="h-auto min-w-0 max-w-full px-0 text-left whitespace-normal"
        disabled={disabled}
        data-testid="landing-starter-primary"
        onClick={() => onSelect(primary)}
      >
        {labelFor(primary)}
      </Button>
      {alternatives.length > 0 && (
        <DropdownMenu>
          <Tooltip content={t("agent.landing-starters.more")}>
            <DropdownMenuTrigger
              render={
                <Button
                  type="button"
                  appearance="secondary"
                  size="xs"
                  className="shrink-0"
                  disabled={disabled}
                  aria-label={t("agent.landing-starters.more")}
                  data-testid="landing-starter-overflow"
                >
                  <ChevronDown className="size-3.5" />
                </Button>
              }
            />
          </Tooltip>
          <DropdownMenuContent
            align="start"
            aria-label={t("agent.landing-starters.menu")}
            data-testid="landing-starter-menu"
            className="w-80 max-w-[calc(100vw-2rem)] max-h-64 overflow-y-auto p-1"
          >
            {alternatives.map((starterId) => (
              <DropdownMenuItem
                key={starterId}
                disabled={disabled}
                data-starter-id={starterId}
                className="h-auto min-w-0 whitespace-normal break-words py-1.5"
                onClick={() => onSelect(starterId)}
              >
                {labelFor(starterId)}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </div>
  );
}
