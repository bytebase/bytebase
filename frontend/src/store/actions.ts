type StringRecord = Record<string, string>;

const makeAction = (type: string, namspace?: string) => {
  const mutationType = namspace ? `${namspace}/${type}` : type;
  return ({ commit }: any, ...args: any) => commit(mutationType, ...args);
};

export function makeActions<T>(types: StringRecord = {}): T {
  const actions = {};

  for (const type of Object.keys(types)) {
    const action = {
      [type]: makeAction(types[type]),
    };
    Object.assign(actions, action);
  }

  return actions as T;
}
