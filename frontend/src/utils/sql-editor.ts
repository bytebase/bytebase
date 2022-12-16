import { has, uniqueId } from "lodash-es";
import {
  ConnectionAtom,
  ConnectionAtomType,
  Database,
  Instance,
} from "@/types";
import { TableMetadata } from "@/types/proto/database";

export const mapConnectionAtom =
  (
    type: ConnectionAtomType,
    parentId: number,
    overrides: Partial<ConnectionAtom> = {}
  ) =>
  (item: Instance | Database | TableMetadata) => {
    const atomId =
      type !== "table" && has(item, "id") ? (item as any).id : uniqueId();
    const connectionAtom: ConnectionAtom = {
      parentId,
      id: atomId,
      key: `${type}-${atomId}`,
      label: item.name,
      type,
      isLeaf: type === "table",
      ...overrides,
    };
    return connectionAtom;
  };
