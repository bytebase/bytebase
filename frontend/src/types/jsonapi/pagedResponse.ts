import { ResourceIdentifier, ResourceObject } from "./resourceObject";
import { ResponseWithData } from "./response";

/**
 * example
  {
    "data": {
      "attributes": { "nextToken": "MTAy" },
      "relationships": {
        "issues": {
          "data": [
            { "type": "issue", "id": "109" },
            { "type": "issue", "id": "108" },
            ...
          ]
        }
      }
    },
    "included": [
      ...
    ]
  }
 */
export type PagedResponse<K extends string> = {
  data: {
    attributes: {
      nextToken: string;
    };
    relationships?: {
      [key in K]: {
        data: ResourceIdentifier[];
      };
    };
  };
  included?: ResourceObject[];
};

export function isPagedResponse<K extends string>(
  responseData: ResponseWithData | PagedResponse<K>,
  key: K
): responseData is PagedResponse<K> {
  const { data } = responseData;
  if (Array.isArray(data)) return false;
  if (typeof data === "object") {
    return (
      typeof data.attributes?.nextToken === "string" &&
      Array.isArray(data.relationships?.[key]?.data)
    );
  }

  return false;
}
