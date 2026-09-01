package command

import (
	"flag"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// setFieldReplacing is the single routine that applies "replace" semantics for map and list
// fields, a deep-copying Set for message/group fields, and a plain Set for everything else,
// into dst. It is used both when folding the current config file into base and when overriding
// the merged result with env/flag values, so replacement semantics are defined in exactly one
// place rather than duplicated.
//
// No config message in this repository has a map field today, and every repeated field is a
// `repeated string` (inserter.store_address, mutate.fields, agent.servers), so element copying
// here is a value copy and cannot alias a shared submessage; a future repeated-message field
// would need proto.Clone per element before Append.
//
// The message/group arm deep-copies via proto.Clone instead of installing v's submessage by
// reference (08-REVIEW.md IN-01). This matters because ConfigLayerer.fileLayer is now a
// long-lived accumulator: without the clone, dst would hold a pointer into a caller-owned
// message (e.g. a per-call `live` argument), and a later mutation of that caller's message
// would silently corrupt the accumulator it was never supposed to share state with.
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
	case fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind:
		dst.Set(fd, protoreflect.ValueOfMessage(proto.Clone(v.Message().Interface()).ProtoReflect()))
	default:
		dst.Set(fd, v)
	}
}

// ConfigLayerer implements the precedence flags > env vars > config file > proto defaults
// (PCLI-09) across any number of -config-file occurrences, including for message-typed fields.
// Unlike the superseded LayerConfigFile/matchesBase pair (removed in 08-06 once every component
// was migrated onto ConfigLayerer), it does not infer provenance by comparing a candidate value
// against an accumulating baseline — 08-REVIEW.md CR-01 shows that approach cannot distinguish a
// value that was explicitly supplied by an env var or flag from one merely carried over from an
// earlier config file. Instead it records, once, which top-level fields were supplied by an env
// var or by an already-parsed flag, and that record persists across every subsequent
// -config-file so no later file can erase it.
//
// Two documented limitations carried forward from the superseded matchesBase helper:
//  1. An env var whose value happens to equal the factory default is indistinguishable from
//     unset — this is now moot for provenance tracking itself (provenance is recorded from
//     fs.Visit and from a diff against lastResult, not from value comparison against defaults),
//     but a component's own env-var-vs-default ambiguity elsewhere in its config handling is
//     unaffected by this type.
//  2. Proto3 implicit presence makes a zero-valued scalar, false, or a zero-numbered enum
//     invisible to protoreflect.Message.Range — verified: after msg.Set(store, enum 0) the
//     field's Has() is still false. So PROTOCONF_INSERTER_STORE=consul (where consul = 0) cannot
//     override a config file's non-zero value; -store consul is the escape hatch. This
//     limitation is narrowed but not eliminated by flag provenance: a FLAG setting a field to
//     its zero value is now correctly recorded as explicit by markExplicitFlags (fs.Visit sees
//     the flag was parsed regardless of the value it carries), but that value is still invisible
//     to the Range in step 5 that would need to re-apply it from preFile. The flag escape hatch
//     works because flag.Parse writes directly into the live message before LayerConfigFile ever
//     runs, not because the layerer re-applies it.
type ConfigLayerer struct {
	// defaults is a pristine clone of the factory-default snapshot. Never mutated after
	// construction.
	defaults proto.Message
	// fileLayer is the accumulated values of every config file loaded so far, later file
	// winning. Owns its own storage — see setFieldReplacing's message/group arm (IN-01).
	fileLayer proto.Message
	// lastResult is the effective config this layerer produced on its previous call (the
	// factory defaults before the first call). It is the baseline step 2 diffs preFile
	// against to detect newly-explicit env/flag values.
	lastResult proto.Message
	// explicit is the provenance set: top-level field numbers supplied by an env var or by a
	// flag parsed before some -config-file. Entries are never removed.
	explicit map[protoreflect.FieldNumber]bool
	// fs is the component's flag set, or nil (unit tests pass nil).
	fs *flag.FlagSet
}

// NewConfigLayerer constructs a ConfigLayerer for one Command() instance. defaults must be
// proto.Clone(c.config) taken BEFORE lpc.Environment() ran, so the layerer's pristine snapshot
// reflects only the compiled-in factory defaults. fs is the component's *flag.FlagSet (may be
// nil, and unit tests pass nil).
//
// Must be called AFTER lpc.PopulateFlagSet (so fs is fully populated with the component's
// flags) and BEFORE flag.Parse runs (so markExplicitFlags can later observe, via fs.Visit,
// exactly which flags were parsed before each -config-file occurrence).
func NewConfigLayerer(defaults proto.Message, fs *flag.FlagSet) *ConfigLayerer {
	return &ConfigLayerer{
		defaults:   proto.Clone(defaults),
		fileLayer:  defaults.ProtoReflect().New().Interface(),
		lastResult: proto.Clone(defaults),
		explicit:   make(map[protoreflect.FieldNumber]bool),
		fs:         fs,
	}
}

// LayerConfigFile applies a just-loaded config file on top of the live config message while
// preserving flags > env vars > config file > proto defaults (PCLI-09), across any number of
// -config-file occurrences and for message-typed fields.
//
// live is the message that configtool.NewConfig and PopulateFlagSet were built from, and that
// lpc.Unmarshal has just reset to hold only the values the just-loaded config file supplied.
// preFile is proto.Clone(live) taken immediately before lpc.Unmarshal ran for this file — it
// holds the defaults plus everything env vars and any flags parsed before this -config-file
// already supplied.
//
// live is updated in place and is never replaced: the libprotoconf Config and the flag.FlagSet
// built from it hold this exact pointer, and swapping it would orphan every flag parsed after
// -config-file.
//
// The order of the six steps below is load-bearing: recording provenance before folding the
// file is the whole fix.
func (l *ConfigLayerer) LayerConfigFile(live, preFile proto.Message) {
	// 1. Exact flag provenance for flags already parsed.
	l.markExplicitFlags()

	// 2. Absorb env/flag provenance by difference: any field where preFile differs from
	// lastResult (the effective config as of the end of the previous call, or the factory
	// defaults before the first call) was supplied by something this layerer has not already
	// seen — lpc.Environment() on the first call, or a flag parsed since the previous
	// -config-file on a later call. Entries are never removed.
	preFile.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if !fieldEqual(fd, preFile.ProtoReflect(), l.lastResult.ProtoReflect()) {
			l.explicit[fd.Number()] = true
		}
		return true
	})

	// 3. Fold this file into the accumulator. proto.Merge is correct for scalars (and
	// preserves recursive merge semantics for submessages such as tls_config/store_tls), but
	// it APPENDS to repeated fields instead of replacing them, so every map/list field
	// visited on live is corrected immediately after.
	proto.Merge(l.fileLayer, live)
	live.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.IsMap() || fd.IsList() {
			setFieldReplacing(l.fileLayer.ProtoReflect(), fd, v)
		}
		return true
	})

	// 4. Build the result: defaults overlaid with the accumulated file layer. Same
	// map/list-append correction as step 3.
	result := proto.Clone(l.defaults)
	proto.Merge(result, l.fileLayer)
	l.fileLayer.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.IsMap() || fd.IsList() {
			setFieldReplacing(result.ProtoReflect(), fd, v)
		}
		return true
	})

	// 5. Re-apply provenance: this is where env vars and already-parsed flags beat the
	// config file, for every kind of field including message-typed ones.
	preFile.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if l.explicit[fd.Number()] {
			setFieldReplacing(result.ProtoReflect(), fd, v)
		}
		return true
	})

	// 6. Write the result back through the caller's pointer without replacing it. live is
	// empty at this point (it was reset by lpc.Unmarshal and never repopulated), so step 3's
	// append semantics cannot bite.
	proto.Reset(live)
	proto.Merge(live, result)
	l.lastResult = proto.Clone(live)
}

// markExplicitFlags records provenance for every flag parsed strictly before the current
// -config-file occurrence.
//
// flag.(*FlagSet).parseOne records f.actual[name] = flag only AFTER flag.Value.Set(value)
// returns (verified against Go 1.25.1's flag package), so calling fs.Visit from inside the
// -config-file flag.Func callback enumerates exactly the flags parsed strictly before this
// occurrence — not a guess, an exact fact about parse ordering.
//
// Nested message flags (e.g. tls-config-cert-file) are deliberately not mapped to any field
// number: PopulateFlagSet routes a MessageKind field's flags onto a detached dynamicpb message
// that never reaches the component config (library fact 4), so there is no top-level field
// number for them to correspond to. A flag name matching no top-level field (this includes
// "config-file" itself, seen from the second occurrence onward, and every nested message flag)
// is silently ignored.
func (l *ConfigLayerer) markExplicitFlags() {
	if l.fs == nil {
		return
	}
	fields := l.defaults.ProtoReflect().Descriptor().Fields()
	l.fs.Visit(func(f *flag.Flag) {
		for i := 0; i < fields.Len(); i++ {
			fd := fields.Get(i)
			if fd.JSONName() == f.Name {
				l.explicit[fd.Number()] = true
				return
			}
		}
	})
}

// fieldEqual reports whether fd's value is the same on a and b, delegating to proto.Equal by
// building two single-field scratch messages. Comparing one field at a time this way handles
// every field kind — scalar, enum, bytes, list, map, message, repeated message — through one
// code path, so no kind-specific comparison logic is needed; this is what lets ConfigLayerer
// compare message-typed fields (tls_config, store_tls) without the ==-comparability problem
// that forced matchesBase's old composite arm to short-circuit to false.
func fieldEqual(fd protoreflect.FieldDescriptor, a, b protoreflect.Message) bool {
	if a.Has(fd) != b.Has(fd) {
		return false
	}
	if !a.Has(fd) {
		return true
	}
	x := a.New()
	y := b.New()
	x.Set(fd, a.Get(fd))
	y.Set(fd, b.Get(fd))
	return proto.Equal(x.Interface(), y.Interface())
}
