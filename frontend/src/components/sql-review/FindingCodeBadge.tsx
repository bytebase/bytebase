import { badgeVariants } from "@/components/ui/badge";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { findingCodeTitle } from "./standardRuleCatalog";

// A finding code, such as "WT101", with its title on hover when this build
// knows the code.
export function FindingCodeBadge({ code }: { code: string }) {
  const title = findingCodeTitle(code);
  return (
    <Tooltip
      content={title}
      render={
        <span
          className={cn(badgeVariants(), "shrink-0 font-mono")}
          data-testid="review-code"
          tabIndex={title ? 0 : undefined}
        />
      }
    >
      {code}
    </Tooltip>
  );
}
