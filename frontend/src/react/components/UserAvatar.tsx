import { cn } from "@/react/lib/utils";

const AVATAR_COLORS = [
  "#F59E0B",
  "#10B981",
  "#8B5CF6",
  "#EC4899",
  "#06B6D4",
  "#EF4444",
];

export function getAvatarColor(name: string) {
  let hash = 0;
  for (let i = 0; i < name.length; i++)
    hash = (hash * 31 + name.charCodeAt(i)) | 0;
  return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length];
}

export function getInitials(name: string) {
  return name
    .split(/\s+/)
    .map((w) => w[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);
}

export function UserAvatar({
  title,
  colorSeed,
  size = "md",
  className,
}: {
  title: string;
  /** Stable string for color derivation (e.g. email). Defaults to title. */
  colorSeed?: string;
  size?: "sm" | "md";
  className?: string;
}) {
  const dim = size === "sm" ? "h-7 w-7 text-xs" : "h-9 w-9 text-sm";
  return (
    <div
      className={cn(
        "rounded-full flex items-center justify-center text-white font-medium shrink-0",
        dim,
        className
      )}
      style={{ backgroundColor: getAvatarColor(colorSeed ?? title) }}
    >
      {getInitials(title)}
    </div>
  );
}
