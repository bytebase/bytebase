import type { Instance as NewInstance } from "@/types/proto-es/v1/instance_service_pb";
import type { Instance as OldInstance } from "@/types/proto/v1/instance_service";
import { convertNewInstanceToOld, convertOldInstanceToNew } from "@/utils/v1/instance-conversions";
import type { Environment } from "@/types/v1/environment";
import { environmentNamePrefix } from "@/store";
import { EMPTY_ID, UNKNOWN_ID } from "../const";
import { Engine as NewEngine, State as NewState } from "@/types/proto-es/v1/common_pb";
import { unknownEnvironment } from "./environment";
import { create } from "@bufbuild/protobuf";
import { InstanceSchema } from "@/types/proto-es/v1/instance_service_pb";

export const EMPTY_INSTANCE_NAME = `instances/${EMPTY_ID}`;
export const UNKNOWN_INSTANCE_NAME = `instances/${UNKNOWN_ID}`;

// New primary interface (extends proto-es)
export interface ComposedInstanceV2 extends NewInstance {
  environmentEntity: Environment;
}

// Legacy interface for backward compatibility (temporary)
export interface ComposedInstance extends OldInstance {
  environmentEntity: Environment;
}

// Conversion adapters for transition period
export const adaptComposedInstance = {
  fromLegacy: (legacy: ComposedInstance): ComposedInstanceV2 => {
    const newInstance = convertOldInstanceToNew(legacy);
    return {
      ...newInstance,
      environmentEntity: legacy.environmentEntity
    };
  },
  toLegacy: (current: ComposedInstanceV2): ComposedInstance => {
    const oldInstance = convertNewInstanceToOld(current);
    return {
      ...oldInstance,
      environmentEntity: current.environmentEntity
    };
  }
};

// Helper function to create unknown instance with proto-es types
export const unknownInstanceV2 = (): ComposedInstanceV2 => {
  const environmentEntity = unknownEnvironment();
  const instance = create(InstanceSchema, {
    name: UNKNOWN_INSTANCE_NAME,
    state: NewState.ACTIVE,
    title: "<<Unknown instance>>",
    engine: NewEngine.MYSQL,
    environment: `${environmentNamePrefix}${environmentEntity.id}`,
  });
  return {
    ...instance,
    environmentEntity,
  };
};

// Legacy helper for backward compatibility
export const unknownInstance = (): ComposedInstance => {
  return adaptComposedInstance.toLegacy(unknownInstanceV2());
};

export const isValidInstanceName = (name: any): name is string => {
  return (
    typeof name === "string" &&
    name.startsWith("instances/") &&
    name !== EMPTY_INSTANCE_NAME &&
    name !== UNKNOWN_INSTANCE_NAME
  );
};