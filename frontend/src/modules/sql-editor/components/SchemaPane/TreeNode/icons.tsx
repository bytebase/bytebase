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
import { cn } from "@/lib/utils";

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
      <Table className={cn(baseSize, "text-control-placeholder")} />
      <Glasses
        className="absolute bottom-0 right-0 w-3/4 h-3/4 fill-background stroke-accent"
        strokeWidth={3.5}
      />
    </div>
  );
}

export function ProcedureIcon({ className }: IconProps) {
  return (
    <Parentheses
      className={cn(baseSize, "text-control-placeholder", className)}
    />
  );
}

export function FunctionIcon({ className }: IconProps) {
  return (
    <SquareFunction
      className={cn(baseSize, "text-control-placeholder", className)}
    />
  );
}

/**
 * A custom framed "123" — there is no lucide equivalent.
 */
export function SequenceIcon({ className }: IconProps) {
  return (
    <div
      className={cn(
        "relative h-4 w-5 inline-flex items-center justify-center text-control-light",
        className
      )}
    >
      <div className="inline-flex h-4 w-5 items-center justify-center border-y border-current overflow-visible whitespace-nowrap text-xs font-semibold leading-4">
        <span>123</span>
      </div>
    </div>
  );
}

export function TriggerIcon({ className }: IconProps) {
  return <Zap className={cn(baseSize, "text-warning", className)} />;
}

export function PackageIcon({ className }: IconProps) {
  return (
    <Package className={cn(baseSize, "text-control-placeholder", className)} />
  );
}

export function ForeignKeyIcon({ className }: IconProps) {
  return <Link className={cn("size-3.5 text-control-light", className)} />;
}

export function TablePartitionIcon({ className }: IconProps) {
  return (
    <TableRowsSplit
      className={cn(baseSize, "opacity-75", className)}
      strokeWidth={1.75}
    />
  );
}

export function PrimaryKeyIcon({ className }: IconProps) {
  return (
    <div className={cn("relative overflow-hidden", baseSize, className)}>
      <Key className="w-full h-full mx-auto text-warning" />
    </div>
  );
}

export function IndexIcon({ className }: IconProps) {
  return <Diamond className={cn("w-3 h-3", className)} />;
}

export function CheckConstraintIcon({ className }: IconProps) {
  return <Check className={cn("size-3.5 text-control-light", className)} />;
}

// ColumnIcon is shared with SchemaEditorLite; both surfaces must render
// identical icons so users don't see drift across editors.
export { ColumnIcon } from "@/components/schema/icons";
