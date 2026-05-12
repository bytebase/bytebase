import { RotateCcw, Trash2 } from "lucide-react";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { cn } from "@/react/lib/utils";
import type {
  Database,
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
  TablePartitionMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { TablePartitionMetadata_Type } from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../../context";

interface Props {
  db: Database;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
  readonly: boolean;
}

export function PartitionsEditor({
  db,
  database: _database,
  schema,
  table,
  readonly: isReadonly,
}: Props) {
  const { t } = useTranslation();
  const { editStatus } = useSchemaEditorContext();

  const handleDropPartition = useCallback(
    (partition: TablePartitionMetadata) => {
      const status = editStatus.getPartitionStatus(db, {
        schema,
        table,
        partition,
      });
      if (status === "created") {
        const idx = table.partitions.indexOf(partition);
        if (idx >= 0) table.partitions.splice(idx, 1);
        editStatus.removeEditStatus(db, { schema, table, partition }, true);
      } else {
        editStatus.markEditStatus(db, { schema, table, partition }, "dropped");
      }
    },
    [editStatus, db, schema, table]
  );

  const handleRestorePartition = useCallback(
    (partition: TablePartitionMetadata) => {
      editStatus.removeEditStatus(db, { schema, table, partition }, false);
    },
    [editStatus, db, schema, table]
  );

  const handlePartitionChange = useCallback(
    (partition: TablePartitionMetadata) => {
      const status = editStatus.getPartitionStatus(db, {
        schema,
        table,
        partition,
      });
      if (status === "normal") {
        editStatus.markEditStatus(db, { schema, table, partition }, "updated");
      }
    },
    [editStatus, db, schema, table]
  );

  return (
    <div className="flex size-full flex-col gap-y-2 overflow-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-[160px]">
              {t("schema-editor.column.name")}
            </TableHead>
            <TableHead className="w-[140px]">
              {t("schema-editor.partition.type")}
            </TableHead>
            <TableHead className="w-[180px]">
              {t("schema-editor.partition.expression")}
            </TableHead>
            <TableHead className="w-[140px]">
              {t("schema-editor.partition.value")}
            </TableHead>
            {!isReadonly && (
              <TableHead className="w-16 text-right">
                {t("schema-editor.column.operations")}
              </TableHead>
            )}
          </TableRow>
        </TableHeader>
        <TableBody>
          {table.partitions.map((partition, i) => {
            const status = editStatus.getPartitionStatus(db, {
              schema,
              table,
              partition,
            });
            const isFirst = i === 0;
            return (
              <TableRow
                key={`${partition.name}-${i}`}
                className={cn(
                  status === "created" && "text-success",
                  status === "updated" && "text-warning",
                  status === "dropped" && "text-error line-through"
                )}
              >
                <TableCell>
                  <Input
                    value={partition.name}
                    disabled={isReadonly || status === "dropped"}
                    size="sm"
                    className="border-none bg-transparent shadow-none focus-visible:ring-1"
                    onChange={(e) => {
                      partition.name = e.target.value;
                      handlePartitionChange(partition);
                    }}
                  />
                </TableCell>
                <TableCell className="text-sm">
                  {partitionTypeLabel(partition.type)}
                </TableCell>
                <TableCell>
                  <Input
                    value={partition.expression}
                    disabled={isReadonly || !isFirst || status === "dropped"}
                    size="sm"
                    className="border-none bg-transparent shadow-none focus-visible:ring-1"
                    onChange={(e) => {
                      partition.expression = e.target.value;
                      handlePartitionChange(partition);
                    }}
                  />
                </TableCell>
                <TableCell>
                  <Input
                    value={partition.value}
                    disabled={isReadonly || status === "dropped"}
                    size="sm"
                    className="border-none bg-transparent shadow-none focus-visible:ring-1"
                    onChange={(e) => {
                      partition.value = e.target.value;
                      handlePartitionChange(partition);
                    }}
                  />
                </TableCell>
                {!isReadonly && (
                  <TableCell className="text-right">
                    {status === "dropped" ? (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="size-7 p-0"
                        onClick={() => handleRestorePartition(partition)}
                      >
                        <RotateCcw className="size-3.5" />
                      </Button>
                    ) : (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="size-7 p-0 text-error hover:text-error"
                        onClick={() => handleDropPartition(partition)}
                      >
                        <Trash2 className="size-3.5" />
                      </Button>
                    )}
                  </TableCell>
                )}
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}

function partitionTypeLabel(type: TablePartitionMetadata_Type): string {
  switch (type) {
    case TablePartitionMetadata_Type.RANGE:
      return "RANGE";
    case TablePartitionMetadata_Type.RANGE_COLUMNS:
      return "RANGE COLUMNS";
    case TablePartitionMetadata_Type.LIST:
      return "LIST";
    case TablePartitionMetadata_Type.LIST_COLUMNS:
      return "LIST COLUMNS";
    case TablePartitionMetadata_Type.HASH:
      return "HASH";
    default:
      return String(type);
  }
}
