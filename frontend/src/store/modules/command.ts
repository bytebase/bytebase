import {
  Command,
  CommandID,
  CommandRegisterID,
  CommandState,
} from "../../types";

const state: () => CommandState = () => ({
  commandListByID: new Map(),
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
      registerID,
    }: {
      id: CommandID;
      registerID: CommandRegisterID;
    }
  ) {
    commit("removeCommand", {
      id,
      registerID,
    });
  },

  dispatchCommand({ state }: any, commandID: CommandID) {
    const list = state.commandListByID.get(commandID);
    list?.forEach((cmd: Command) => {
      cmd.run();
    });
  },
};

const mutations = {
  appendCommand(state: CommandState, command: Command) {
    const list = state.commandListByID.get(command.id);
    if (list) {
      const i = list.findIndex(
        (item) => item.registerID === command.registerID
      );
      if (i > -1) {
        throw new Error(
          `'${command.registerID}' has already registered command '${command.id}', this is likely a programming error.`
        );
      }
      list.push(command);
    } else {
      state.commandListByID.set(command.id, [command]);
    }
  },

  removeCommand(
    state: CommandState,
    {
      id,
      registerID,
    }: {
      id: CommandID;
      registerID: CommandRegisterID;
    }
  ) {
    const list = state.commandListByID.get(id);
    if (list) {
      const i = list.findIndex((item) => item.registerID === registerID);
      if (i >= 0) {
        list.splice(i, 1);
        return;
      }
    }
    throw new Error(
      `'${registerID}' attempts to unregister command '${id}' which is not registered before, this is likely a programming error.`
    );
  },
};

export default {
  namespaced: true,
  state,
  actions,
  mutations,
};
