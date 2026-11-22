import type { HTMLAttributes, ReactNode } from "react";
import type { LayerFamily } from "@/components/ui/layer";
import { Tooltip } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

type QueryResultColumn = { id: string; name: string };

interface QueryResultVerticalRowProps<TColumn extends QueryResultColumn>
  extends Omit<HTMLAttributes<HTMLDivElement>, "children"> {
  columns: TColumn[];
  renderColumnName?: (column: TColumn, index: number) => ReactNode;
  renderColumnAccessory?: (column: TColumn, index: number) => ReactNode;
  renderValue: (column: TColumn, index: number) => ReactNode;
  tooltipLayer?: LayerFamily;
  actions?: ReactNode;
}

export function QueryResultVerticalRow<TColumn extends QueryResultColumn>({
  columns,
  renderColumnName = (column) => column.name,
  renderColumnAccessory,
  renderValue,
  tooltipLayer,
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
            <dt className="flex w-28 max-w-1/3 min-w-0 shrink-0 items-center pt-1 text-left font-medium">
              <div className="flex min-w-0 items-center gap-x-1">
                <Tooltip
                  content={column.name}
                  layer={tooltipLayer}
                  popupClassName="max-w-96 whitespace-pre-wrap wrap-anywhere"
                  render={
                    <span
                      tabIndex={0}
                      className="min-w-0 truncate focus-visible:outline-2 focus-visible:outline-accent"
                    />
                  }
                >
                  {renderColumnName(column, columnIndex)}
                </Tooltip>
                {renderColumnAccessory?.(column, columnIndex)}
              </div>
              {": "}
            </dt>
            <dd className="min-w-0 flex-1 wrap-anywhere">
              {renderValue(column, columnIndex)}
            </dd>
          </div>
        ))}
      </dl>
    </div>
  );
}
