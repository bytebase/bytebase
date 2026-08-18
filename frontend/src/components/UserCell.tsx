import type { ReactNode } from "react";
import type { RouteTarget } from "@/app/router";
import { RouterLink } from "@/components/RouterLink";
import { UserHoverCard } from "@/components/UserHoverCard";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { UserAvatar } from "./UserAvatar";

interface UserCellProps {
  /** Display name. Falls back to email for avatar if empty. */
  title: string;
  /** Subtitle line (typically email). */
  subtitle?: string;
  /** Size of the avatar. */
  size?: "sm" | "md";
  /** Whether to show the avatar. Default true. */
  showAvatar?: boolean;
  /** Custom avatar element (e.g. group icon). Overrides the default UserAvatar. */
  avatar?: ReactNode;
  /** Name styling when deleted / expired. */
  nameClassName?: string;
  /** Wrap the name in a clickable element. */
  nameLink?: {
    onClick?: () => void;
    to?: RouteTarget;
  };
  /**
   * Email of the person this cell names. When set, the name gains a hover card
   * showing who they are. Used where a name is mentioned in passing and there
   * is no page to send the reader to.
   */
  hoverEmail?: string;
  /** Inline badges rendered after the name. */
  badges?: ReactNode;
  /** Extra className on the outer wrapper. */
  className?: string;
}

export function UserCell({
  title,
  subtitle,
  size = "md",
  showAvatar = true,
  avatar,
  nameClassName,
  nameLink,
  hoverEmail,
  badges,
  className,
}: UserCellProps) {
  const nameContent = title || subtitle || "?";

  const nameClassNameMerged = cn(
    "font-medium text-accent hover:underline cursor-pointer",
    size === "sm" && "text-sm",
    nameClassName
  );

  const nameEl = nameLink?.to ? (
    <RouterLink
      className={nameClassNameMerged}
      to={nameLink.to}
      onClick={(e) => {
        e.stopPropagation();
      }}
    >
      {nameContent}
    </RouterLink>
  ) : nameLink?.onClick ? (
    <Button
      type="button"
      appearance="link"
      size="sm"
      className={cn(
        nameClassNameMerged,
        "h-auto p-0 justify-start rounded-none leading-normal"
      )}
      onClick={(e) => {
        e.stopPropagation();
        nameLink.onClick?.();
      }}
    >
      {nameContent}
    </Button>
  ) : (
    <span
      className={cn(
        "font-medium text-main",
        size === "sm" && "text-sm",
        nameClassName
      )}
    >
      {nameContent}
    </span>
  );

  const hoverableNameEl = hoverEmail ? (
    <UserHoverCard email={hoverEmail} fallbackTitle={title}>
      {nameEl}
    </UserHoverCard>
  ) : (
    nameEl
  );

  return (
    <div className={cn("flex items-center gap-x-3", className)}>
      {showAvatar &&
        (avatar ?? <UserAvatar title={title || subtitle || "?"} size={size} />)}
      <div className="flex flex-col">
        <div className="flex items-center gap-x-1.5">
          {hoverableNameEl}
          {badges}
        </div>
        {subtitle && (
          <span className="text-control-light text-xs">{subtitle}</span>
        )}
      </div>
    </div>
  );
}
