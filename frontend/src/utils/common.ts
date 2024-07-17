import { unknownUser, emptyUser, type ComposedUser } from "@/types";
import { Environment } from "@/types/proto/v1/environment_service";

export type ResourceType = "USER" | "ENVIRONMENT";

interface ResourceMaker {
  (type: "USER"): ComposedUser;
  (type: "ENVIRONMENT"): Environment;
}

const makeUnknown = (type: ResourceType) => {
  const UNKNOWN_ENVIRONMENT: Environment = Environment.fromPartial({
    title: "<<Unknown environment>>",
  });

  switch (type) {
    case "USER":
      return unknownUser();
    case "ENVIRONMENT":
      return UNKNOWN_ENVIRONMENT;
  }
};

export const unknown = makeUnknown as ResourceMaker;

const makeEmpty = (type: ResourceType) => {
  const EMPTY_ENVIRONMENT: Environment = Environment.fromPartial({});

  switch (type) {
    case "USER":
      return emptyUser();
    case "ENVIRONMENT":
      return EMPTY_ENVIRONMENT;
  }
};

export const empty = makeEmpty as ResourceMaker;
