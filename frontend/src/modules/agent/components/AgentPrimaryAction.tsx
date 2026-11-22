import { ArrowUpRight, Check, FilePenLine, Loader2 } from "lucide-react";
import type { ComponentProps } from "react";
import { RouterLink, type RouterLinkProps } from "@/components/RouterLink";
import { Button, buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface ActionPresentation {
  children: string;
  actionKind?: "open" | "apply";
  loading?: boolean;
  completed?: boolean;
}

const actionClassName = "max-w-full min-w-0 justify-start gap-1.5 text-left";

function ActionContent({
  children,
  actionKind = "open",
  loading,
  completed,
}: ActionPresentation) {
  const Icon = loading
    ? Loader2
    : completed
      ? Check
      : actionKind === "apply"
        ? FilePenLine
        : ArrowUpRight;
  return (
    <>
      <span className="min-w-0 truncate">{children}</span>
      <Icon
        className={cn("size-4 shrink-0", loading && "animate-spin")}
        aria-hidden="true"
      />
    </>
  );
}

type AgentPrimaryActionProps = Omit<
  ComponentProps<typeof Button>,
  "children" | "size" | "appearance" | "variant"
> &
  ActionPresentation;

export function AgentPrimaryAction({
  children,
  actionKind,
  loading,
  completed,
  disabled,
  className,
  title,
  ...props
}: AgentPrimaryActionProps) {
  return (
    <Button
      {...props}
      appearance="ghost"
      size="sm"
      className={cn(actionClassName, className)}
      title={title ?? children}
      data-agent-primary-action
      aria-busy={loading || undefined}
      disabled={disabled || loading || completed}
    >
      <ActionContent
        actionKind={actionKind}
        loading={loading}
        completed={completed}
      >
        {children}
      </ActionContent>
    </Button>
  );
}

type AgentPrimaryActionLinkProps = Omit<RouterLinkProps, "children"> &
  ActionPresentation;

export function AgentPrimaryActionLink({
  children,
  actionKind,
  loading,
  completed,
  className,
  title,
  onClick,
  ...props
}: AgentPrimaryActionLinkProps) {
  const unavailable = loading || completed;
  return (
    <RouterLink
      {...props}
      className={buttonVariants({
        appearance: "ghost",
        size: "sm",
        className: cn(
          actionClassName,
          "aria-disabled:cursor-not-allowed aria-disabled:opacity-50",
          className
        ),
      })}
      title={title ?? children}
      data-agent-primary-action
      aria-busy={loading || undefined}
      aria-disabled={unavailable || undefined}
      onClick={(event) => {
        if (unavailable) event.preventDefault();
        else onClick?.(event);
      }}
    >
      <ActionContent
        actionKind={actionKind}
        loading={loading}
        completed={completed}
      >
        {children}
      </ActionContent>
    </RouterLink>
  );
}
