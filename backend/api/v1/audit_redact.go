package v1

import (
	"log/slog"
	"sync"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"

	spb "google.golang.org/genproto/googleapis/rpc/status"

	"github.com/bytebase/bytebase/backend/common/log"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// Redaction is driven by the (bytebase.v1.audit_behavior) field option rather
// than by a per-RPC redactor. A field marked SENSITIVE or OMIT in the .proto is
// dropped wherever it appears, at any depth, on every request and response the
// audit interceptor records.
//
// It runs from createAuditLog rather than from WrapUnary because the streaming
// path builds its own auditEntry and calls createAuditLog directly, so a walk
// in the unary interceptor would skip AdminExecute — the one streaming RPC, and
// the one carrying every row of an admin-mode query.
//
// It is a denylist, so an unannotated field is logged. The inventory lint in
// audit_redact_inventory_test.go moves that failure to CI.

const anyFullName protoreflect.FullName = "google.protobuf.Any"

// The two Any-carrying fields on an audit row, redacted where they are
// assigned rather than by walking the row.
const (
	auditServiceDataField   protoreflect.FullName = "bytebase.store.AuditLog.service_data"
	auditStatusDetailsField protoreflect.FullName = "google.rpc.Status.details"
)

// auditAnyRegistry names the types each Any field on an audit row may carry.
// An Any whose packed type is not listed for its field is DROPPED rather than
// logged: protojson.Marshal fails the entire row on an unresolvable Any, so
// keeping it would lose the record.
//
// The descriptor walk sees only type_url and value, so a field annotated inside
// a packed message is invisible to the redactor, the coverage sweep and the
// inventory alike. Registering a type is what pulls its fields into the
// inventory for classification.
//
// TestLintAuditAnyFieldsAreRegistered enforces the descriptor half: a new Any
// FIELD reaching the row must be registered. The other half — a new TYPE packed
// into an existing field — changes no descriptor, so nothing fails the build;
// redactPackedAny logs the dropped type at runtime instead.
var auditAnyRegistry = map[protoreflect.FullName][]protoreflect.FullName{
	// UpdateSetting's before-image and the locked re-capture in the
	// workspace-profile merge (setting_service.go:174, :658), plus SetIamPolicy
	// on projects and on the workspace (project_service.go:621,
	// workspace_service.go:509).
	//
	// Setting is the one that carries credentials; convertToSettingMessage
	// blanks its own, but that is a property of the call site rather than an
	// enforced one, which is why it is classified through the inventory like
	// any other field. v1.AuditLog.service_data is deliberately absent: it is
	// annotated OMIT, so SearchAuditLogs' own row does not re-transcribe the
	// before-images of every row the caller read.
	auditServiceDataField: {
		"bytebase.v1.Setting",
		"bytebase.v1.AuditData",
	},
	// Attached to a FAILED RPC by convertErrToStatus — exactly when a handler
	// adds context about what went wrong. PermissionDeniedDetail carries a
	// method, permissions and resource names; PlanCheckRun.Result carries
	// advisory text.
	auditStatusDetailsField: {
		"bytebase.v1.PermissionDeniedDetail",
		"bytebase.v1.PlanCheckRun.Result",
	},
}

// auditAnyPermits reports whether a packed type may be recorded in one Any
// field. An unregistered field permits nothing, so a new Any reaching the row
// before anyone classifies it drops its payload rather than writing it.
func auditAnyPermits(field, packed protoreflect.FullName) bool {
	for _, permitted := range auditAnyRegistry[field] {
		if permitted == packed {
			return true
		}
	}
	return false
}

// auditBehaviorFieldNumber is (bytebase.v1.audit_behavior)'s extension number,
// used to recognize the annotation when it survives only as an unknown field.
const auditBehaviorFieldNumber protowire.Number = 100010

type actionKind uint8

const (
	// actionDrop leaves the field out of the copy entirely.
	actionDrop actionKind = iota + 1
	// actionBlank copies the field with its zero value, keeping presence. Used
	// for SENSITIVE scalars that have explicit presence — a oneof arm or a
	// proto3 optional — so the row still records that a credential was
	// supplied. Clearing DataSourceExternalSecret.token would make token auth
	// indistinguishable from unconfigured while its AppRole sibling stayed
	// legible, and dropping LoginRequest.otp_code would make a failed login
	// with an OTP indistinguishable from one without.
	actionBlank
	// actionPacked resolves and redacts a google.protobuf.Any. The descriptor
	// walk sees only type_url and value, never the packed message.
	actionPacked
	// actionDescend rebuilds a subtree that contains something to redact.
	actionDescend
)

type fieldAction struct {
	kind actionKind
	// sub is the descriptor to descend into, set only for actionDescend. It is
	// a DESCRIPTOR rather than a plan so a recursive type resolves its own plan
	// at every depth: caching a plan value here would freeze a finite chain,
	// and an annotation below its end would be applied at the top and nowhere
	// else.
	sub protoreflect.MessageDescriptor
}

// redactPlan says what to do with one message type. A nil plan means the type
// holds nothing to redact anywhere beneath it, so a value of that type is
// shared by pointer and only read.
type redactPlan struct {
	fields map[protoreflect.FieldNumber]fieldAction
}

// planCache maps a message descriptor to its *redactPlan. Keyed on the
// descriptor itself, not on its full name: plans are applied by field NUMBER,
// so two distinct descriptors sharing a name would otherwise get each other's
// plan. A nil-valued entry is meaningful — it records that the type needs
// nothing.
var planCache sync.Map

// needsCache memoizes typeNeedsRedaction.
var needsCache sync.Map

// planFor returns the plan for a message type, building it on first use.
func planFor(descriptor protoreflect.MessageDescriptor) *redactPlan {
	if cached, ok := planCache.Load(descriptor); ok {
		if plan, isPlan := cached.(*redactPlan); isPlan {
			return plan
		}
	}

	var fields map[protoreflect.FieldNumber]fieldAction
	record := func(field protoreflect.FieldDescriptor, action fieldAction) {
		if fields == nil {
			fields = map[protoreflect.FieldNumber]fieldAction{}
		}
		fields[field.Number()] = action
	}

	declared := descriptor.Fields()
	for i := range declared.Len() {
		field := declared.Get(i)
		switch auditBehaviorOf(field) {
		case v1pb.AuditBehavior_SENSITIVE:
			// A blanked field needs a scalar zero value to stand in for the
			// removed one, and explicit presence for that value to survive
			// marshaling. Everything else is dropped.
			if isBlankable(field) && field.HasPresence() {
				record(field, fieldAction{kind: actionBlank})
			} else {
				record(field, fieldAction{kind: actionDrop})
			}
			continue
		case v1pb.AuditBehavior_OMIT:
			record(field, fieldAction{kind: actionDrop})
			continue
		default:
		}
		sub := submessageOf(field)
		if sub == nil {
			continue
		}
		if sub.FullName() == anyFullName {
			record(field, fieldAction{kind: actionPacked})
			continue
		}
		if typeNeedsRedaction(sub) {
			record(field, fieldAction{kind: actionDescend, sub: sub})
		}
	}

	var plan *redactPlan
	if fields != nil {
		plan = &redactPlan{fields: fields}
	}
	cached, _ := planCache.LoadOrStore(descriptor, plan)
	if stored, isPlan := cached.(*redactPlan); isPlan {
		return stored
	}
	return plan
}

// typeNeedsRedaction reports whether anything reachable from a message type has
// to be redacted. It is plain reachability with a visited set rather than a
// fixed point, which is what makes the four cyclic components in the descriptor
// graph terminate: ObjectSchema/StructKind/ArrayKind,
// google.protobuf.Value/Struct/ListValue, google.api.expr's Expr, and
// TablePartitionMetadata.subpartitions.
func typeNeedsRedaction(descriptor protoreflect.MessageDescriptor) bool {
	if cached, ok := needsCache.Load(descriptor); ok {
		if needs, isBool := cached.(bool); isBool {
			return needs
		}
	}
	needs := reachesRedaction(descriptor, map[protoreflect.MessageDescriptor]bool{})
	needsCache.Store(descriptor, needs)
	return needs
}

func reachesRedaction(descriptor protoreflect.MessageDescriptor, seen map[protoreflect.MessageDescriptor]bool) bool {
	if seen[descriptor] {
		return false
	}
	seen[descriptor] = true

	fields := descriptor.Fields()
	for i := range fields.Len() {
		field := fields.Get(i)
		if auditBehaviorOf(field) != v1pb.AuditBehavior_AUDIT_BEHAVIOR_UNSPECIFIED {
			return true
		}
		sub := submessageOf(field)
		if sub == nil {
			continue
		}
		if sub.FullName() == anyFullName {
			return true
		}
		if reachesRedaction(sub, seen) {
			return true
		}
	}
	return false
}

// submessageOf returns the message descriptor a field leads to: the element
// type of a repeated field, the value type of a map, or the field's own type.
func submessageOf(field protoreflect.FieldDescriptor) protoreflect.MessageDescriptor {
	if field.IsMap() {
		if isMessageKind(field.MapValue().Kind()) {
			return field.MapValue().Message()
		}
		return nil
	}
	if isMessageKind(field.Kind()) {
		return field.Message()
	}
	return nil
}

func isMessageKind(kind protoreflect.Kind) bool {
	return kind == protoreflect.MessageKind || kind == protoreflect.GroupKind
}

func isBlankable(field protoreflect.FieldDescriptor) bool {
	return !field.IsList() && !field.IsMap() && !isMessageKind(field.Kind())
}

// auditBehaviorOf reads the annotation, failing CLOSED. A denylist that cannot
// read its own marker has to assume the field is a credential: the alternative
// is writing it out verbatim with no error and no log line.
func auditBehaviorOf(field protoreflect.FieldDescriptor) v1pb.AuditBehavior {
	options, ok := field.Options().(*descriptorpb.FieldOptions)
	if !ok {
		return v1pb.AuditBehavior_SENSITIVE
	}
	if options == nil {
		return v1pb.AuditBehavior_AUDIT_BEHAVIOR_UNSPECIFIED
	}
	if behavior, ok := proto.GetExtension(options, v1pb.E_AuditBehavior).(v1pb.AuditBehavior); ok &&
		behavior != v1pb.AuditBehavior_AUDIT_BEHAVIOR_UNSPECIFIED {
		return behavior
	}
	// The annotation survives as an UNKNOWN field when the descriptor was
	// unmarshaled by a resolver that does not know this extension — which is
	// reachable, since redactPackedAny resolves types out of a registry that
	// can hold runtime-registered files. Reading UNSPECIFIED there would log
	// the very field the annotation exists to protect.
	if unknownCarriesAuditBehavior(options) {
		return v1pb.AuditBehavior_SENSITIVE
	}
	return v1pb.AuditBehavior_AUDIT_BEHAVIOR_UNSPECIFIED
}

func unknownCarriesAuditBehavior(options *descriptorpb.FieldOptions) bool {
	unknown := options.ProtoReflect().GetUnknown()
	for len(unknown) > 0 {
		number, kind, tagLen := protowire.ConsumeTag(unknown)
		if tagLen < 0 {
			// Unparseable options: fail closed rather than guess.
			return true
		}
		if number == auditBehaviorFieldNumber {
			return true
		}
		valueLen := protowire.ConsumeFieldValue(number, kind, unknown[tagLen:])
		if valueLen < 0 {
			return true
		}
		unknown = unknown[tagLen+valueLen:]
	}
	return false
}

// redactForAudit returns the message to marshal into an audit payload. When
// nothing under the message needs redacting it returns the message itself —
// the common case, and what makes a 5 MB sheet free.
//
// Otherwise it returns a partial copy. Every message on the path to a redacted
// field is copied by Range — never by iterating Descriptor().Fields() and
// calling Set, which panics on an unset message, list or map field. Range
// yields only populated fields, which is also how a dropped field is never
// copied: it is skipped during the copy rather than cleared afterwards on a
// message that never held it. Every subtree with nothing to redact is shared by
// pointer and only read.
//
// Because subtrees are shared, the result is write-once: marshal it and discard
// it. Mutating it would write through to the caller's live message, which the
// handler is still about to return to the client.
func redactForAudit[M proto.Message](message M) M {
	reflected := message.ProtoReflect()
	if !reflected.IsValid() {
		return message
	}
	plan := planFor(reflected.Descriptor())
	if plan == nil {
		return message
	}
	redacted, ok := redactMessage(reflected, plan).Interface().(M)
	if !ok {
		// src.New() returns the same concrete type, so this cannot happen. If
		// a future runtime made it possible, an empty message of the right type
		// is the only safe answer: returning the input would log it unredacted,
		// and returning the zero value would hand a typed nil to callers that
		// dereference it.
		empty, isM := reflected.New().Interface().(M)
		if !isM {
			return message
		}
		return empty
	}
	return redacted
}

// redactMessage copies src, applying plan.
func redactMessage(src protoreflect.Message, plan *redactPlan) protoreflect.Message {
	dst := src.New()
	src.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		action, ok := plan.fields[field.Number()]
		if !ok {
			dst.Set(field, value)
			return true
		}
		switch action.kind {
		case actionDrop:
		case actionBlank:
			dst.Set(field, field.Default())
		case actionPacked:
			copyAnyField(dst, field, value)
		case actionDescend:
			copyDescended(dst, field, value, action.sub)
		default:
		}
		return true
	})
	return dst
}

// copyDescended rebuilds the subtree under one field.
func copyDescended(dst protoreflect.Message, field protoreflect.FieldDescriptor, value protoreflect.Value, sub protoreflect.MessageDescriptor) {
	plan := planFor(sub)
	if plan == nil {
		dst.Set(field, value)
		return
	}
	switch {
	case field.IsMap():
		// dst.Set(field, value) on a map shares the Go map itself and its
		// message values, so redacting an entry would write through to the
		// caller's live message. Rebuild entry by entry.
		out := dst.Mutable(field).Map()
		value.Map().Range(func(key protoreflect.MapKey, entry protoreflect.Value) bool {
			out.Set(key, protoreflect.ValueOfMessage(redactMessage(entry.Message(), plan)))
			return true
		})
	case field.IsList():
		// Sharing the list cannot produce redacted elements without writing
		// through to the caller's, so it is rebuilt per element.
		// Instance.roles[].password and instance_resource.data_sources[] both
		// cross one of these — the first is the leak this design opens with.
		out := dst.Mutable(field).List()
		list := value.List()
		for i := range list.Len() {
			out.Append(protoreflect.ValueOfMessage(redactMessage(list.Get(i).Message(), plan)))
		}
	default:
		dst.Set(field, protoreflect.ValueOfMessage(redactMessage(value.Message(), plan)))
	}
}

// copyAnyField copies an Any-typed field, redacting each packed message and
// dropping any the registry does not permit for this field.
func copyAnyField(dst protoreflect.Message, field protoreflect.FieldDescriptor, value protoreflect.Value) {
	switch {
	case field.IsMap():
		out := dst.Mutable(field).Map()
		value.Map().Range(func(key protoreflect.MapKey, entry protoreflect.Value) bool {
			if redacted, ok := redactPackedAny(field.FullName(), entry.Message()); ok {
				out.Set(key, protoreflect.ValueOfMessage(redacted))
			}
			return true
		})
	case field.IsList():
		out := dst.Mutable(field).List()
		list := value.List()
		for i := range list.Len() {
			if redacted, ok := redactPackedAny(field.FullName(), list.Get(i).Message()); ok {
				out.Append(protoreflect.ValueOfMessage(redacted))
			}
		}
	default:
		if redacted, ok := redactPackedAny(field.FullName(), value.Message()); ok {
			dst.Set(field, protoreflect.ValueOfMessage(redacted))
		}
	}
}

// redactAuditServiceData redacts the Any a handler attached to the row through
// common.SetServiceData — an UpdateSetting before-image, whose Setting carries
// SMTP and AI credentials, or a SetIamPolicy delta. Returns nil when the packed
// type is not one the registry names for this field.
func redactAuditServiceData(serviceData *anypb.Any) *anypb.Any {
	if serviceData == nil {
		return nil
	}
	redacted, ok := redactPackedAny(auditServiceDataField, serviceData.ProtoReflect())
	if !ok {
		return nil
	}
	out, isAny := redacted.Interface().(*anypb.Any)
	if !isAny {
		return nil
	}
	return out
}

// redactAuditStatus rebuilds a failed RPC's status with only the details the
// registry permits.
//
// Not about secrecy: PermissionDeniedDetail and PlanCheckRun.Result carry
// nothing annotated. It is about keeping the row at all — protojson.Marshal
// fails the WHOLE AuditLog on an Any it cannot resolve, so one unrecognized
// detail from a wrapped upstream error would cost the entire record rather
// than just its own payload.
func redactAuditStatus(status *spb.Status) *spb.Status {
	if len(status.GetDetails()) == 0 {
		return status
	}
	details := make([]*anypb.Any, 0, len(status.GetDetails()))
	for _, detail := range status.GetDetails() {
		redacted, ok := redactPackedAny(auditStatusDetailsField, detail.ProtoReflect())
		if !ok {
			continue
		}
		if out, ok := redacted.Interface().(*anypb.Any); ok {
			details = append(details, out)
		}
	}
	if len(details) == len(status.GetDetails()) {
		return status
	}
	// Copied rather than edited: the caller's error is still in flight.
	return &spb.Status{Code: status.GetCode(), Message: status.GetMessage(), Details: details}
}

// redactPackedAny redacts one google.protobuf.Any. The bool reports whether it
// may be recorded at all: an Any this function cannot vouch for is DROPPED, not
// logged, because protojson.Marshal fails the ENTIRE audit row on an
// unresolvable one — keeping it would lose the record rather than the payload.
//
// Vouching means resolving the type through the SAME registry protojson will
// use and parsing the bytes, whether or not there is anything to redact.
// Checking only the types that happen to have a plan would leave the drop
// guarantee off exactly the types that do not — both Status.details types among
// them — so a corrupt payload there would take the whole row down.
//
// Losing a payload silently is its own defect, so a drop is logged.
func redactPackedAny(field protoreflect.FullName, packed protoreflect.Message) (protoreflect.Message, bool) {
	fields := packed.Descriptor().Fields()
	typeURLField := fields.ByName("type_url")
	valueField := fields.ByName("value")
	if typeURLField == nil || valueField == nil {
		return nil, false
	}
	typeURL := packed.Get(typeURLField).String()
	name := packedTypeName(typeURL)
	drop := func(reason string) (protoreflect.Message, bool) {
		slog.Warn("audit: dropped an Any from the audit row",
			slog.String("field", string(field)),
			slog.String("type", string(name)),
			slog.String("reason", reason))
		return nil, false
	}
	if !auditAnyPermits(field, name) {
		return drop("type is not in auditAnyRegistry for this field")
	}

	messageType, err := protoregistry.GlobalTypes.FindMessageByName(name)
	if err != nil {
		return drop("type is not in the global type registry")
	}
	inner := messageType.New().Interface()
	if err := proto.Unmarshal(packed.Get(valueField).Bytes(), inner); err != nil {
		return drop("packed bytes do not parse")
	}

	plan := planFor(messageType.Descriptor())
	if plan == nil {
		// Resolvable and parseable, with nothing to redact. Returned as found
		// so the type_url keeps whatever form the producer wrote: connect's
		// ErrorDetail.Type() returns a bare full name, and a round-trip through
		// anypb.New would rewrite it to type.googleapis.com/... and silently
		// change what SearchAuditLogs emits as @type.
		return packed, true
	}

	body, err := proto.Marshal(redactMessage(inner.ProtoReflect(), plan).Interface())
	if err != nil {
		slog.Warn("audit: dropped an Any from the audit row", log.BBError(err),
			slog.String("field", string(field)), slog.String("type", string(name)),
			slog.String("reason", "redacted message does not marshal"))
		return nil, false
	}
	out := packed.New()
	out.Set(typeURLField, protoreflect.ValueOfString(typeURL))
	out.Set(valueField, protoreflect.ValueOfBytes(body))
	return out, true
}

// packedTypeName strips the optional "type.googleapis.com/" prefix an Any's
// type_url may carry.
func packedTypeName(typeURL string) protoreflect.FullName {
	for i := len(typeURL) - 1; i >= 0; i-- {
		if typeURL[i] == '/' {
			return protoreflect.FullName(typeURL[i+1:])
		}
	}
	return protoreflect.FullName(typeURL)
}
