import { TaskRunLogEntry_Type } from "@/types/proto-es/v1/rollout_service_pb";
import type { FlattenLogEntry } from "../common";

export const detailCellRowSpan = (entry: FlattenLogEntry) => {
  if (
    entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE &&
    entry.commandExecute
  ) {
    const { commandExecute } = entry;
    if (!commandExecute.raw.response) {
      // Not finished yet, combine several rows
      return commandExecute.raw.commandIndexes.length;
    }
    if (commandExecute.raw.response.error) {
      // Error, combine several rows
      return commandExecute.raw.commandIndexes.length;
    }
    if (typeof commandExecute.affectedRows !== "undefined") {
      // Has detailed affectedRows for each command
      return 1;
    }
  }
  return 1;
};
