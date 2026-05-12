import {
  Check,
  Diamond,
  Glasses,
  Key,
  Link,
  Package,
  Parentheses,
  SquareFunction,
  Table,
  TableProperties,
  TableRowsSplit,
  Zap,
} from "lucide-react";
import { cn } from "@/react/lib/utils";

/**
 * React equivalents of `frontend/src/components/Icon/*Icon.vue`. Kept
 * local to the SchemaPane port for now; will lift to a shared
 * `src/react/components/Icon/` directory if Stage 16+ surfaces need them.
 *
 * Each icon mirrors its Vue counterpart's tint / stroke choices so the
 * tree visually matches the Vue version 1:1.
 */

type IconProps = {
  readonly className?: string;
};

const baseSize = "size-4";

export function TableLeafIcon({ className }: IconProps) {
  return <Table className={cn(baseSize, className)} />;
}

export function ExternalTableIcon({ className }: IconProps) {
  return <TableProperties className={cn(baseSize, className)} />;
}

export function ViewIcon({ className }: IconProps) {
  return (
    <div className={cn("relative", baseSize, className)}>
      <Table className={cn(baseSize, "text-gray-400")} />
      <Glasses
        className="absolute bottom-0 right-0 w-3/4 h-3/4 fill-white"
        stroke="rgb(var(--color-accent))"
        strokeWidth={3.5}
      />
    </div>
  );
}

export function ProcedureIcon({ className }: IconProps) {
  return <Parentheses className={cn(baseSize, "text-gray-400", className)} />;
}

export function FunctionIcon({ className }: IconProps) {
  return (
    <SquareFunction className={cn(baseSize, "text-gray-400", className)} />
  );
}

/**
 * Vue's SequenceIcon is a custom 3.5×3-rem framed "123" — there is no
 * lucide equivalent, so we replicate the markup verbatim.
 */
export function SequenceIcon({ className }: IconProps) {
  return (
    <div
      className={cn(
        "relative w-4 h-4 inline-flex items-center justify-center text-gray-500",
        className
      )}
    >
      <div className="border-y border-current w-3.5 h-3 inline-flex items-center justify-center text-[7px] overflow-visible whitespace-nowrap font-semibold">
        <span className="leading-3">123</span>
      </div>
    </div>
  );
}

export function TriggerIcon({ className }: IconProps) {
  return <Zap className={cn(baseSize, "text-amber-500", className)} />;
}

export function PackageIcon({ className }: IconProps) {
  return <Package className={cn(baseSize, "text-gray-400", className)} />;
}

export function ForeignKeyIcon({ className }: IconProps) {
  return <Link className={cn("size-3.5 text-gray-500", className)} />;
}

export function TablePartitionIcon({ className }: IconProps) {
  return (
    <TableRowsSplit
      className={cn(baseSize, "opacity-75", className)}
      strokeWidth={1.75}
    />
  );
}

/**
 * Vue's PrimaryKeyIcon is heroicons/key in `text-amber-500`.
 */
export function PrimaryKeyIcon({ className }: IconProps) {
  return (
    <div className={cn("relative overflow-hidden", baseSize, className)}>
      <Key className="w-full h-full mx-auto text-amber-500" />
    </div>
  );
}

/**
 * Vue's IndexIcon is tabler/diamonds (a 3-diamond cluster) — lucide's
 * Diamond is a single diamond which is close enough at 12-14px.
 */
export function IndexIcon({ className }: IconProps) {
  return <Diamond className={cn("w-3 h-3", className)} />;
}

/**
 * Vue's CheckIcon is tabler/check at 12px in `text-gray-500`.
 */
export function CheckConstraintIcon({ className }: IconProps) {
  return <Check className={cn("size-3.5 text-gray-500", className)} />;
}

// ColumnIcon is shared with SchemaEditorLite; both surfaces must render
// identical icons so users don't see drift across editors.
export { ColumnIcon } from "@/react/components/schema/icons";
