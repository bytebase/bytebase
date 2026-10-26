import type { HTMLAttributes, ReactNode } from "react";
import { cn } from "@/lib/utils";

type QueryResultColumn = { id: string; name: string };

interface QueryResultVerticalRowProps<TColumn extends QueryResultColumn>
  extends Omit<HTMLAttributes<HTMLDivElement>, "children"> {
  columns: TColumn[];
  renderColumnName?: (column: TColumn, index: number) => ReactNode;
  renderColumnAccessory?: (column: TColumn, index: number) => ReactNode;
  renderValue: (column: TColumn, index: number) => ReactNode;
  actions?: ReactNode;
}

export function QueryResultVerticalRow<TColumn extends QueryResultColumn>({
  columns,
  renderColumnName = (column) => column.name,
  renderColumnAccessory,
  renderValue,
  actions,
  className,
  ...divProps
}: QueryResultVerticalRowProps<TColumn>) {
  return (
    <div
      {...divProps}
      className={cn(
        "relative rounded-sm bg-control-bg/40 px-3 py-2",
        className
      )}
    >
      {actions}
      <dl>
        {columns.map((column, columnIndex) => (
          <div
            key={`${column.id}-${columnIndex}`}
            className="flex items-start text-sm text-control-light"
          >
            <dt className="flex min-w-28 items-center pt-1 text-left font-medium">
              <div className="flex items-center gap-x-1">
                {renderColumnName(column, columnIndex)}
                {renderColumnAccessory?.(column, columnIndex)}
              </div>
              {": "}
            </dt>
            <dd className="min-w-0 flex-1">
              {renderValue(column, columnIndex)}
            </dd>
          </div>
        ))}
      </dl>
    </div>
  );
}
