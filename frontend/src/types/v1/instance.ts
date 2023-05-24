import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { Engine, State } from "../proto/v1/common";
import { Environment } from "../proto/v1/environment_service";
import { Instance } from "../proto/v1/instance_service";
import { emptyEnvironment, unknownEnvironment } from "./environment";

export const EMPTY_INSTANCE_NAME = `instances/${EMPTY_ID}`;
export const UNKNOWN_INSTANCE_NAME = `instances/${UNKNOWN_ID}`;

export interface ComposedInstance extends Instance {
  environmentEntity: Environment;
}

export const emptyInstance = (): ComposedInstance => {
  const environmentEntity = emptyEnvironment();
  const instance = Instance.fromJSON({
    name: EMPTY_INSTANCE_NAME,
    uid: String(EMPTY_ID),
    state: State.ACTIVE,
    title: "",
    engine: Engine.MYSQL,
    environment: environmentEntity.name,
  });
  return {
    ...instance,
    environmentEntity,
  };
};

export const unknownInstance = (): ComposedInstance => {
  const environmentEntity = unknownEnvironment();
  const instance = {
    ...emptyInstance(),
    name: UNKNOWN_INSTANCE_NAME,
    uid: String(UNKNOWN_ID),
    title: "<<Unknown instance>>",
    environment: environmentEntity.name,
  };
  return {
    ...instance,
    environmentEntity,
  };
};
