/**
 * An index of Links.
 */
export interface Links {
  [index: string]: string | LinkObject;
}

/**
 * A Link.
 */
export interface LinkObject {
  href: string;
  meta: Meta;
}

/**
 * An index of Meta data.
 */
export interface Meta {
  [index: string]: any;
}
