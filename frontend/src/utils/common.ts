import { User } from "@/types/proto/v1/auth_service";
import { Environment } from "@/types/proto/v1/environment_service";

export type ResourceType = "USER" | "ENVIRONMENT";

interface ResourceMaker {
  (type: "USER"): User;
  (type: "ENVIRONMENT"): Environment;
}

const makeUnknown = (type: ResourceType) => {
  const UNKNOWN_USER: User = User.fromPartial({
    title: "<<Unknown user>>",
  });

  const UNKNOWN_ENVIRONMENT: Environment = Environment.fromPartial({
    title: "<<Unknown environment>>",
  });

  switch (type) {
    case "USER":
      return UNKNOWN_USER;
    case "ENVIRONMENT":
      return UNKNOWN_ENVIRONMENT;
  }
};

export const unknown = makeUnknown as ResourceMaker;

const makeEmpty = (type: ResourceType) => {
  const EMPTY_USER: User = User.fromPartial({});

  const EMPTY_ENVIRONMENT: Environment = Environment.fromPartial({});

  switch (type) {
    case "USER":
      return EMPTY_USER;
    case "ENVIRONMENT":
      return EMPTY_ENVIRONMENT;
  }
};

export const empty = makeEmpty as ResourceMaker;
