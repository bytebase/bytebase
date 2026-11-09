import { Bot } from "lucide-react";
import { Button, type ButtonProps } from "@/components/ui/button";
import { type ControlSize, controlSize } from "@/components/ui/styles.stylex";
import { cn } from "@/lib/utils";

type AgentHelpButtonProps = Omit<
  ButtonProps,
  "appearance" | "variant" | "size" | "children"
> & {
  children: string;
  size?: ControlSize;
  iconOnly?: boolean;
};

const iconOnlyWidth = { xs: "w-6", sm: "w-7", md: "w-9", lg: "w-10" };

export function AgentHelpButton({
  children,
  size = "md",
  iconOnly = false,
  className,
  title,
  type = "button",
  ...props
}: AgentHelpButtonProps) {
  return (
    <Button
      {...props}
      type={type}
      appearance="ghost"
      size={size}
      title={title ?? children}
      aria-label={props["aria-label"] ?? children}
      className={cn(
        "max-w-full min-w-0",
        iconOnly && [iconOnlyWidth[size], "shrink-0 px-0"],
        className
      )}
    >
      <Bot
        size={controlSize[size].iconSize}
        className="shrink-0"
        aria-hidden="true"
      />
      {!iconOnly && <span className="min-w-0 truncate">{children}</span>}
    </Button>
  );
}
