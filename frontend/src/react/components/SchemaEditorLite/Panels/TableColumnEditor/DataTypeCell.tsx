import { useMemo } from "react";
import { Combobox } from "@/react/components/ui/combobox";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { ColumnMetadata } from "@/types/proto-es/v1/database_service_pb";
import { getDataTypeSuggestionList } from "@/utils";

interface Props {
  column: ColumnMetadata;
  engine: Engine;
  readonly: boolean;
  onUpdateValue: (value: string) => void;
}

export function DataTypeCell({
  column,
  engine,
  readonly: isReadonly,
  onUpdateValue,
}: Props) {
  const options = useMemo(() => {
    return getDataTypeSuggestionList(engine).map((dataType) => ({
      label: dataType,
      value: dataType,
    }));
  }, [engine]);

  return (
    <Combobox
      value={column.type || ""}
      onChange={(val) => onUpdateValue(val as string)}
      options={options}
      disabled={isReadonly}
      placeholder="column type"
      className="h-7"
    />
  );
}
