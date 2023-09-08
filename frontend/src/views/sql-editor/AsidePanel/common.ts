import {
  idForConnectionAtomItem,
  useConnectionTreeStore,
  useDatabaseV1Store,
  useDBSchemaV1Store,
} from "@/store";
import { ConnectionAtom } from "@/types";

export const fetchDatabaseSubTree = async (atom: ConnectionAtom) => {
  const uid = atom.id;
  const db = useDatabaseV1Store().getDatabaseByUID(uid);
  const databaseMetadata =
    await useDBSchemaV1Store().getOrFetchDatabaseMetadata(db.name);
  const { schemas } = databaseMetadata;
  if (schemas.length === 0) {
    // Empty database
    atom.children = [];
    return;
  }

  if (schemas.length === 1 && schemas[0].name === "") {
    // A single schema database, should render tables directly as a database
    // node's children
    atom.children = schemas[0].tables.map((table) =>
      useConnectionTreeStore().mapAtom(table, "table", atom.id)
    );
    return;
  } else {
    // Multiple schema database
    atom.children = schemas.map((schema) => {
      const id = idForConnectionAtomItem("schema", schema);
      return useConnectionTreeStore().mapAtom(
        schema,
        "schema",
        atom.id,
        schema.tables.map((table) =>
          useConnectionTreeStore().mapAtom(table, "table", id)
        )
      );
    });
    return;
  }
};
