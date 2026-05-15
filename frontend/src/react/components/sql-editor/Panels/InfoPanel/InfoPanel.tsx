import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useVueState } from "@/react/hooks/useVueState";
import { DatabaseOverviewInfo } from "@/react/pages/project/database-detail/overview/DatabaseOverviewInfo";
import { useConnectionOfCurrentSQLEditorTab } from "@/react/stores/sqlEditor/tab-vue-state";
import { useDBSchemaV1Store } from "@/store";
import {
  getInstanceResource,
  instanceV1SupportsExternalTable,
  instanceV1SupportsPackage,
  instanceV1SupportsSequence,
} from "@/utils";
import { keyWithPosition } from "@/views/sql-editor/EditorCommon/utils";
import { PanelSearchBox } from "../common/PanelSearchBox";
import { useViewStateNav } from "../common/useViewStateNav";
import { ExternalTablesTable } from "../ExternalTablesPanel/ExternalTablesTable";
import { FunctionsTable } from "../FunctionsPanel/FunctionsTable";
import { PackagesTable } from "../PackagesPanel/PackagesTable";
import { ProceduresTable } from "../ProceduresPanel/ProceduresTable";
import { SequencesTable } from "../SequencesPanel/SequencesTable";
import { TablesTable } from "../TablesPanel/TablesTable";
import { ViewsTable } from "../ViewsPanel/ViewsTable";

export function InfoPanel() {
  const { t } = useTranslation();
  const dbSchemaStore = useDBSchemaV1Store();
  const { database } = useConnectionOfCurrentSQLEditorTab();
  const databaseName = useVueState(() => database.value.name);
  const db = useVueState(() => database.value);
  const databaseMetadata = useVueState(
    () => dbSchemaStore.getDatabaseMetadata(databaseName ?? ""),
    { deep: true }
  );

  const { schema: schemaName, updateViewState } = useViewStateNav();

  const [tables, setTables] = useState("");
  const [views, setViews] = useState("");
  const [functions, setFunctions] = useState("");
  const [procedures, setProcedures] = useState("");
  const [sequences, setSequences] = useState("");
  const [externalTables, setExternalTables] = useState("");
  const [packages, setPackages] = useState("");

  if (!db || !databaseMetadata) return null;
  const schema = databaseMetadata.schemas.find((s) => s.name === schemaName);
  const instance = getInstanceResource(db);
  const showSequences = instanceV1SupportsSequence(instance);
  const showExternalTables = instanceV1SupportsExternalTable(instance);
  const showPackages = instanceV1SupportsPackage(instance);

  return (
    <div className="px-2 py-2 gap-4 h-full overflow-hidden flex flex-col">
      <div className="flex-1 overflow-auto flex flex-col gap-4">
        <DatabaseOverviewInfo database={db} />

        {schema ? (
          <>
            <Section
              title={t("db.tables")}
              keyword={tables}
              onKeywordChange={setTables}
            >
              <div className="max-h-64 overflow-auto">
                <TablesTable
                  db={db}
                  database={databaseMetadata}
                  schema={schema}
                  tables={schema.tables}
                  keyword={tables}
                  onSelect={({ table }) =>
                    updateViewState({
                      view: "TABLES",
                      detail: { table: table.name },
                    })
                  }
                />
              </div>
            </Section>

            <Section
              title={t("db.views")}
              keyword={views}
              onKeywordChange={setViews}
            >
              <div className="max-h-64 overflow-auto">
                <ViewsTable
                  database={databaseMetadata}
                  schema={schema}
                  views={schema.views}
                  keyword={views}
                  onSelect={({ view }) =>
                    updateViewState({
                      view: "VIEWS",
                      detail: { view: view.name },
                    })
                  }
                />
              </div>
            </Section>

            <Section
              title={t("db.functions")}
              keyword={functions}
              onKeywordChange={setFunctions}
            >
              <div className="max-h-64 overflow-auto">
                <FunctionsTable
                  database={databaseMetadata}
                  schema={schema}
                  funcs={schema.functions}
                  keyword={functions}
                  onSelect={({ func, position }) =>
                    updateViewState({
                      view: "FUNCTIONS",
                      detail: {
                        func: keyWithPosition(func.name, position),
                      },
                    })
                  }
                />
              </div>
            </Section>

            <Section
              title={t("db.procedures")}
              keyword={procedures}
              onKeywordChange={setProcedures}
            >
              <div className="max-h-64 overflow-auto">
                <ProceduresTable
                  database={databaseMetadata}
                  schema={schema}
                  procedures={schema.procedures}
                  keyword={procedures}
                  onSelect={({ procedure, position }) =>
                    updateViewState({
                      view: "PROCEDURES",
                      detail: {
                        procedure: keyWithPosition(procedure.name, position),
                      },
                    })
                  }
                />
              </div>
            </Section>

            {showSequences ? (
              <Section
                title={t("db.sequences")}
                keyword={sequences}
                onKeywordChange={setSequences}
              >
                <div className="max-h-64 overflow-auto">
                  <SequencesTable
                    database={databaseMetadata}
                    schema={schema}
                    sequences={schema.sequences}
                    keyword={sequences}
                    onSelect={({ sequence, position }) =>
                      updateViewState({
                        view: "SEQUENCES",
                        detail: {
                          sequence: keyWithPosition(sequence.name, position),
                        },
                      })
                    }
                  />
                </div>
              </Section>
            ) : null}

            {showExternalTables ? (
              <Section
                title={t("db.external-tables")}
                keyword={externalTables}
                onKeywordChange={setExternalTables}
              >
                <div className="max-h-64 overflow-auto">
                  <ExternalTablesTable
                    database={databaseMetadata}
                    schema={schema}
                    externalTables={schema.externalTables}
                    keyword={externalTables}
                    onSelect={({ externalTable }) =>
                      updateViewState({
                        view: "EXTERNAL_TABLES",
                        detail: { externalTable: externalTable.name },
                      })
                    }
                  />
                </div>
              </Section>
            ) : null}

            {showPackages ? (
              <Section
                title={t("db.packages")}
                keyword={packages}
                onKeywordChange={setPackages}
              >
                <div className="max-h-64 overflow-auto">
                  <PackagesTable
                    database={databaseMetadata}
                    schema={schema}
                    packages={schema.packages}
                    keyword={packages}
                    onSelect={({ pack, position }) =>
                      updateViewState({
                        view: "PACKAGES",
                        detail: {
                          package: keyWithPosition(pack.name, position),
                        },
                      })
                    }
                  />
                </div>
              </Section>
            ) : null}
          </>
        ) : null}
      </div>
    </div>
  );
}

function Section({
  title,
  keyword,
  onKeywordChange,
  children,
}: {
  title: string;
  keyword: string;
  onKeywordChange: (value: string) => void;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <h2 className="text-lg">{title}</h2>
        <PanelSearchBox value={keyword} onChange={onKeywordChange} />
      </div>
      {children}
    </div>
  );
}
