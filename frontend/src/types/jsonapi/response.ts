import { ResourceObject, ResourceObjectOrObjects } from "./resourceObject";
import { Links, Meta } from "./shared";

/**
 * A Response for sure containing data.
 */
export interface ResponseWithData<
  D extends ResourceObjectOrObjects = ResourceObjectOrObjects
> {
  data: D;
  included?: ResourceObject[];
  links?: Links;
  errors?: Error[];
  meta?: Meta;
}

/**
 * A Response for sure containing Errors.
 */
export interface ResponseWithErrors<
  D extends ResourceObjectOrObjects = ResourceObjectOrObjects
> {
  data?: D;
  included?: ResourceObject[];
  links?: Links;
  errors: Error[];
  meta?: Meta;
}

/**
 * A Response for sure containing top-level Meta data.
 */
export interface ResponseWithMetaData<
  D extends ResourceObjectOrObjects = ResourceObjectOrObjects
> {
  data?: D;
  included?: ResourceObject[];
  links?: Links;
  errors?: Error[];
  meta: Meta;
}

/**
 * A Response from a JSON API-compliant server.
 */
export interface Response<
  D extends ResourceObjectOrObjects = ResourceObjectOrObjects
> {
  data?: D;
  included?: ResourceObject[];
  links?: Links;
  errors?: Error[];
  meta?: Meta;
}

/**
 * An Error.
 */
export interface Error {
  id?: string;
  links?: Links;
  status?: string;
  code?: string;
  title?: string;
  detail?: string;
  source?: {
    pointer?: string;
    parameter?: string;
  };
  meta?: Meta;
}
