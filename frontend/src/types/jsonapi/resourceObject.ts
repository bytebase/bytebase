/**
 * A representation of a single resource.
 */
export interface ResourceObject {
  type: string;
  id: string;
  // [NOTE] This diverges from the spec a bit to make attributes required to avoid "possibly undefined ts warning"
  attributes: Attributes;
}

/**
 * An array of Resource Objects.
 */
export type ResourceObjects = ResourceObject[];

/**
 * Either or a single Resource Object or an array of Resource Objects.
 */
export type ResourceObjectOrObjects = ResourceObject | ResourceObjects;

/**
 * A ResourceIdentifier identifies and individual resource.
 */
export type ResourceIdentifier<T extends ResourceObject = ResourceObject> =
  Pick<T, "type" | "id">;

/**
 * Attributes describing a Resource Object
 */
export interface Attributes {
  [index: string]: string | number | boolean | object | undefined;
}
