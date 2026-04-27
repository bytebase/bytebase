import { useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { HighlightLabelText } from "@/react/components/HighlightLabelText";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import type {
  TableMetadata,
  TriggerMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { keyWithPosition } from "@/views/sql-editor/EditorCommon/utils";
import { useViewStateNav } from "../common/useViewStateNav";

interface TriggersTableProps {
  table?: TableMetadata;
  triggers?: TriggerMetadata[];
  keyword?: string;
  onSelect: (selected: {
    table?: TableMetadata;
    trigger: TriggerMetadata;
    position: number;
  }) => void;
}

/**
 * Single-name trigger list. Used both standalone (in `TriggersPanel`)
 * and inside `TableDetail`'s TRIGGERS tab — `table` is required only
 * for the latter so the click handler can echo the parent table back.
 */
export function TriggersTable({
  table,
  triggers,
  keyword,
  onSelect,
}: TriggersTableProps) {
  const { t } = useTranslation();
  const { detail } = useViewStateNav();
  const containerRef = useRef<HTMLDivElement>(null);

  const rows = useMemo(() => {
    const all = (triggers ?? []).map((trigger, position) => ({
      trigger,
      position,
    }));
    const trimmed = keyword?.trim().toLowerCase();
    if (!trimmed) return all;
    return all.filter(({ trigger }) =>
      trigger.name.toLowerCase().includes(trimmed)
    );
  }, [triggers, keyword]);

  const selectedKey = detail?.trigger;

  useEffect(() => {
    if (!selectedKey) return;
    const el = containerRef.current?.querySelector<HTMLElement>(
      `[data-key="${CSS.escape(selectedKey)}"]`
    );
    el?.scrollIntoView({ block: "nearest" });
  }, [selectedKey]);

  return (
    <div ref={containerRef} className="w-full h-full overflow-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t("schema-editor.database.name")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map(({ trigger, position }) => {
            const rowKey = keyWithPosition(trigger.name, position);
            return (
              <TableRow
                key={rowKey}
                data-key={rowKey}
                className="cursor-pointer"
                data-state={selectedKey === rowKey ? "selected" : undefined}
                onClick={() => onSelect({ table, trigger, position })}
              >
                <TableCell className="truncate">
                  <HighlightLabelText text={trigger.name} keyword={keyword} />
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
