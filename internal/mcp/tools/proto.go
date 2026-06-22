package tools

import "google.golang.org/protobuf/reflect/protoreflect"

// protoMessage is the minimal interface protojson.Marshal needs. We
// avoid pulling in google.golang.org/protobuf/proto.Message at every
// callsite by stating just the method we care about.
type protoMessage interface {
	ProtoReflect() protoreflect.Message
}
