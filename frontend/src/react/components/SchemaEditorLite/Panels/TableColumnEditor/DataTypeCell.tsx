import { useMemo } from "react";
import { Input } from "@/react/components/ui/input";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import type { ColumnMetadata } from "@/types/proto-es/v1/database_service_pb";
import { getDataTypeSuggestionList } from "@/utils";
import { INLINE_EDIT_INPUT_CLASS } from "../common";

interface Props {
  column: ColumnMetadata;
  engine: Engine;
  readonly: boolean;
  onUpdateValue: (value: string) => void;
  /** Shared id for the host <datalist> element (one per engine). */
  datalistId: string;
}

export function DataTypeCell({
  column,
  readonly: isReadonly,
  onUpdateValue,
  datalistId,
}: Props) {
  return (
    <Input
      list={datalistId}
      value={column.type ?? ""}
      disabled={isReadonly}
      placeholder="column type"
      size="xs"
      className={INLINE_EDIT_INPUT_CLASS}
      onChange={(e) => onUpdateValue(e.target.value)}
    />
  );
}

/**
 * Renders the suggestion datalist that DataTypeCell rows reference via
 * the `list` attribute. Mount once per table.
 */
export function DataTypeSuggestionsDatalist({
  id,
  engine,
}: {
  id: string;
  engine: Engine;
}) {
  const suggestions = useMemo(
    () => getDataTypeSuggestionList(engine),
    [engine]
  );
  return (
    <datalist id={id}>
      {suggestions.map((dataType) => (
        <option key={dataType} value={dataType} />
      ))}
    </datalist>
  );
}
