import { Engine } from "@/types/proto/v1/common";

export const engineSupportsEditIndexes = (engine: Engine) => {
  return [Engine.MYSQL, Engine.TIDB].includes(engine);
};

export const engineSupportsEditTablePartitions = (engine: Engine) => {
  return [Engine.MYSQL, Engine.TIDB].includes(engine);
};

export const engineSupportsEditViews = (engine: Engine) => {
  return [Engine.MYSQL, Engine.TIDB].includes(engine);
};

export const engineSupportsEditProcedures = (engine: Engine) => {
  return [Engine.MYSQL, Engine.TIDB].includes(engine);
};

export const engineSupportsEditFunctions = (engine: Engine) => {
  return [Engine.MYSQL, Engine.TIDB].includes(engine);
};

export const engineSupportsMultiSchema = (engine: Engine) => {
  return [Engine.POSTGRES].includes(engine);
};
