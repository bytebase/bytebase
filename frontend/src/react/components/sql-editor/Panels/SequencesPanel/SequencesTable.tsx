import { useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { Checkbox } from "@/react/components/ui/checkbox";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  SequenceMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { keyWithPosition } from "@/views/sql-editor/EditorCommon/utils";
import { EllipsisCell } from "../common/EllipsisCell";
import { useViewStateNav } from "../common/useViewStateNav";

type SequenceRow = {
  sequence: SequenceMetadata;
  position: number;
};

interface SequencesTableProps {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  sequences: SequenceMetadata[];
  keyword?: string;
  onSelect: (selected: {
    database: DatabaseMetadata;
    schema: SchemaMetadata;
    sequence: SequenceMetadata;
    position: number;
  }) => void;
}

export function SequencesTable({
  database,
  schema,
  sequences,
  keyword,
  onSelect,
}: SequencesTableProps) {
  const { t } = useTranslation();
  const { detail } = useViewStateNav();
  const containerRef = useRef<HTMLDivElement>(null);

  const rows = useMemo(() => {
    const all = sequences.map<SequenceRow>((sequence, position) => ({
      sequence,
      position,
    }));
    const trimmed = keyword?.trim().toLowerCase();
    if (!trimmed) return all;
    return all.filter(({ sequence }) =>
      sequence.name.toLowerCase().includes(trimmed)
    );
  }, [sequences, keyword]);

  const selectedKey = detail?.sequence;

  useEffect(() => {
    if (!selectedKey) return;
    const el = containerRef.current?.querySelector<HTMLElement>(
      `[data-key="${CSS.escape(selectedKey)}"]`
    );
    el?.scrollIntoView({ block: "nearest" });
  }, [selectedKey]);

  return (
    <div ref={containerRef} className="w-full h-full overflow-auto px-2 py-2">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t("schema-editor.database.name")}</TableHead>
            <TableHead>{t("db.sequence.data-type")}</TableHead>
            <TableHead>{t("db.sequence.start")}</TableHead>
            <TableHead>{t("db.sequence.min-value")}</TableHead>
            <TableHead>{t("db.sequence.max-value")}</TableHead>
            <TableHead>{t("db.sequence.increment")}</TableHead>
            <TableHead>{t("db.sequence.cycle")}</TableHead>
            <TableHead>{t("db.sequence.cacheSize")}</TableHead>
            <TableHead>{t("db.sequence.lastValue")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map(({ sequence, position }) => {
            const rowKey = keyWithPosition(sequence.name, position);
            return (
              <TableRow
                key={rowKey}
                data-key={rowKey}
                className="cursor-pointer"
                data-state={selectedKey === rowKey ? "selected" : undefined}
                onClick={() =>
                  onSelect({ database, schema, sequence, position })
                }
              >
                <TableCell>
                  <EllipsisCell content={sequence.name} keyword={keyword} />
                </TableCell>
                <TableCell>{sequence.dataType}</TableCell>
                <TableCell>{sequence.start}</TableCell>
                <TableCell>{sequence.minValue}</TableCell>
                <TableCell>
                  <EllipsisCell content={sequence.maxValue} />
                </TableCell>
                <TableCell>{sequence.increment}</TableCell>
                <TableCell>
                  <Checkbox checked={sequence.cycle} disabled />
                </TableCell>
                <TableCell>{sequence.cacheSize}</TableCell>
                <TableCell>
                  <EllipsisCell content={sequence.lastValue} />
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
