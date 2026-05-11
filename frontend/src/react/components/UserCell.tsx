import type { ReactNode } from "react";
import { cn } from "@/react/lib/utils";
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
    onClick: () => void;
  };
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
  badges,
  className,
}: UserCellProps) {
  const nameContent = title || subtitle || "?";

  const nameEl = nameLink ? (
    <button
      type="button"
      className={cn(
        "font-medium text-accent hover:underline cursor-pointer",
        size === "sm" && "text-sm",
        nameClassName
      )}
      onClick={nameLink.onClick}
    >
      {nameContent}
    </button>
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

  return (
    <div className={cn("flex items-center gap-x-3", className)}>
      {showAvatar &&
        (avatar ?? <UserAvatar title={title || subtitle || "?"} size={size} />)}
      <div className="flex flex-col">
        <div className="flex items-center gap-x-1.5">
          {nameEl}
          {badges}
        </div>
        {subtitle && (
          <span className="text-control-light text-xs">{subtitle}</span>
        )}
      </div>
    </div>
  );
}
