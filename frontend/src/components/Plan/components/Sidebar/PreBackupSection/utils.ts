import { Engine } from "@/types/proto/v1/common";

export const getArchiveDatabase = (engine: Engine): string => {
  if (engine === Engine.ORACLE) {
    return "BBDATAARCHIVE";
  }
  return "bbdataarchive";
};
