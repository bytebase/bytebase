import {
  isPagedResponse,
  PagedResponse,
  ResourceIdentifier,
  ResourceObject,
  ResponseWithData,
} from "@/types";

type ConvertEntityFn<T> = (
  data: ResourceObject,
  includedList: ResourceObject[]
) => T;

type ConvertEntityFromIncludedListFn<T> = (
  date:
    | ResourceIdentifier<ResourceObject>
    | ResourceIdentifier<ResourceObject>[]
    | undefined,
  includedList: ResourceObject[]
) => T;

// convert entity list from response data
// works for normal array response data and paged response data
export function convertEntityList<T, K extends string>(
  responseData: ResponseWithData | PagedResponse<K>,
  key: K,
  convert: ConvertEntityFn<T>,
  convertFromIncludedList: ConvertEntityFromIncludedListFn<T>
) {
  if (isPagedResponse(responseData, key)) {
    const resourceIdentifierList =
      responseData.data.relationships?.[key].data ?? [];
    return resourceIdentifierList.map((item) => {
      return convertFromIncludedList(item, responseData.included ?? []);
    });
  }

  const resourceList = responseData.data as ResourceObject[];
  return resourceList.map((obj: ResourceObject) => {
    return convert(obj, responseData.included ?? []);
  });
}
