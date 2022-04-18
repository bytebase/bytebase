import { defineStore } from "pinia";
import { onMounted, onUnmounted } from "vue";
import { Command, CommandId, CommandRegisterId, CommandState } from "@/types";

export const useCommandStore = defineStore("command", {
  state: (): CommandState => ({
    commandListById: new Map(),
  }),
  actions: {
    registerCommand(command: Command) {
      const list = this.commandListById.get(command.id);
      if (list) {
        const index = list.findIndex(
          (item) => item.registerId === command.registerId
        );
        if (index > -1) {
          throw new Error(
            `'${command.registerId}' has already registered command '${command.id}', this is likely a programming error.`
          );
        }
        list.push(command);
      } else {
        this.commandListById.set(command.id, [command]);
      }
      return command;
    },

    unregisterCommand({
      id,
      registerId,
    }: {
      id: CommandId;
      registerId: CommandRegisterId;
    }) {
      const list = this.commandListById.get(id);
      if (list) {
        const index = list.findIndex((item) => item.registerId === registerId);
        if (index >= 0) {
          list.splice(index, 1);
          return;
        }
      }
      throw new Error(
        `'${registerId}' attempts to unregister command '${id}' which is not registered before, this is likely a programming error.`
      );
    },

    dispatchCommand(commandId: CommandId) {
      const list = this.commandListById.get(commandId);
      list?.forEach((cmd: Command) => {
        cmd.run();
      });
    },
  },
});

export const useRegisterCommand = (command: Command) => {
  const store = useCommandStore();
  onMounted(() => {
    store.registerCommand(command);
  });
  onUnmounted(() => {
    store.unregisterCommand(command);
  });
};
