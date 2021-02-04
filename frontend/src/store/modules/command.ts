import {
  Command,
  CommandId,
  CommandRegisterId,
  CommandState,
} from "../../types";

const state: () => CommandState = () => ({
  commandListById: new Map(),
});

const actions = {
  registerCommand({ commit }: any, newCommand: Command) {
    commit("appendCommand", newCommand);
    return newCommand;
  },

  unregisterCommand(
    { commit }: any,
    {
      id,
      registerId,
    }: {
      id: CommandId;
      registerId: CommandRegisterId;
    }
  ) {
    commit("removeCommand", {
      id,
      registerId,
    });
  },

  dispatchCommand({ state }: any, commandId: CommandId) {
    const list = state.commandListById.get(commandId);
    list?.forEach((cmd: Command) => {
      cmd.run();
    });
  },
};

const mutations = {
  appendCommand(state: CommandState, command: Command) {
    const list = state.commandListById.get(command.id);
    if (list) {
      const i = list.findIndex(
        (item) => item.registerId === command.registerId
      );
      if (i > -1) {
        throw new Error(
          `'${command.registerId}' has already registered command '${command.id}', this is likely a programming error.`
        );
      }
      list.push(command);
    } else {
      state.commandListById.set(command.id, [command]);
    }
  },

  removeCommand(
    state: CommandState,
    {
      id,
      registerId,
    }: {
      id: CommandId;
      registerId: CommandRegisterId;
    }
  ) {
    const list = state.commandListById.get(id);
    if (list) {
      const i = list.findIndex((item) => item.registerId === registerId);
      if (i >= 0) {
        list.splice(i, 1);
        return;
      }
    }
    throw new Error(
      `'${registerId}' attempts to unregister command '${id}' which is not registered before, this is likely a programming error.`
    );
  },
};

export default {
  namespaced: true,
  state,
  actions,
  mutations,
};
