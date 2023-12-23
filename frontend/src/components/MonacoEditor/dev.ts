import { useActuatorV1Store } from "@/store";

export const shouldUseNewLSP = () => {
  return useActuatorV1Store().serverInfo?.lsp;
};
