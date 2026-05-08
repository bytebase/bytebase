import { create } from "@bufbuild/protobuf";
import { ChevronRight } from "lucide-react";
import { EngineIcon } from "@/react/components/EngineIcon";
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
import { RequestQueryButton } from "@/react/components/sql-editor/RequestQueryButton";
import { Checkbox } from "@/react/components/ui/checkbox";
import { Tooltip } from "@/react/components/ui/tooltip";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useSQLEditorTabStore } from "@/store";
import type { SQLEditorTreeNode } from "@/types";
import { PermissionDeniedDetailSchema } from "@/types/proto-es/v1/common_pb";
import {
  extractDatabaseResourceName,
  getInstanceResource,
  instanceV1Name,
  isDatabaseV1Queryable,
} from "@/utils";

type Props = {
  readonly node: SQLEditorTreeNode;
  readonly keyword: string;
  readonly checked?: boolean;
  readonly checkDisabled?: boolean;
  readonly checkTooltip?: string;
  readonly onCheckedChange?: (checked: boolean) => void;
};

/**
 * Replaces frontend/src/views/sql-editor/ConnectionPanel/ConnectionPane/TreeNode/DatabaseNode.vue.
 * Shows an optional batch-mode checkbox, an inline "Instance → Database"
 * breadcrumb (engine icon + instance name, chevron, database name), and a
 * RequestQueryButton when the current user can't query this database.
 *
 * The breadcrumb is purely presentational — clicks bubble to the parent
 * row so the row-level handler (TreeRow → onConnect → connect()) runs.
 * Only interactive children (checkbox, RequestQueryButton) stop
 * propagation locally to avoid double-firing the row click.
 */
export function DatabaseNode({
  node,
  keyword,
  checked,
  checkDisabled,
  checkTooltip,
  onCheckedChange,
}: Props) {
  const tabStore = useSQLEditorTabStore();
  const supportBatchMode = useVueState(() => tabStore.supportBatchMode);

  const database = (node as SQLEditorTreeNode<"database">).meta.target;
  const instance = getInstanceResource(database);
  const databaseName = extractDatabaseResourceName(database.name).databaseName;
  const canQuery = isDatabaseV1Queryable(database);

  const checkbox = supportBatchMode ? (
    <Tooltip content={checkTooltip ?? ""}>
      <Checkbox
        checked={!!checked}
        className="mr-2"
        disabled={checkDisabled}
        onClick={(e) => e.stopPropagation()}
        onCheckedChange={(checked) => onCheckedChange?.(checked)}
      />
    </Tooltip>
  ) : null;

  return (
    <div className="flex items-center max-w-full overflow-hidden gap-x-1">
      {checkbox}

      <div
        className={cn(
          "cursor-pointer tree-node-database",
          "flex flex-row justify-start items-center gap-x-1 min-w-0"
        )}
      >
        <div className="flex flex-row items-center gap-x-1 min-w-0">
          <EngineIcon engine={instance.engine} className="size-4" />
          <span className="truncate">{instanceV1Name(instance)}</span>
        </div>
        <ChevronRight className="size-3 shrink-0" />
        <span className="truncate">
          <HighlightLabelText text={databaseName} keyword={keyword} />
        </span>
      </div>

      {!canQuery && (
        <div className="ml-auto">
          <RequestQueryButton
            size="sm"
            text
            permissionDeniedDetail={create(PermissionDeniedDetailSchema, {
              resources: [database.name],
              requiredPermissions: ["bb.sql.select"],
            })}
          />
        </div>
      )}
    </div>
  );
}
