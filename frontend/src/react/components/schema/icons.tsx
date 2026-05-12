import { cn } from "@/react/lib/utils";

interface IconProps {
  className?: string;
}

// Single-column variant of `lucide:columns-3` (the second internal gap line
// removed). Used in both Schema Editor and SQL Editor tree views so the two
// surfaces stay visually identical.
export function ColumnIcon({ className }: IconProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={cn("size-4 text-control-light", className)}
    >
      <rect width="18" height="18" x="3" y="3" rx="2" />
      <path d="M9 3v18" />
    </svg>
  );
}
