import { cn } from "@/react/lib/utils";

type Props = {
  readonly className?: string;
  readonly children: React.ReactNode;
};

// Horizontal rule with centered content (text or links); the children mask
// the line with their own background (bg-white on the auth surfaces).
export function AuthDivider({ className, children }: Props) {
  return (
    <div className={cn("relative", className)}>
      <div aria-hidden="true" className="absolute inset-0 flex items-center">
        <div className="w-full border-t border-control-border" />
      </div>
      <div className="relative flex justify-center text-sm">{children}</div>
    </div>
  );
}
