import { ChevronDown } from "lucide-react";
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
  TablePartitionMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { useViewStateNav } from "../useViewStateNav";

type PartitionRow = {
  partition: TablePartitionMetadata;
  parent?: TablePartitionMetadata;
};

interface PartitionsTableProps {
  table: TableMetadata;
  keyword?: string;
}

export function PartitionsTable({ table, keyword }: PartitionsTableProps) {
  const { t } = useTranslation();
  const { detail } = useViewStateNav();
  const containerRef = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => {
    const trimmed = keyword?.trim().toLowerCase();
    if (!trimmed) return table.partitions;
    return table.partitions.filter(
      (partition) =>
        partition.name.toLowerCase().includes(trimmed) ||
        partition.subpartitions.some((sub) =>
          sub.name.toLowerCase().includes(trimmed)
        )
    );
  }, [table.partitions, keyword]);

  const flat = useMemo(() => {
    const out: PartitionRow[] = [];
    const walk = (
      partition: TablePartitionMetadata,
      parent?: TablePartitionMetadata
    ) => {
      out.push({ partition, parent });
      partition.subpartitions?.forEach((child) => walk(child, partition));
    };
    filtered.forEach((partition) => walk(partition));
    return out;
  }, [filtered]);

  const selectedKey = detail?.partition;

  useEffect(() => {
    if (!selectedKey) return;
    const el = containerRef.current?.querySelector<HTMLElement>(
      `[data-key="${CSS.escape(selectedKey)}"]`
    );
    el?.scrollIntoView({ block: "nearest" });
  }, [selectedKey]);

  const rowKey = (item: PartitionRow) => {
    return item.parent
      ? `${item.parent.name}/${item.partition.name}`
      : item.partition.name;
  };

  return (
    <div ref={containerRef} className="w-full h-full overflow-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-6" />
            <TableHead>{t("common.name")}</TableHead>
            <TableHead>{t("common.type")}</TableHead>
            <TableHead>
              {t("schema-editor.table-partition.expression")}
            </TableHead>
            <TableHead>{t("schema-editor.table-partition.value")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {flat.map((item) => {
            const key = rowKey(item);
            return (
              <TableRow
                key={key}
                data-key={key}
                data-state={selectedKey === key ? "selected" : undefined}
              >
                <TableCell className="w-6">
                  {item.partition.subpartitions?.length > 0 ? (
                    <ChevronDown className="size-4" />
                  ) : null}
                </TableCell>
                <TableCell className="truncate max-w-[320px]">
                  <HighlightLabelText
                    text={item.partition.name}
                    keyword={keyword}
                  />
                </TableCell>
                <TableCell className="truncate">
                  {item.partition.type}
                </TableCell>
                <TableCell className="truncate">
                  {item.partition.expression}
                </TableCell>
                <TableCell className="truncate">
                  {item.partition.value}
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
