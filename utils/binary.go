package utils

// Adjusted from https://github.com/jhump/protoreflect/blob/d24a736ca94c38e8d70a2de6be6254aa99a5966b/dynamic/binary.go

import (
	"fmt"
	"io"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/jhump/protoreflect/desc"
)

type decoder struct {
	msgDesc *desc.MessageDescriptor
	visitor func(pos int, len int, msgDesc *desc.MessageDescriptor)
}

func (d *decoder) Unmarshal(b []byte) error {
	return d.unmarshal(newCodedBuffer(b), false)
}

func (d *decoder) unmarshal(buf *codedBuffer, isGroup bool) error {
	for !buf.eof() {
		tagNumber, wireType, err := buf.decodeTagAndWireType()
		if err != nil {
			return err
		}
		if wireType == proto.WireEndGroup {
			if isGroup {
				// finished parsing group
				return nil
			}
			return proto.ErrInternalBadWireType
		}
		fd := d.msgDesc.FindFieldByNumber(tagNumber)
		if fd == nil {
			err := unmarshalUnknownField(wireType, buf)
			if err != nil {
				return err
			}
		} else {
			err := d.unmarshalKnownField(fd, wireType, buf)
			if err != nil {
				return err
			}
		}
	}
	if isGroup {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func (d *decoder) unmarshalKnownField(fd *desc.FieldDescriptor, encoding int8, b *codedBuffer) error {
	var err error
	switch encoding {
	case proto.WireFixed32:
		_, err = b.decodeFixed32()
	case proto.WireFixed64:
		_, err = b.decodeFixed64()
	case proto.WireVarint:
		_, err = b.decodeVarint()

	case proto.WireBytes:
		start := b.index
		_, err = b.decodeVarint()
		if err == nil {
			startData := b.index
			b.index = start
			var raw []byte
			raw, err = b.decodeRawBytes(false)
			if err == nil && (fd.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE || fd.GetType() == descriptor.FieldDescriptorProto_TYPE_GROUP) {
				if fd.GetMessageType() == nil {
					return fmt.Errorf("cannot parse field %s from byte-encoded wire type", fd.GetFullyQualifiedName())
				}

				visitor := func(pos int, len int, msgDesc *desc.MessageDescriptor) {
					d.visitor(startData+pos, len, msgDesc)
				}
				newDecoder := &decoder{msgDesc: fd.GetMessageType(), visitor: visitor}
				err = newDecoder.Unmarshal(raw)
				if err == nil {
					d.visitor(startData, len(raw), fd.GetMessageType())
				}
			}
		}

	case proto.WireStartGroup:
		if fd.GetMessageType() == nil {
			return fmt.Errorf("cannot parse field %s from group-encoded wire type", fd.GetFullyQualifiedName())
		}
		newDecoder := &decoder{msgDesc: fd.GetMessageType(), visitor: d.visitor}
		before := b.index
		err = newDecoder.unmarshal(b, true)
		if err == nil {
			d.visitor(before, b.index-before, fd.GetMessageType())
		}
	default:
		return proto.ErrInternalBadWireType
	}
	if err != nil {
		return err
	}

	return nil
}

func unmarshalUnknownField(encoding int8, b *codedBuffer) error {
	var err error
	switch encoding {
	case proto.WireFixed32:
		_, err = b.decodeFixed32()
	case proto.WireFixed64:
		_, err = b.decodeFixed64()
	case proto.WireVarint:
		_, err = b.decodeVarint()
	case proto.WireBytes:
		_, err = b.decodeRawBytes(false)
	case proto.WireStartGroup:
		var groupEnd int
		groupEnd, _, err = skipGroup(b)
		if err == nil {
			b.index = groupEnd
		}
	default:
		err = proto.ErrInternalBadWireType
	}
	if err != nil {
		return err
	}
	return nil
}

func skipGroup(b *codedBuffer) (int, int, error) {
	bs := b.buf
	start := b.index
	defer func() {
		b.index = start
	}()
	for {
		fieldStart := b.index
		// read a field tag
		_, wireType, err := b.decodeTagAndWireType()
		if err != nil {
			return 0, 0, err
		}
		// skip past the field's data
		switch wireType {
		case proto.WireFixed32:
			if !b.skip(4) {
				return 0, 0, io.ErrUnexpectedEOF
			}
		case proto.WireFixed64:
			if !b.skip(8) {
				return 0, 0, io.ErrUnexpectedEOF
			}
		case proto.WireVarint:
			// skip varint by finding last byte (has high bit unset)
			i := b.index
			for {
				if i >= len(bs) {
					return 0, 0, io.ErrUnexpectedEOF
				}
				if bs[i]&0x80 == 0 {
					break
				}
				i++
			}
			b.index++
		case proto.WireBytes:
			l, err := b.decodeVarint()
			if err != nil {
				return 0, 0, err
			}
			lInt := int(l)
			if lInt < 0 {
				return 0, 0, fmt.Errorf("proto: bad byte length %d", lInt)
			}
			if !b.skip(int(l)) {
				return 0, 0, io.ErrUnexpectedEOF
			}
		case proto.WireStartGroup:
			endIndex, _, err := skipGroup(b)
			if err != nil {
				return 0, 0, err
			}
			b.index = endIndex
		case proto.WireEndGroup:
			return b.index, fieldStart, nil
		default:
			return 0, 0, proto.ErrInternalBadWireType
		}
	}
}
