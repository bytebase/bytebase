import { Engine } from "@/types/proto/v1/common";

export const engineHasSchema = (engine: Engine) => {
  return [Engine.POSTGRES].includes(engine);
};
