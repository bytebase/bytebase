import { Engine } from "@/types/proto/v1/common";

export const engineSupportsEditIndexes = (engine: Engine) => {
  return [Engine.MYSQL, Engine.TIDB].includes(engine);
};

export const engineSupportsEditTablePartitions = (engine: Engine) => {
  return [Engine.MYSQL, Engine.TIDB].includes(engine);
};

export const engineSupportsEditProcedures = (engine: Engine) => {
  return [Engine.MYSQL, Engine.TIDB].includes(engine);
};

export const engineSupportsEditFunctions = (engine: Engine) => {
  return [Engine.MYSQL, Engine.TIDB].includes(engine);
};
