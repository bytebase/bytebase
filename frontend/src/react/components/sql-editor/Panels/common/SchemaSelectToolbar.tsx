import { useTranslation } from "react-i18next";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/react/components/ui/select";
import { useVueState } from "@/react/hooks/useVueState";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/stores/sqlEditor/tab-vue-state";
import { useDBSchemaV1Store } from "@/store";
import { hasSchemaProperty } from "@/utils";
import { useViewStateNav } from "./useViewStateNav";

/**
 * React port of `frontend/src/views/sql-editor/EditorPanel/Panels/common/SchemaSelectToolbar.vue`
 * (`simple` mode — the only mode `Panels.vue` actually uses).
 *
 * Renders nothing for engines without a schema concept (matching Vue's
 * `v-if="showSchemaSelect"`).
 */
export function SchemaSelectToolbar() {
  const { t } = useTranslation();
  const dbSchemaStore = useDBSchemaV1Store();
  const { database, instance } = useConnectionOfCurrentSQLEditorTab();
  const databaseName = useVueState(() => database.value.name);
  const engine = useVueState(() => instance.value.engine);
  const databaseMetadata = useVueState(
    () => dbSchemaStore.getDatabaseMetadata(databaseName ?? ""),
    { deep: true }
  );

  const { schema, setSchema } = useViewStateNav();

  if (engine === undefined || !hasSchemaProperty(engine)) return null;
  if (!databaseMetadata) return null;

  const options = databaseMetadata.schemas.map((s) => ({
    label: s.name || t("db.schema.default"),
    value: s.name,
  }));

  return (
    <Select
      value={schema ?? ""}
      onValueChange={(value) => {
        if (typeof value === "string") setSchema(value);
      }}
    >
      {/* Use `h-8` to align with the sibling `DatabaseChooser` (also h-8);
          the default `size="sm"` trigger is `h-7` and looked shorter. */}
      <SelectTrigger size="sm" className="min-w-32 h-8 shrink-0">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
