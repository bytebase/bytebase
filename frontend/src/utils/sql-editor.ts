import {
  ConnectionAtom,
  ConnectionAtomType,
  Database,
  Instance,
  Table,
} from "@/types";

export const mapConnectionAtom =
  (
    type: ConnectionAtomType,
    parentId: number,
    overrides: Partial<ConnectionAtom> = {}
  ) =>
  (item: Instance | Database | Table) => {
    const connectionAtom: ConnectionAtom = {
      parentId,
      id: item.id,
      key: `${type}-${item.id}`,
      label: item.name,
      type,
      isLeaf: type === "table",
      ...overrides,
    };

    return connectionAtom;
  };
