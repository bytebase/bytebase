import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { Engine, State } from "../proto/v1/common";
import { Instance } from "../proto/v1/instance_service";

export const UNKNOWN_INSTANCE_NAME = `instances/${UNKNOWN_ID}`;

export const emptyInstance = () => {
  return Instance.fromJSON({
    name: `instances/${EMPTY_ID}`,
    uid: String(EMPTY_ID),
    state: State.ACTIVE,
    title: "",
    engine: Engine.MYSQL,
  });
};

export const unknownInstance = () => {
  return {
    ...emptyInstance(),
    name: UNKNOWN_INSTANCE_NAME,
    uid: String(UNKNOWN_ID),
    title: "<<Unknown instance>>",
  };
};
