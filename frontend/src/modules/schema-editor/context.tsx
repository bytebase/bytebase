import { createContext, useContext } from "react";
import type { SchemaEditorContextValue } from "./types";

const SchemaEditorContext = createContext<SchemaEditorContextValue | null>(
  null
);

export function SchemaEditorProvider({
  value,
  children,
}: {
  value: SchemaEditorContextValue;
  children: React.ReactNode;
}) {
  return (
    <SchemaEditorContext.Provider value={value}>
      {children}
    </SchemaEditorContext.Provider>
  );
}

export function useSchemaEditorContext(): SchemaEditorContextValue {
  const context = useContext(SchemaEditorContext);
  if (!context) {
    throw new Error(
      "useSchemaEditorContext must be used within a SchemaEditorProvider"
    );
  }
  return context;
}
