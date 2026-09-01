package command

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// LayerConfigFile applies a just-loaded config file on top of a live config message while
// preserving the precedence flags > env vars > config file > proto defaults (PCLI-09).
//
// live is the message that configtool.NewConfig and PopulateFlagSet were built from, and that
// lpc.Unmarshal has just reset to hold only the values the just-loaded config file supplied.
// base is the factory-default snapshot taken before lpc.Environment() ran; it is mutated in
// place so it accumulates the values of every config file loaded so far, which is what lets a
// later -config-file override an earlier one. preFile is proto.Clone(live) taken immediately
// before lpc.Unmarshal ran for this file — it holds the defaults plus everything env vars and
// any flags parsed before this -config-file already supplied.
//
// live is updated in place and is never replaced: the libprotoconf Config and the flag.FlagSet
// built from it hold this exact pointer, and swapping it would orphan every flag parsed after
// -config-file.
func LayerConfigFile(live, base, preFile proto.Message) {
	// prev is defaults plus every config file loaded before this one, captured before this
	// file is folded into base. Comparing against it (not against base after folding) is what
	// lets matchesBase tell "already explicitly supplied" apart from "just came from this file".
	prev := proto.Clone(base)

	// Fold this file's values into base so a later -config-file overrides an earlier one.
	// proto.Merge is correct for scalars (and preserves recursive merge semantics for
	// submessages such as tls_config/store_tls), but it APPENDS to repeated fields instead of
	// replacing them, so every map/list field visited on live is corrected immediately after.
	proto.Merge(base, live)
	live.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.IsMap() || fd.IsList() {
			setFieldReplacing(base.ProtoReflect(), fd, v)
		}
		return true
	})

	merged := proto.Clone(base)
	mm := merged.ProtoReflect()
	pm := prev.ProtoReflect()

	// Re-apply anything preFile supplied that did NOT come from this file layer (i.e. env vars
	// or flags parsed before -config-file), so those values win over what the file just set.
	preFile.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if matchesBase(fd, v, pm) {
			return true
		}
		setFieldReplacing(mm, fd, v)
		return true
	})

	// Write the result back through the caller's pointer. live is empty at this point (it was
	// reset by lpc.Unmarshal and never repopulated), so step 2's append semantics cannot bite.
	proto.Reset(live)
	proto.Merge(live, merged)
}

// setFieldReplacing is the single routine that applies "replace" semantics for map and list
// fields (and a plain Set for everything else) into dst. It is used both when folding the
// current config file into base and when overriding the merged result with env/flag values, so
// replacement semantics are defined in exactly one place rather than duplicated.
//
// No config message in this repository has a map field today, and every repeated field is a
// `repeated string` (inserter.store_address, mutate.fields, agent.servers), so element copying
// here is a value copy and cannot alias a shared submessage; a future repeated-message field
// would need proto.Clone per element before Append.
func setFieldReplacing(dst protoreflect.Message, fd protoreflect.FieldDescriptor, v protoreflect.Value) {
	switch {
	case fd.IsMap():
		dst.Clear(fd)
		dstMap := dst.Mutable(fd).Map()
		v.Map().Range(func(mk protoreflect.MapKey, mv protoreflect.Value) bool {
			dstMap.Set(mk, mv)
			return true
		})
	case fd.IsList():
		dl := dst.Mutable(fd).List()
		dl.Truncate(0)
		srcList := v.List()
		for i := 0; i < srcList.Len(); i++ {
			dl.Append(srcList.Get(i))
		}
	default:
		dst.Set(fd, v)
	}
}

// matchesBase answers whether fd's value v (as seen on preFile, the pre-file-load snapshot) was
// already supplied by something OTHER than the config file being folded in right now — i.e. it
// came from an env var or an already-parsed flag, and the config-file layer must not overwrite
// it. false means "apply v"; true means "leave the file-derived value alone".
//
// A value equal to the factory default is indistinguishable from unset, and proto3 zero values
// (empty string, false, enum 0) are likewise invisible to Range — verified: after
// msg.Set(store, enum 0) the field's Has() is still false, so an env var setting an enum to its
// zero-numbered value cannot be detected at all. Both are accepted, documented limitations.
func matchesBase(fd protoreflect.FieldDescriptor, v protoreflect.Value, prev protoreflect.Message) bool {
	if !prev.Has(fd) {
		return false
	}
	if fd.IsList() {
		if fd.Message() != nil || fd.Kind() == protoreflect.BytesKind {
			// Element type is not comparable with ==; treat as explicitly supplied.
			return false
		}
		prevList := prev.Get(fd).List()
		vList := v.List()
		if prevList.Len() != vList.Len() {
			return false
		}
		for i := 0; i < vList.Len(); i++ {
			if prevList.Get(i).Interface() != vList.Get(i).Interface() {
				return false
			}
		}
		return true
	}
	if fd.IsMap() || fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind || fd.Kind() == protoreflect.BytesKind {
		// Value.Interface() results are not comparable with ==, and no component sets a
		// composite factory default, so treating these as explicitly supplied is correct.
		return false
	}
	// Covers enums too: Value.Interface() for an enum is a comparable protoreflect.EnumNumber.
	return v.Interface() == prev.Get(fd).Interface()
}
