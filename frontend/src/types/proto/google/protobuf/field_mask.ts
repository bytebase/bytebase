/* eslint-disable */
import _m0 from "protobufjs/minimal";

export const protobufPackage = "google.protobuf";

/**
 * `FieldMask` represents a set of symbolic field paths, for example:
 *
 *     paths: "f.a"
 *     paths: "f.b.d"
 *
 * Here `f` represents a field in some root message, `a` and `b`
 * fields in the message found in `f`, and `d` a field found in the
 * message in `f.b`.
 *
 * Field masks are used to specify a subset of fields that should be
 * returned by a get operation or modified by an update operation.
 * Field masks also have a custom JSON encoding (see below).
 *
 * # Field Masks in Projections
 *
 * When used in the context of a projection, a response message or
 * sub-message is filtered by the API to only contain those fields as
 * specified in the mask. For example, if the mask in the previous
 * example is applied to a response message as follows:
 *
 *     f {
 *       a : 22
 *       b {
 *         d : 1
 *         x : 2
 *       }
 *       y : 13
 *     }
 *     z: 8
 *
 * The result will not contain specific values for fields x,y and z
 * (their value will be set to the default, and omitted in proto text
 * output):
 *
 *     f {
 *       a : 22
 *       b {
 *         d : 1
 *       }
 *     }
 *
 * A repeated field is not allowed except at the last position of a
 * paths string.
 *
 * If a FieldMask object is not present in a get operation, the
 * operation applies to all fields (as if a FieldMask of all fields
 * had been specified).
 *
 * Note that a field mask does not necessarily apply to the
 * top-level response message. In case of a REST get operation, the
 * field mask applies directly to the response, but in case of a REST
 * list operation, the mask instead applies to each individual message
 * in the returned resource list. In case of a REST custom method,
 * other definitions may be used. Where the mask applies will be
 * clearly documented together with its declaration in the API.  In
 * any case, the effect on the returned resource/resources is required
 * behavior for APIs.
 *
 * # Field Masks in Update Operations
 *
 * A field mask in update operations specifies which fields of the
 * targeted resource are going to be updated. The API is required
 * to only change the values of the fields as specified in the mask
 * and leave the others untouched. If a resource is passed in to
 * describe the updated values, the API ignores the values of all
 * fields not covered by the mask.
 *
 * If a repeated field is specified for an update operation, new values will
 * be appended to the existing repeated field in the target resource. Note that
 * a repeated field is only allowed in the last position of a `paths` string.
 *
 * If a sub-message is specified in the last position of the field mask for an
 * update operation, then new value will be merged into the existing sub-message
 * in the target resource.
 *
 * For example, given the target message:
 *
 *     f {
 *       b {
 *         d: 1
 *         x: 2
 *       }
 *       c: [1]
 *     }
 *
 * And an update message:
 *
 *     f {
 *       b {
 *         d: 10
 *       }
 *       c: [2]
 *     }
 *
 * then if the field mask is:
 *
 *  paths: ["f.b", "f.c"]
 *
 * then the result will be:
 *
 *     f {
 *       b {
 *         d: 10
 *         x: 2
 *       }
 *       c: [1, 2]
 *     }
 *
 * An implementation may provide options to override this default behavior for
 * repeated and message fields.
 *
 * In order to reset a field's value to the default, the field must
 * be in the mask and set to the default value in the provided resource.
 * Hence, in order to reset all fields of a resource, provide a default
 * instance of the resource and set all fields in the mask, or do
 * not provide a mask as described below.
 *
 * If a field mask is not present on update, the operation applies to
 * all fields (as if a field mask of all fields has been specified).
 * Note that in the presence of schema evolution, this may mean that
 * fields the client does not know and has therefore not filled into
 * the request will be reset to their default. If this is unwanted
 * behavior, a specific service may require a client to always specify
 * a field mask, producing an error if not.
 *
 * As with get operations, the location of the resource which
 * describes the updated values in the request message depends on the
 * operation kind. In any case, the effect of the field mask is
 * required to be honored by the API.
 *
 * ## Considerations for HTTP REST
 *
 * The HTTP kind of an update operation which uses a field mask must
 * be set to PATCH instead of PUT in order to satisfy HTTP semantics
 * (PUT must only be used for full updates).
 *
 * # JSON Encoding of Field Masks
 *
 * In JSON, a field mask is encoded as a single string where paths are
 * separated by a comma. Fields name in each path are converted
 * to/from lower-camel naming conventions.
 *
 * As an example, consider the following message declarations:
 *
 *     message Profile {
 *       User user = 1;
 *       Photo photo = 2;
 *     }
 *     message User {
 *       string display_name = 1;
 *       string address = 2;
 *     }
 *
 * In proto a field mask for `Profile` may look as such:
 *
 *     mask {
 *       paths: "user.display_name"
 *       paths: "photo"
 *     }
 *
 * In JSON, the same mask is represented as below:
 *
 *     {
 *       mask: "user.displayName,photo"
 *     }
 *
 * # Field Masks and Oneof Fields
 *
 * Field masks treat fields in oneofs just as regular fields. Consider the
 * following message:
 *
 *     message SampleMessage {
 *       oneof test_oneof {
 *         string name = 4;
 *         SubMessage sub_message = 9;
 *       }
 *     }
 *
 * The field mask can be:
 *
 *     mask {
 *       paths: "name"
 *     }
 *
 * Or:
 *
 *     mask {
 *       paths: "sub_message"
 *     }
 *
 * Note that oneof type names ("test_oneof" in this case) cannot be used in
 * paths.
 *
 * ## Field Mask Verification
 *
 * The implementation of any API method which has a FieldMask type field in the
 * request should verify the included field paths, and return an
 * `INVALID_ARGUMENT` error if any path is unmappable.
 */
export interface FieldMask {
  /** The set of field mask paths. */
  paths: string[];
}

function createBaseFieldMask(): FieldMask {
  return { paths: [] };
}

export const FieldMask = {
  encode(message: FieldMask, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.paths) {
      writer.uint32(10).string(v!);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): FieldMask {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseFieldMask();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 10) {
            break;
          }

          message.paths.push(reader.string());
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): FieldMask {
    return {
      paths: typeof (object) === "string"
        ? object.split(",").filter(Boolean)
        : Array.isArray(object?.paths)
        ? object.paths.map(String)
        : [],
    };
  },

  toJSON(message: FieldMask): string {
    return message.paths.join(",");
  },

  create(base?: DeepPartial<FieldMask>): FieldMask {
    return FieldMask.fromPartial(base ?? {});
  },

  fromPartial(object: DeepPartial<FieldMask>): FieldMask {
    const message = createBaseFieldMask();
    message.paths = object.paths?.map((e) => e) || [];
    return message;
  },

  wrap(paths: string[]): FieldMask {
    const result = createBaseFieldMask();
    result.paths = paths;
    return result;
  },

  unwrap(message: FieldMask): string[] {
    return message.paths;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;
