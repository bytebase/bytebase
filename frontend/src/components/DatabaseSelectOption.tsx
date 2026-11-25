import { DatabaseTargetDisplay } from "@/components/DatabaseTargetDisplay";
import { HighlightLabelText } from "@/components/HighlightLabelText";
import type { Database } from "@/types/proto-es/v1/database_service_pb";

// A database as a row in a picker: the target display, its resource name, and
// the reason it cannot be picked, when there is one.
export function DatabaseSelectOption({
  database,
  keyword,
  disabledReason,
}: {
  readonly database: Database;
  readonly keyword: string;
  readonly disabledReason?: string;
}) {
  return (
    <div className="flex flex-col gap-1">
      <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
        <DatabaseTargetDisplay
          database={database}
          keyword={keyword}
          showEnvironment
        />
        {disabledReason && (
          <span className="text-xs text-warning">{disabledReason}</span>
        )}
      </div>
      <HighlightLabelText
        text={database.name}
        keyword={keyword}
        className="text-xs text-control-placeholder"
      />
    </div>
  );
}
