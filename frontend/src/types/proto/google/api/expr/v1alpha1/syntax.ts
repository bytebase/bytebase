/* eslint-disable */
import * as Long from "long";
import * as _m0 from "protobufjs/minimal";
import { Duration } from "../../../protobuf/duration";
import { NullValue, nullValueFromJSON, nullValueToJSON } from "../../../protobuf/struct";
import { Timestamp } from "../../../protobuf/timestamp";

export const protobufPackage = "google.api.expr.v1alpha1";

/** An expression together with source information as returned by the parser. */
export interface ParsedExpr {
  /** The parsed expression. */
  expr?: Expr;
  /** The source info derived from input that generated the parsed `expr`. */
  sourceInfo?: SourceInfo;
}

/**
 * An abstract representation of a common expression.
 *
 * Expressions are abstractly represented as a collection of identifiers,
 * select statements, function calls, literals, and comprehensions. All
 * operators with the exception of the '.' operator are modelled as function
 * calls. This makes it easy to represent new operators into the existing AST.
 *
 * All references within expressions must resolve to a [Decl][google.api.expr.v1alpha1.Decl] provided at
 * type-check for an expression to be valid. A reference may either be a bare
 * identifier `name` or a qualified identifier `google.api.name`. References
 * may either refer to a value or a function declaration.
 *
 * For example, the expression `google.api.name.startsWith('expr')` references
 * the declaration `google.api.name` within a [Expr.Select][google.api.expr.v1alpha1.Expr.Select] expression, and
 * the function declaration `startsWith`.
 */
export interface Expr {
  /**
   * Required. An id assigned to this node by the parser which is unique in a
   * given expression tree. This is used to associate type information and other
   * attributes to a node in the parse tree.
   */
  id: number;
  /** A literal expression. */
  constExpr?:
    | Constant
    | undefined;
  /** An identifier expression. */
  identExpr?:
    | Expr_Ident
    | undefined;
  /** A field selection expression, e.g. `request.auth`. */
  selectExpr?:
    | Expr_Select
    | undefined;
  /** A call expression, including calls to predefined functions and operators. */
  callExpr?:
    | Expr_Call
    | undefined;
  /** A list creation expression. */
  listExpr?:
    | Expr_CreateList
    | undefined;
  /** A map or message creation expression. */
  structExpr?:
    | Expr_CreateStruct
    | undefined;
  /** A comprehension expression. */
  comprehensionExpr?: Expr_Comprehension | undefined;
}

/** An identifier expression. e.g. `request`. */
export interface Expr_Ident {
  /**
   * Required. Holds a single, unqualified identifier, possibly preceded by a
   * '.'.
   *
   * Qualified names are represented by the [Expr.Select][google.api.expr.v1alpha1.Expr.Select] expression.
   */
  name: string;
}

/** A field selection expression. e.g. `request.auth`. */
export interface Expr_Select {
  /**
   * Required. The target of the selection expression.
   *
   * For example, in the select expression `request.auth`, the `request`
   * portion of the expression is the `operand`.
   */
  operand?: Expr;
  /**
   * Required. The name of the field to select.
   *
   * For example, in the select expression `request.auth`, the `auth` portion
   * of the expression would be the `field`.
   */
  field: string;
  /**
   * Whether the select is to be interpreted as a field presence test.
   *
   * This results from the macro `has(request.auth)`.
   */
  testOnly: boolean;
}

/**
 * A call expression, including calls to predefined functions and operators.
 *
 * For example, `value == 10`, `size(map_value)`.
 */
export interface Expr_Call {
  /**
   * The target of an method call-style expression. For example, `x` in
   * `x.f()`.
   */
  target?: Expr;
  /** Required. The name of the function or method being called. */
  function: string;
  /** The arguments. */
  args: Expr[];
}

/**
 * A list creation expression.
 *
 * Lists may either be homogenous, e.g. `[1, 2, 3]`, or heterogeneous, e.g.
 * `dyn([1, 'hello', 2.0])`
 */
export interface Expr_CreateList {
  /** The elements part of the list. */
  elements: Expr[];
  /**
   * The indices within the elements list which are marked as optional
   * elements.
   *
   * When an optional-typed value is present, the value it contains
   * is included in the list. If the optional-typed value is absent, the list
   * element is omitted from the CreateList result.
   */
  optionalIndices: number[];
}

/**
 * A map or message creation expression.
 *
 * Maps are constructed as `{'key_name': 'value'}`. Message construction is
 * similar, but prefixed with a type name and composed of field ids:
 * `types.MyType{field_id: 'value'}`.
 */
export interface Expr_CreateStruct {
  /**
   * The type name of the message to be created, empty when creating map
   * literals.
   */
  messageName: string;
  /** The entries in the creation expression. */
  entries: Expr_CreateStruct_Entry[];
}

/** Represents an entry. */
export interface Expr_CreateStruct_Entry {
  /**
   * Required. An id assigned to this node by the parser which is unique
   * in a given expression tree. This is used to associate type
   * information and other attributes to the node.
   */
  id: number;
  /** The field key for a message creator statement. */
  fieldKey?:
    | string
    | undefined;
  /** The key expression for a map creation statement. */
  mapKey?:
    | Expr
    | undefined;
  /**
   * Required. The value assigned to the key.
   *
   * If the optional_entry field is true, the expression must resolve to an
   * optional-typed value. If the optional value is present, the key will be
   * set; however, if the optional value is absent, the key will be unset.
   */
  value?: Expr;
  /** Whether the key-value pair is optional. */
  optionalEntry: boolean;
}

/**
 * A comprehension expression applied to a list or map.
 *
 * Comprehensions are not part of the core syntax, but enabled with macros.
 * A macro matches a specific call signature within a parsed AST and replaces
 * the call with an alternate AST block. Macro expansion happens at parse
 * time.
 *
 * The following macros are supported within CEL:
 *
 * Aggregate type macros may be applied to all elements in a list or all keys
 * in a map:
 *
 * *  `all`, `exists`, `exists_one` -  test a predicate expression against
 *    the inputs and return `true` if the predicate is satisfied for all,
 *    any, or only one value `list.all(x, x < 10)`.
 * *  `filter` - test a predicate expression against the inputs and return
 *    the subset of elements which satisfy the predicate:
 *    `payments.filter(p, p > 1000)`.
 * *  `map` - apply an expression to all elements in the input and return the
 *    output aggregate type: `[1, 2, 3].map(i, i * i)`.
 *
 * The `has(m.x)` macro tests whether the property `x` is present in struct
 * `m`. The semantics of this macro depend on the type of `m`. For proto2
 * messages `has(m.x)` is defined as 'defined, but not set`. For proto3, the
 * macro tests whether the property is set to its default. For map and struct
 * types, the macro tests whether the property `x` is defined on `m`.
 */
export interface Expr_Comprehension {
  /** The name of the iteration variable. */
  iterVar: string;
  /** The range over which var iterates. */
  iterRange?: Expr;
  /** The name of the variable used for accumulation of the result. */
  accuVar: string;
  /** The initial value of the accumulator. */
  accuInit?: Expr;
  /**
   * An expression which can contain iter_var and accu_var.
   *
   * Returns false when the result has been computed and may be used as
   * a hint to short-circuit the remainder of the comprehension.
   */
  loopCondition?: Expr;
  /**
   * An expression which can contain iter_var and accu_var.
   *
   * Computes the next value of accu_var.
   */
  loopStep?: Expr;
  /**
   * An expression which can contain accu_var.
   *
   * Computes the result.
   */
  result?: Expr;
}

/**
 * Represents a primitive literal.
 *
 * Named 'Constant' here for backwards compatibility.
 *
 * This is similar as the primitives supported in the well-known type
 * `google.protobuf.Value`, but richer so it can represent CEL's full range of
 * primitives.
 *
 * Lists and structs are not included as constants as these aggregate types may
 * contain [Expr][google.api.expr.v1alpha1.Expr] elements which require evaluation and are thus not constant.
 *
 * Examples of literals include: `"hello"`, `b'bytes'`, `1u`, `4.2`, `-2`,
 * `true`, `null`.
 */
export interface Constant {
  /** null value. */
  nullValue?:
    | NullValue
    | undefined;
  /** boolean value. */
  boolValue?:
    | boolean
    | undefined;
  /** int64 value. */
  int64Value?:
    | number
    | undefined;
  /** uint64 value. */
  uint64Value?:
    | number
    | undefined;
  /** double value. */
  doubleValue?:
    | number
    | undefined;
  /** string value. */
  stringValue?:
    | string
    | undefined;
  /** bytes value. */
  bytesValue?:
    | Uint8Array
    | undefined;
  /**
   * protobuf.Duration value.
   *
   * Deprecated: duration is no longer considered a builtin cel type.
   *
   * @deprecated
   */
  durationValue?:
    | Duration
    | undefined;
  /**
   * protobuf.Timestamp value.
   *
   * Deprecated: timestamp is no longer considered a builtin cel type.
   *
   * @deprecated
   */
  timestampValue?: Date | undefined;
}

/** Source information collected at parse time. */
export interface SourceInfo {
  /** The syntax version of the source, e.g. `cel1`. */
  syntaxVersion: string;
  /**
   * The location name. All position information attached to an expression is
   * relative to this location.
   *
   * The location could be a file, UI element, or similar. For example,
   * `acme/app/AnvilPolicy.cel`.
   */
  location: string;
  /**
   * Monotonically increasing list of code point offsets where newlines
   * `\n` appear.
   *
   * The line number of a given position is the index `i` where for a given
   * `id` the `line_offsets[i] < id_positions[id] < line_offsets[i+1]`. The
   * column may be derivd from `id_positions[id] - line_offsets[i]`.
   */
  lineOffsets: number[];
  /**
   * A map from the parse node id (e.g. `Expr.id`) to the code point offset
   * within the source.
   */
  positions: { [key: number]: number };
  /**
   * A map from the parse node id where a macro replacement was made to the
   * call `Expr` that resulted in a macro expansion.
   *
   * For example, `has(value.field)` is a function call that is replaced by a
   * `test_only` field selection in the AST. Likewise, the call
   * `list.exists(e, e > 10)` translates to a comprehension expression. The key
   * in the map corresponds to the expression id of the expanded macro, and the
   * value is the call `Expr` that was replaced.
   */
  macroCalls: { [key: number]: Expr };
}

export interface SourceInfo_PositionsEntry {
  key: number;
  value: number;
}

export interface SourceInfo_MacroCallsEntry {
  key: number;
  value?: Expr;
}

/** A specific position in source. */
export interface SourcePosition {
  /** The soucre location name (e.g. file name). */
  location: string;
  /** The UTF-8 code unit offset. */
  offset: number;
  /**
   * The 1-based index of the starting line in the source text
   * where the issue occurs, or 0 if unknown.
   */
  line: number;
  /**
   * The 0-based index of the starting position within the line of source text
   * where the issue occurs.  Only meaningful if line is nonzero.
   */
  column: number;
}

function createBaseParsedExpr(): ParsedExpr {
  return { expr: undefined, sourceInfo: undefined };
}

export const ParsedExpr = {
  encode(message: ParsedExpr, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.expr !== undefined) {
      Expr.encode(message.expr, writer.uint32(18).fork()).ldelim();
    }
    if (message.sourceInfo !== undefined) {
      SourceInfo.encode(message.sourceInfo, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): ParsedExpr {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParsedExpr();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 2:
          message.expr = Expr.decode(reader, reader.uint32());
          break;
        case 3:
          message.sourceInfo = SourceInfo.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): ParsedExpr {
    return {
      expr: isSet(object.expr) ? Expr.fromJSON(object.expr) : undefined,
      sourceInfo: isSet(object.sourceInfo) ? SourceInfo.fromJSON(object.sourceInfo) : undefined,
    };
  },

  toJSON(message: ParsedExpr): unknown {
    const obj: any = {};
    message.expr !== undefined && (obj.expr = message.expr ? Expr.toJSON(message.expr) : undefined);
    message.sourceInfo !== undefined &&
      (obj.sourceInfo = message.sourceInfo ? SourceInfo.toJSON(message.sourceInfo) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<ParsedExpr>): ParsedExpr {
    const message = createBaseParsedExpr();
    message.expr = (object.expr !== undefined && object.expr !== null) ? Expr.fromPartial(object.expr) : undefined;
    message.sourceInfo = (object.sourceInfo !== undefined && object.sourceInfo !== null)
      ? SourceInfo.fromPartial(object.sourceInfo)
      : undefined;
    return message;
  },
};

function createBaseExpr(): Expr {
  return {
    id: 0,
    constExpr: undefined,
    identExpr: undefined,
    selectExpr: undefined,
    callExpr: undefined,
    listExpr: undefined,
    structExpr: undefined,
    comprehensionExpr: undefined,
  };
}

export const Expr = {
  encode(message: Expr, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== 0) {
      writer.uint32(16).int64(message.id);
    }
    if (message.constExpr !== undefined) {
      Constant.encode(message.constExpr, writer.uint32(26).fork()).ldelim();
    }
    if (message.identExpr !== undefined) {
      Expr_Ident.encode(message.identExpr, writer.uint32(34).fork()).ldelim();
    }
    if (message.selectExpr !== undefined) {
      Expr_Select.encode(message.selectExpr, writer.uint32(42).fork()).ldelim();
    }
    if (message.callExpr !== undefined) {
      Expr_Call.encode(message.callExpr, writer.uint32(50).fork()).ldelim();
    }
    if (message.listExpr !== undefined) {
      Expr_CreateList.encode(message.listExpr, writer.uint32(58).fork()).ldelim();
    }
    if (message.structExpr !== undefined) {
      Expr_CreateStruct.encode(message.structExpr, writer.uint32(66).fork()).ldelim();
    }
    if (message.comprehensionExpr !== undefined) {
      Expr_Comprehension.encode(message.comprehensionExpr, writer.uint32(74).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Expr {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExpr();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 2:
          message.id = longToNumber(reader.int64() as Long);
          break;
        case 3:
          message.constExpr = Constant.decode(reader, reader.uint32());
          break;
        case 4:
          message.identExpr = Expr_Ident.decode(reader, reader.uint32());
          break;
        case 5:
          message.selectExpr = Expr_Select.decode(reader, reader.uint32());
          break;
        case 6:
          message.callExpr = Expr_Call.decode(reader, reader.uint32());
          break;
        case 7:
          message.listExpr = Expr_CreateList.decode(reader, reader.uint32());
          break;
        case 8:
          message.structExpr = Expr_CreateStruct.decode(reader, reader.uint32());
          break;
        case 9:
          message.comprehensionExpr = Expr_Comprehension.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Expr {
    return {
      id: isSet(object.id) ? Number(object.id) : 0,
      constExpr: isSet(object.constExpr) ? Constant.fromJSON(object.constExpr) : undefined,
      identExpr: isSet(object.identExpr) ? Expr_Ident.fromJSON(object.identExpr) : undefined,
      selectExpr: isSet(object.selectExpr) ? Expr_Select.fromJSON(object.selectExpr) : undefined,
      callExpr: isSet(object.callExpr) ? Expr_Call.fromJSON(object.callExpr) : undefined,
      listExpr: isSet(object.listExpr) ? Expr_CreateList.fromJSON(object.listExpr) : undefined,
      structExpr: isSet(object.structExpr) ? Expr_CreateStruct.fromJSON(object.structExpr) : undefined,
      comprehensionExpr: isSet(object.comprehensionExpr)
        ? Expr_Comprehension.fromJSON(object.comprehensionExpr)
        : undefined,
    };
  },

  toJSON(message: Expr): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = Math.round(message.id));
    message.constExpr !== undefined &&
      (obj.constExpr = message.constExpr ? Constant.toJSON(message.constExpr) : undefined);
    message.identExpr !== undefined &&
      (obj.identExpr = message.identExpr ? Expr_Ident.toJSON(message.identExpr) : undefined);
    message.selectExpr !== undefined &&
      (obj.selectExpr = message.selectExpr ? Expr_Select.toJSON(message.selectExpr) : undefined);
    message.callExpr !== undefined &&
      (obj.callExpr = message.callExpr ? Expr_Call.toJSON(message.callExpr) : undefined);
    message.listExpr !== undefined &&
      (obj.listExpr = message.listExpr ? Expr_CreateList.toJSON(message.listExpr) : undefined);
    message.structExpr !== undefined &&
      (obj.structExpr = message.structExpr ? Expr_CreateStruct.toJSON(message.structExpr) : undefined);
    message.comprehensionExpr !== undefined && (obj.comprehensionExpr = message.comprehensionExpr
      ? Expr_Comprehension.toJSON(message.comprehensionExpr)
      : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<Expr>): Expr {
    const message = createBaseExpr();
    message.id = object.id ?? 0;
    message.constExpr = (object.constExpr !== undefined && object.constExpr !== null)
      ? Constant.fromPartial(object.constExpr)
      : undefined;
    message.identExpr = (object.identExpr !== undefined && object.identExpr !== null)
      ? Expr_Ident.fromPartial(object.identExpr)
      : undefined;
    message.selectExpr = (object.selectExpr !== undefined && object.selectExpr !== null)
      ? Expr_Select.fromPartial(object.selectExpr)
      : undefined;
    message.callExpr = (object.callExpr !== undefined && object.callExpr !== null)
      ? Expr_Call.fromPartial(object.callExpr)
      : undefined;
    message.listExpr = (object.listExpr !== undefined && object.listExpr !== null)
      ? Expr_CreateList.fromPartial(object.listExpr)
      : undefined;
    message.structExpr = (object.structExpr !== undefined && object.structExpr !== null)
      ? Expr_CreateStruct.fromPartial(object.structExpr)
      : undefined;
    message.comprehensionExpr = (object.comprehensionExpr !== undefined && object.comprehensionExpr !== null)
      ? Expr_Comprehension.fromPartial(object.comprehensionExpr)
      : undefined;
    return message;
  },
};

function createBaseExpr_Ident(): Expr_Ident {
  return { name: "" };
}

export const Expr_Ident = {
  encode(message: Expr_Ident, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Expr_Ident {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExpr_Ident();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Expr_Ident {
    return { name: isSet(object.name) ? String(object.name) : "" };
  },

  toJSON(message: Expr_Ident): unknown {
    const obj: any = {};
    message.name !== undefined && (obj.name = message.name);
    return obj;
  },

  fromPartial(object: DeepPartial<Expr_Ident>): Expr_Ident {
    const message = createBaseExpr_Ident();
    message.name = object.name ?? "";
    return message;
  },
};

function createBaseExpr_Select(): Expr_Select {
  return { operand: undefined, field: "", testOnly: false };
}

export const Expr_Select = {
  encode(message: Expr_Select, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.operand !== undefined) {
      Expr.encode(message.operand, writer.uint32(10).fork()).ldelim();
    }
    if (message.field !== "") {
      writer.uint32(18).string(message.field);
    }
    if (message.testOnly === true) {
      writer.uint32(24).bool(message.testOnly);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Expr_Select {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExpr_Select();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.operand = Expr.decode(reader, reader.uint32());
          break;
        case 2:
          message.field = reader.string();
          break;
        case 3:
          message.testOnly = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Expr_Select {
    return {
      operand: isSet(object.operand) ? Expr.fromJSON(object.operand) : undefined,
      field: isSet(object.field) ? String(object.field) : "",
      testOnly: isSet(object.testOnly) ? Boolean(object.testOnly) : false,
    };
  },

  toJSON(message: Expr_Select): unknown {
    const obj: any = {};
    message.operand !== undefined && (obj.operand = message.operand ? Expr.toJSON(message.operand) : undefined);
    message.field !== undefined && (obj.field = message.field);
    message.testOnly !== undefined && (obj.testOnly = message.testOnly);
    return obj;
  },

  fromPartial(object: DeepPartial<Expr_Select>): Expr_Select {
    const message = createBaseExpr_Select();
    message.operand = (object.operand !== undefined && object.operand !== null)
      ? Expr.fromPartial(object.operand)
      : undefined;
    message.field = object.field ?? "";
    message.testOnly = object.testOnly ?? false;
    return message;
  },
};

function createBaseExpr_Call(): Expr_Call {
  return { target: undefined, function: "", args: [] };
}

export const Expr_Call = {
  encode(message: Expr_Call, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.target !== undefined) {
      Expr.encode(message.target, writer.uint32(10).fork()).ldelim();
    }
    if (message.function !== "") {
      writer.uint32(18).string(message.function);
    }
    for (const v of message.args) {
      Expr.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Expr_Call {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExpr_Call();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.target = Expr.decode(reader, reader.uint32());
          break;
        case 2:
          message.function = reader.string();
          break;
        case 3:
          message.args.push(Expr.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Expr_Call {
    return {
      target: isSet(object.target) ? Expr.fromJSON(object.target) : undefined,
      function: isSet(object.function) ? String(object.function) : "",
      args: Array.isArray(object?.args) ? object.args.map((e: any) => Expr.fromJSON(e)) : [],
    };
  },

  toJSON(message: Expr_Call): unknown {
    const obj: any = {};
    message.target !== undefined && (obj.target = message.target ? Expr.toJSON(message.target) : undefined);
    message.function !== undefined && (obj.function = message.function);
    if (message.args) {
      obj.args = message.args.map((e) => e ? Expr.toJSON(e) : undefined);
    } else {
      obj.args = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<Expr_Call>): Expr_Call {
    const message = createBaseExpr_Call();
    message.target = (object.target !== undefined && object.target !== null)
      ? Expr.fromPartial(object.target)
      : undefined;
    message.function = object.function ?? "";
    message.args = object.args?.map((e) => Expr.fromPartial(e)) || [];
    return message;
  },
};

function createBaseExpr_CreateList(): Expr_CreateList {
  return { elements: [], optionalIndices: [] };
}

export const Expr_CreateList = {
  encode(message: Expr_CreateList, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    for (const v of message.elements) {
      Expr.encode(v!, writer.uint32(10).fork()).ldelim();
    }
    writer.uint32(18).fork();
    for (const v of message.optionalIndices) {
      writer.int32(v);
    }
    writer.ldelim();
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Expr_CreateList {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExpr_CreateList();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.elements.push(Expr.decode(reader, reader.uint32()));
          break;
        case 2:
          if ((tag & 7) === 2) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.optionalIndices.push(reader.int32());
            }
          } else {
            message.optionalIndices.push(reader.int32());
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Expr_CreateList {
    return {
      elements: Array.isArray(object?.elements) ? object.elements.map((e: any) => Expr.fromJSON(e)) : [],
      optionalIndices: Array.isArray(object?.optionalIndices) ? object.optionalIndices.map((e: any) => Number(e)) : [],
    };
  },

  toJSON(message: Expr_CreateList): unknown {
    const obj: any = {};
    if (message.elements) {
      obj.elements = message.elements.map((e) => e ? Expr.toJSON(e) : undefined);
    } else {
      obj.elements = [];
    }
    if (message.optionalIndices) {
      obj.optionalIndices = message.optionalIndices.map((e) => Math.round(e));
    } else {
      obj.optionalIndices = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<Expr_CreateList>): Expr_CreateList {
    const message = createBaseExpr_CreateList();
    message.elements = object.elements?.map((e) => Expr.fromPartial(e)) || [];
    message.optionalIndices = object.optionalIndices?.map((e) => e) || [];
    return message;
  },
};

function createBaseExpr_CreateStruct(): Expr_CreateStruct {
  return { messageName: "", entries: [] };
}

export const Expr_CreateStruct = {
  encode(message: Expr_CreateStruct, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.messageName !== "") {
      writer.uint32(10).string(message.messageName);
    }
    for (const v of message.entries) {
      Expr_CreateStruct_Entry.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Expr_CreateStruct {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExpr_CreateStruct();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.messageName = reader.string();
          break;
        case 2:
          message.entries.push(Expr_CreateStruct_Entry.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Expr_CreateStruct {
    return {
      messageName: isSet(object.messageName) ? String(object.messageName) : "",
      entries: Array.isArray(object?.entries)
        ? object.entries.map((e: any) => Expr_CreateStruct_Entry.fromJSON(e))
        : [],
    };
  },

  toJSON(message: Expr_CreateStruct): unknown {
    const obj: any = {};
    message.messageName !== undefined && (obj.messageName = message.messageName);
    if (message.entries) {
      obj.entries = message.entries.map((e) => e ? Expr_CreateStruct_Entry.toJSON(e) : undefined);
    } else {
      obj.entries = [];
    }
    return obj;
  },

  fromPartial(object: DeepPartial<Expr_CreateStruct>): Expr_CreateStruct {
    const message = createBaseExpr_CreateStruct();
    message.messageName = object.messageName ?? "";
    message.entries = object.entries?.map((e) => Expr_CreateStruct_Entry.fromPartial(e)) || [];
    return message;
  },
};

function createBaseExpr_CreateStruct_Entry(): Expr_CreateStruct_Entry {
  return { id: 0, fieldKey: undefined, mapKey: undefined, value: undefined, optionalEntry: false };
}

export const Expr_CreateStruct_Entry = {
  encode(message: Expr_CreateStruct_Entry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.id !== 0) {
      writer.uint32(8).int64(message.id);
    }
    if (message.fieldKey !== undefined) {
      writer.uint32(18).string(message.fieldKey);
    }
    if (message.mapKey !== undefined) {
      Expr.encode(message.mapKey, writer.uint32(26).fork()).ldelim();
    }
    if (message.value !== undefined) {
      Expr.encode(message.value, writer.uint32(34).fork()).ldelim();
    }
    if (message.optionalEntry === true) {
      writer.uint32(40).bool(message.optionalEntry);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Expr_CreateStruct_Entry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExpr_CreateStruct_Entry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = longToNumber(reader.int64() as Long);
          break;
        case 2:
          message.fieldKey = reader.string();
          break;
        case 3:
          message.mapKey = Expr.decode(reader, reader.uint32());
          break;
        case 4:
          message.value = Expr.decode(reader, reader.uint32());
          break;
        case 5:
          message.optionalEntry = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Expr_CreateStruct_Entry {
    return {
      id: isSet(object.id) ? Number(object.id) : 0,
      fieldKey: isSet(object.fieldKey) ? String(object.fieldKey) : undefined,
      mapKey: isSet(object.mapKey) ? Expr.fromJSON(object.mapKey) : undefined,
      value: isSet(object.value) ? Expr.fromJSON(object.value) : undefined,
      optionalEntry: isSet(object.optionalEntry) ? Boolean(object.optionalEntry) : false,
    };
  },

  toJSON(message: Expr_CreateStruct_Entry): unknown {
    const obj: any = {};
    message.id !== undefined && (obj.id = Math.round(message.id));
    message.fieldKey !== undefined && (obj.fieldKey = message.fieldKey);
    message.mapKey !== undefined && (obj.mapKey = message.mapKey ? Expr.toJSON(message.mapKey) : undefined);
    message.value !== undefined && (obj.value = message.value ? Expr.toJSON(message.value) : undefined);
    message.optionalEntry !== undefined && (obj.optionalEntry = message.optionalEntry);
    return obj;
  },

  fromPartial(object: DeepPartial<Expr_CreateStruct_Entry>): Expr_CreateStruct_Entry {
    const message = createBaseExpr_CreateStruct_Entry();
    message.id = object.id ?? 0;
    message.fieldKey = object.fieldKey ?? undefined;
    message.mapKey = (object.mapKey !== undefined && object.mapKey !== null)
      ? Expr.fromPartial(object.mapKey)
      : undefined;
    message.value = (object.value !== undefined && object.value !== null) ? Expr.fromPartial(object.value) : undefined;
    message.optionalEntry = object.optionalEntry ?? false;
    return message;
  },
};

function createBaseExpr_Comprehension(): Expr_Comprehension {
  return {
    iterVar: "",
    iterRange: undefined,
    accuVar: "",
    accuInit: undefined,
    loopCondition: undefined,
    loopStep: undefined,
    result: undefined,
  };
}

export const Expr_Comprehension = {
  encode(message: Expr_Comprehension, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.iterVar !== "") {
      writer.uint32(10).string(message.iterVar);
    }
    if (message.iterRange !== undefined) {
      Expr.encode(message.iterRange, writer.uint32(18).fork()).ldelim();
    }
    if (message.accuVar !== "") {
      writer.uint32(26).string(message.accuVar);
    }
    if (message.accuInit !== undefined) {
      Expr.encode(message.accuInit, writer.uint32(34).fork()).ldelim();
    }
    if (message.loopCondition !== undefined) {
      Expr.encode(message.loopCondition, writer.uint32(42).fork()).ldelim();
    }
    if (message.loopStep !== undefined) {
      Expr.encode(message.loopStep, writer.uint32(50).fork()).ldelim();
    }
    if (message.result !== undefined) {
      Expr.encode(message.result, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Expr_Comprehension {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseExpr_Comprehension();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.iterVar = reader.string();
          break;
        case 2:
          message.iterRange = Expr.decode(reader, reader.uint32());
          break;
        case 3:
          message.accuVar = reader.string();
          break;
        case 4:
          message.accuInit = Expr.decode(reader, reader.uint32());
          break;
        case 5:
          message.loopCondition = Expr.decode(reader, reader.uint32());
          break;
        case 6:
          message.loopStep = Expr.decode(reader, reader.uint32());
          break;
        case 7:
          message.result = Expr.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Expr_Comprehension {
    return {
      iterVar: isSet(object.iterVar) ? String(object.iterVar) : "",
      iterRange: isSet(object.iterRange) ? Expr.fromJSON(object.iterRange) : undefined,
      accuVar: isSet(object.accuVar) ? String(object.accuVar) : "",
      accuInit: isSet(object.accuInit) ? Expr.fromJSON(object.accuInit) : undefined,
      loopCondition: isSet(object.loopCondition) ? Expr.fromJSON(object.loopCondition) : undefined,
      loopStep: isSet(object.loopStep) ? Expr.fromJSON(object.loopStep) : undefined,
      result: isSet(object.result) ? Expr.fromJSON(object.result) : undefined,
    };
  },

  toJSON(message: Expr_Comprehension): unknown {
    const obj: any = {};
    message.iterVar !== undefined && (obj.iterVar = message.iterVar);
    message.iterRange !== undefined && (obj.iterRange = message.iterRange ? Expr.toJSON(message.iterRange) : undefined);
    message.accuVar !== undefined && (obj.accuVar = message.accuVar);
    message.accuInit !== undefined && (obj.accuInit = message.accuInit ? Expr.toJSON(message.accuInit) : undefined);
    message.loopCondition !== undefined &&
      (obj.loopCondition = message.loopCondition ? Expr.toJSON(message.loopCondition) : undefined);
    message.loopStep !== undefined && (obj.loopStep = message.loopStep ? Expr.toJSON(message.loopStep) : undefined);
    message.result !== undefined && (obj.result = message.result ? Expr.toJSON(message.result) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<Expr_Comprehension>): Expr_Comprehension {
    const message = createBaseExpr_Comprehension();
    message.iterVar = object.iterVar ?? "";
    message.iterRange = (object.iterRange !== undefined && object.iterRange !== null)
      ? Expr.fromPartial(object.iterRange)
      : undefined;
    message.accuVar = object.accuVar ?? "";
    message.accuInit = (object.accuInit !== undefined && object.accuInit !== null)
      ? Expr.fromPartial(object.accuInit)
      : undefined;
    message.loopCondition = (object.loopCondition !== undefined && object.loopCondition !== null)
      ? Expr.fromPartial(object.loopCondition)
      : undefined;
    message.loopStep = (object.loopStep !== undefined && object.loopStep !== null)
      ? Expr.fromPartial(object.loopStep)
      : undefined;
    message.result = (object.result !== undefined && object.result !== null)
      ? Expr.fromPartial(object.result)
      : undefined;
    return message;
  },
};

function createBaseConstant(): Constant {
  return {
    nullValue: undefined,
    boolValue: undefined,
    int64Value: undefined,
    uint64Value: undefined,
    doubleValue: undefined,
    stringValue: undefined,
    bytesValue: undefined,
    durationValue: undefined,
    timestampValue: undefined,
  };
}

export const Constant = {
  encode(message: Constant, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.nullValue !== undefined) {
      writer.uint32(8).int32(message.nullValue);
    }
    if (message.boolValue !== undefined) {
      writer.uint32(16).bool(message.boolValue);
    }
    if (message.int64Value !== undefined) {
      writer.uint32(24).int64(message.int64Value);
    }
    if (message.uint64Value !== undefined) {
      writer.uint32(32).uint64(message.uint64Value);
    }
    if (message.doubleValue !== undefined) {
      writer.uint32(41).double(message.doubleValue);
    }
    if (message.stringValue !== undefined) {
      writer.uint32(50).string(message.stringValue);
    }
    if (message.bytesValue !== undefined) {
      writer.uint32(58).bytes(message.bytesValue);
    }
    if (message.durationValue !== undefined) {
      Duration.encode(message.durationValue, writer.uint32(66).fork()).ldelim();
    }
    if (message.timestampValue !== undefined) {
      Timestamp.encode(toTimestamp(message.timestampValue), writer.uint32(74).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Constant {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseConstant();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.nullValue = reader.int32() as any;
          break;
        case 2:
          message.boolValue = reader.bool();
          break;
        case 3:
          message.int64Value = longToNumber(reader.int64() as Long);
          break;
        case 4:
          message.uint64Value = longToNumber(reader.uint64() as Long);
          break;
        case 5:
          message.doubleValue = reader.double();
          break;
        case 6:
          message.stringValue = reader.string();
          break;
        case 7:
          message.bytesValue = reader.bytes();
          break;
        case 8:
          message.durationValue = Duration.decode(reader, reader.uint32());
          break;
        case 9:
          message.timestampValue = fromTimestamp(Timestamp.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): Constant {
    return {
      nullValue: isSet(object.nullValue) ? nullValueFromJSON(object.nullValue) : undefined,
      boolValue: isSet(object.boolValue) ? Boolean(object.boolValue) : undefined,
      int64Value: isSet(object.int64Value) ? Number(object.int64Value) : undefined,
      uint64Value: isSet(object.uint64Value) ? Number(object.uint64Value) : undefined,
      doubleValue: isSet(object.doubleValue) ? Number(object.doubleValue) : undefined,
      stringValue: isSet(object.stringValue) ? String(object.stringValue) : undefined,
      bytesValue: isSet(object.bytesValue) ? bytesFromBase64(object.bytesValue) : undefined,
      durationValue: isSet(object.durationValue) ? Duration.fromJSON(object.durationValue) : undefined,
      timestampValue: isSet(object.timestampValue) ? fromJsonTimestamp(object.timestampValue) : undefined,
    };
  },

  toJSON(message: Constant): unknown {
    const obj: any = {};
    message.nullValue !== undefined &&
      (obj.nullValue = message.nullValue !== undefined ? nullValueToJSON(message.nullValue) : undefined);
    message.boolValue !== undefined && (obj.boolValue = message.boolValue);
    message.int64Value !== undefined && (obj.int64Value = Math.round(message.int64Value));
    message.uint64Value !== undefined && (obj.uint64Value = Math.round(message.uint64Value));
    message.doubleValue !== undefined && (obj.doubleValue = message.doubleValue);
    message.stringValue !== undefined && (obj.stringValue = message.stringValue);
    message.bytesValue !== undefined &&
      (obj.bytesValue = message.bytesValue !== undefined ? base64FromBytes(message.bytesValue) : undefined);
    message.durationValue !== undefined &&
      (obj.durationValue = message.durationValue ? Duration.toJSON(message.durationValue) : undefined);
    message.timestampValue !== undefined && (obj.timestampValue = message.timestampValue.toISOString());
    return obj;
  },

  fromPartial(object: DeepPartial<Constant>): Constant {
    const message = createBaseConstant();
    message.nullValue = object.nullValue ?? undefined;
    message.boolValue = object.boolValue ?? undefined;
    message.int64Value = object.int64Value ?? undefined;
    message.uint64Value = object.uint64Value ?? undefined;
    message.doubleValue = object.doubleValue ?? undefined;
    message.stringValue = object.stringValue ?? undefined;
    message.bytesValue = object.bytesValue ?? undefined;
    message.durationValue = (object.durationValue !== undefined && object.durationValue !== null)
      ? Duration.fromPartial(object.durationValue)
      : undefined;
    message.timestampValue = object.timestampValue ?? undefined;
    return message;
  },
};

function createBaseSourceInfo(): SourceInfo {
  return { syntaxVersion: "", location: "", lineOffsets: [], positions: {}, macroCalls: {} };
}

export const SourceInfo = {
  encode(message: SourceInfo, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.syntaxVersion !== "") {
      writer.uint32(10).string(message.syntaxVersion);
    }
    if (message.location !== "") {
      writer.uint32(18).string(message.location);
    }
    writer.uint32(26).fork();
    for (const v of message.lineOffsets) {
      writer.int32(v);
    }
    writer.ldelim();
    Object.entries(message.positions).forEach(([key, value]) => {
      SourceInfo_PositionsEntry.encode({ key: key as any, value }, writer.uint32(34).fork()).ldelim();
    });
    Object.entries(message.macroCalls).forEach(([key, value]) => {
      SourceInfo_MacroCallsEntry.encode({ key: key as any, value }, writer.uint32(42).fork()).ldelim();
    });
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SourceInfo {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSourceInfo();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.syntaxVersion = reader.string();
          break;
        case 2:
          message.location = reader.string();
          break;
        case 3:
          if ((tag & 7) === 2) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.lineOffsets.push(reader.int32());
            }
          } else {
            message.lineOffsets.push(reader.int32());
          }
          break;
        case 4:
          const entry4 = SourceInfo_PositionsEntry.decode(reader, reader.uint32());
          if (entry4.value !== undefined) {
            message.positions[entry4.key] = entry4.value;
          }
          break;
        case 5:
          const entry5 = SourceInfo_MacroCallsEntry.decode(reader, reader.uint32());
          if (entry5.value !== undefined) {
            message.macroCalls[entry5.key] = entry5.value;
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SourceInfo {
    return {
      syntaxVersion: isSet(object.syntaxVersion) ? String(object.syntaxVersion) : "",
      location: isSet(object.location) ? String(object.location) : "",
      lineOffsets: Array.isArray(object?.lineOffsets) ? object.lineOffsets.map((e: any) => Number(e)) : [],
      positions: isObject(object.positions)
        ? Object.entries(object.positions).reduce<{ [key: number]: number }>((acc, [key, value]) => {
          acc[Number(key)] = Number(value);
          return acc;
        }, {})
        : {},
      macroCalls: isObject(object.macroCalls)
        ? Object.entries(object.macroCalls).reduce<{ [key: number]: Expr }>((acc, [key, value]) => {
          acc[Number(key)] = Expr.fromJSON(value);
          return acc;
        }, {})
        : {},
    };
  },

  toJSON(message: SourceInfo): unknown {
    const obj: any = {};
    message.syntaxVersion !== undefined && (obj.syntaxVersion = message.syntaxVersion);
    message.location !== undefined && (obj.location = message.location);
    if (message.lineOffsets) {
      obj.lineOffsets = message.lineOffsets.map((e) => Math.round(e));
    } else {
      obj.lineOffsets = [];
    }
    obj.positions = {};
    if (message.positions) {
      Object.entries(message.positions).forEach(([k, v]) => {
        obj.positions[k] = Math.round(v);
      });
    }
    obj.macroCalls = {};
    if (message.macroCalls) {
      Object.entries(message.macroCalls).forEach(([k, v]) => {
        obj.macroCalls[k] = Expr.toJSON(v);
      });
    }
    return obj;
  },

  fromPartial(object: DeepPartial<SourceInfo>): SourceInfo {
    const message = createBaseSourceInfo();
    message.syntaxVersion = object.syntaxVersion ?? "";
    message.location = object.location ?? "";
    message.lineOffsets = object.lineOffsets?.map((e) => e) || [];
    message.positions = Object.entries(object.positions ?? {}).reduce<{ [key: number]: number }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[Number(key)] = Number(value);
        }
        return acc;
      },
      {},
    );
    message.macroCalls = Object.entries(object.macroCalls ?? {}).reduce<{ [key: number]: Expr }>(
      (acc, [key, value]) => {
        if (value !== undefined) {
          acc[Number(key)] = Expr.fromPartial(value);
        }
        return acc;
      },
      {},
    );
    return message;
  },
};

function createBaseSourceInfo_PositionsEntry(): SourceInfo_PositionsEntry {
  return { key: 0, value: 0 };
}

export const SourceInfo_PositionsEntry = {
  encode(message: SourceInfo_PositionsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== 0) {
      writer.uint32(8).int64(message.key);
    }
    if (message.value !== 0) {
      writer.uint32(16).int32(message.value);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SourceInfo_PositionsEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSourceInfo_PositionsEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = longToNumber(reader.int64() as Long);
          break;
        case 2:
          message.value = reader.int32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SourceInfo_PositionsEntry {
    return { key: isSet(object.key) ? Number(object.key) : 0, value: isSet(object.value) ? Number(object.value) : 0 };
  },

  toJSON(message: SourceInfo_PositionsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = Math.round(message.key));
    message.value !== undefined && (obj.value = Math.round(message.value));
    return obj;
  },

  fromPartial(object: DeepPartial<SourceInfo_PositionsEntry>): SourceInfo_PositionsEntry {
    const message = createBaseSourceInfo_PositionsEntry();
    message.key = object.key ?? 0;
    message.value = object.value ?? 0;
    return message;
  },
};

function createBaseSourceInfo_MacroCallsEntry(): SourceInfo_MacroCallsEntry {
  return { key: 0, value: undefined };
}

export const SourceInfo_MacroCallsEntry = {
  encode(message: SourceInfo_MacroCallsEntry, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.key !== 0) {
      writer.uint32(8).int64(message.key);
    }
    if (message.value !== undefined) {
      Expr.encode(message.value, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SourceInfo_MacroCallsEntry {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSourceInfo_MacroCallsEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.key = longToNumber(reader.int64() as Long);
          break;
        case 2:
          message.value = Expr.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SourceInfo_MacroCallsEntry {
    return {
      key: isSet(object.key) ? Number(object.key) : 0,
      value: isSet(object.value) ? Expr.fromJSON(object.value) : undefined,
    };
  },

  toJSON(message: SourceInfo_MacroCallsEntry): unknown {
    const obj: any = {};
    message.key !== undefined && (obj.key = Math.round(message.key));
    message.value !== undefined && (obj.value = message.value ? Expr.toJSON(message.value) : undefined);
    return obj;
  },

  fromPartial(object: DeepPartial<SourceInfo_MacroCallsEntry>): SourceInfo_MacroCallsEntry {
    const message = createBaseSourceInfo_MacroCallsEntry();
    message.key = object.key ?? 0;
    message.value = (object.value !== undefined && object.value !== null) ? Expr.fromPartial(object.value) : undefined;
    return message;
  },
};

function createBaseSourcePosition(): SourcePosition {
  return { location: "", offset: 0, line: 0, column: 0 };
}

export const SourcePosition = {
  encode(message: SourcePosition, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.location !== "") {
      writer.uint32(10).string(message.location);
    }
    if (message.offset !== 0) {
      writer.uint32(16).int32(message.offset);
    }
    if (message.line !== 0) {
      writer.uint32(24).int32(message.line);
    }
    if (message.column !== 0) {
      writer.uint32(32).int32(message.column);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): SourcePosition {
    const reader = input instanceof _m0.Reader ? input : new _m0.Reader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSourcePosition();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.location = reader.string();
          break;
        case 2:
          message.offset = reader.int32();
          break;
        case 3:
          message.line = reader.int32();
          break;
        case 4:
          message.column = reader.int32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },

  fromJSON(object: any): SourcePosition {
    return {
      location: isSet(object.location) ? String(object.location) : "",
      offset: isSet(object.offset) ? Number(object.offset) : 0,
      line: isSet(object.line) ? Number(object.line) : 0,
      column: isSet(object.column) ? Number(object.column) : 0,
    };
  },

  toJSON(message: SourcePosition): unknown {
    const obj: any = {};
    message.location !== undefined && (obj.location = message.location);
    message.offset !== undefined && (obj.offset = Math.round(message.offset));
    message.line !== undefined && (obj.line = Math.round(message.line));
    message.column !== undefined && (obj.column = Math.round(message.column));
    return obj;
  },

  fromPartial(object: DeepPartial<SourcePosition>): SourcePosition {
    const message = createBaseSourcePosition();
    message.location = object.location ?? "";
    message.offset = object.offset ?? 0;
    message.line = object.line ?? 0;
    message.column = object.column ?? 0;
    return message;
  },
};

declare var self: any | undefined;
declare var window: any | undefined;
declare var global: any | undefined;
var globalThis: any = (() => {
  if (typeof globalThis !== "undefined") {
    return globalThis;
  }
  if (typeof self !== "undefined") {
    return self;
  }
  if (typeof window !== "undefined") {
    return window;
  }
  if (typeof global !== "undefined") {
    return global;
  }
  throw "Unable to locate global object";
})();

function bytesFromBase64(b64: string): Uint8Array {
  if (globalThis.Buffer) {
    return Uint8Array.from(globalThis.Buffer.from(b64, "base64"));
  } else {
    const bin = globalThis.atob(b64);
    const arr = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; ++i) {
      arr[i] = bin.charCodeAt(i);
    }
    return arr;
  }
}

function base64FromBytes(arr: Uint8Array): string {
  if (globalThis.Buffer) {
    return globalThis.Buffer.from(arr).toString("base64");
  } else {
    const bin: string[] = [];
    arr.forEach((byte) => {
      bin.push(String.fromCharCode(byte));
    });
    return globalThis.btoa(bin.join(""));
  }
}

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function toTimestamp(date: Date): Timestamp {
  const seconds = date.getTime() / 1_000;
  const nanos = (date.getTime() % 1_000) * 1_000_000;
  return { seconds, nanos };
}

function fromTimestamp(t: Timestamp): Date {
  let millis = t.seconds * 1_000;
  millis += t.nanos / 1_000_000;
  return new Date(millis);
}

function fromJsonTimestamp(o: any): Date {
  if (o instanceof Date) {
    return o;
  } else if (typeof o === "string") {
    return new Date(o);
  } else {
    return fromTimestamp(Timestamp.fromJSON(o));
  }
}

function longToNumber(long: Long): number {
  if (long.gt(Number.MAX_SAFE_INTEGER)) {
    throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
  }
  return long.toNumber();
}

// If you get a compile-error about 'Constructor<Long> and ... have no overlap',
// add '--ts_proto_opt=esModuleInterop=true' as a flag when calling 'protoc'.
if (_m0.util.Long !== Long) {
  _m0.util.Long = Long as any;
  _m0.configure();
}

function isObject(value: any): boolean {
  return typeof value === "object" && value !== null;
}

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
