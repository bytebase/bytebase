import { Collapsible as BaseCollapsible } from "@base-ui/react/collapsible";
import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";

function Collapsible({
  className,
  ...props
}: ComponentProps<typeof BaseCollapsible.Root>) {
  return (
    <BaseCollapsible.Root
      className={cn("flex flex-col", className)}
      {...props}
    />
  );
}

/**
 * The button that opens and closes the panel. Base UI owns `aria-expanded` and
 * `aria-controls`; a consumer supplies the visible label and its own indicator.
 */
function CollapsibleTrigger({
  className,
  ...props
}: ComponentProps<typeof BaseCollapsible.Trigger>) {
  return (
    <BaseCollapsible.Trigger
      className={cn(
        "flex w-full items-center gap-x-2 text-left cursor-pointer",
        "focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2",
        className
      )}
      {...props}
    />
  );
}

function CollapsiblePanel({
  className,
  ...props
}: ComponentProps<typeof BaseCollapsible.Panel>) {
  return <BaseCollapsible.Panel className={className} {...props} />;
}

export { Collapsible, CollapsiblePanel, CollapsibleTrigger };
