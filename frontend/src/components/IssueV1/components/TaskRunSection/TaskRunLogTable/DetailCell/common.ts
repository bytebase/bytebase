import { TaskRunLogEntry_Type } from "@/types/proto-es/v1/rollout_service_pb";
import type { FlattenLogEntry } from "../common";

export const detailCellRowSpan = (entry: FlattenLogEntry) => {
  if (
    entry.type === TaskRunLogEntry_Type.COMMAND_EXECUTE &&
    entry.commandExecute
  ) {
    const { commandExecute } = entry;
    if (commandExecute.kind === "commandIndexes") {
      if (!commandExecute.done) {
        // Not finished yet, combine several rows
        return commandExecute.commandIndexes.length;
      }
      if (commandExecute.error) {
        // Error, combine several rows
        return commandExecute.commandIndexes.length;
      }
    }
  }
  return 1;
};
