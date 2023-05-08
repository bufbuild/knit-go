// Copyright 2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        (unknown)
// source: buf/knittest/scalars.proto

package knittest

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Scalars struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	I32       int32              `protobuf:"varint,1,opt,name=i32,proto3" json:"i32,omitempty"`
	I32List   []int32            `protobuf:"varint,2,rep,packed,name=i32_list,json=i32List,proto3" json:"i32_list,omitempty"`
	I32Map    map[int32]int32    `protobuf:"bytes,3,rep,name=i32_map,json=i32Map,proto3" json:"i32_map,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
	I64       int64              `protobuf:"varint,4,opt,name=i64,proto3" json:"i64,omitempty"`
	I64List   []int64            `protobuf:"varint,5,rep,packed,name=i64_list,json=i64List,proto3" json:"i64_list,omitempty"`
	I64Map    map[int64]int64    `protobuf:"bytes,6,rep,name=i64_map,json=i64Map,proto3" json:"i64_map,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
	U32       uint32             `protobuf:"varint,7,opt,name=u32,proto3" json:"u32,omitempty"`
	U32List   []uint32           `protobuf:"varint,8,rep,packed,name=u32_list,json=u32List,proto3" json:"u32_list,omitempty"`
	U32Map    map[uint32]uint32  `protobuf:"bytes,9,rep,name=u32_map,json=u32Map,proto3" json:"u32_map,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
	U64       uint64             `protobuf:"varint,10,opt,name=u64,proto3" json:"u64,omitempty"`
	U64List   []uint64           `protobuf:"varint,11,rep,packed,name=u64_list,json=u64List,proto3" json:"u64_list,omitempty"`
	U64Map    map[uint64]uint64  `protobuf:"bytes,12,rep,name=u64_map,json=u64Map,proto3" json:"u64_map,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
	S32       int32              `protobuf:"zigzag32,13,opt,name=s32,proto3" json:"s32,omitempty"`
	S32List   []int32            `protobuf:"zigzag32,14,rep,packed,name=s32_list,json=s32List,proto3" json:"s32_list,omitempty"`
	S32Map    map[int32]int32    `protobuf:"bytes,15,rep,name=s32_map,json=s32Map,proto3" json:"s32_map,omitempty" protobuf_key:"zigzag32,1,opt,name=key,proto3" protobuf_val:"zigzag32,2,opt,name=value,proto3"`
	S64       int64              `protobuf:"zigzag64,16,opt,name=s64,proto3" json:"s64,omitempty"`
	S64List   []int64            `protobuf:"zigzag64,17,rep,packed,name=s64_list,json=s64List,proto3" json:"s64_list,omitempty"`
	S64Map    map[int64]int64    `protobuf:"bytes,18,rep,name=s64_map,json=s64Map,proto3" json:"s64_map,omitempty" protobuf_key:"zigzag64,1,opt,name=key,proto3" protobuf_val:"zigzag64,2,opt,name=value,proto3"`
	Fx32      uint32             `protobuf:"fixed32,19,opt,name=fx32,proto3" json:"fx32,omitempty"`
	Fx32List  []uint32           `protobuf:"fixed32,20,rep,packed,name=fx32_list,json=fx32List,proto3" json:"fx32_list,omitempty"`
	Fx32Map   map[uint32]uint32  `protobuf:"bytes,21,rep,name=fx32_map,json=fx32Map,proto3" json:"fx32_map,omitempty" protobuf_key:"fixed32,1,opt,name=key,proto3" protobuf_val:"fixed32,2,opt,name=value,proto3"`
	Fx64      uint64             `protobuf:"fixed64,22,opt,name=fx64,proto3" json:"fx64,omitempty"`
	Fx64List  []uint64           `protobuf:"fixed64,23,rep,packed,name=fx64_list,json=fx64List,proto3" json:"fx64_list,omitempty"`
	Fx64Map   map[uint64]uint64  `protobuf:"bytes,24,rep,name=fx64_map,json=fx64Map,proto3" json:"fx64_map,omitempty" protobuf_key:"fixed64,1,opt,name=key,proto3" protobuf_val:"fixed64,2,opt,name=value,proto3"`
	Sfx32     int32              `protobuf:"fixed32,25,opt,name=sfx32,proto3" json:"sfx32,omitempty"`
	Sfx32List []int32            `protobuf:"fixed32,26,rep,packed,name=sfx32_list,json=sfx32List,proto3" json:"sfx32_list,omitempty"`
	Sfx32Map  map[int32]int32    `protobuf:"bytes,27,rep,name=sfx32_map,json=sfx32Map,proto3" json:"sfx32_map,omitempty" protobuf_key:"fixed32,1,opt,name=key,proto3" protobuf_val:"fixed32,2,opt,name=value,proto3"`
	Sfx64     int64              `protobuf:"fixed64,28,opt,name=sfx64,proto3" json:"sfx64,omitempty"`
	Sfx64List []int64            `protobuf:"fixed64,29,rep,packed,name=sfx64_list,json=sfx64List,proto3" json:"sfx64_list,omitempty"`
	Sfx64Map  map[int64]int64    `protobuf:"bytes,30,rep,name=sfx64_map,json=sfx64Map,proto3" json:"sfx64_map,omitempty" protobuf_key:"fixed64,1,opt,name=key,proto3" protobuf_val:"fixed64,2,opt,name=value,proto3"`
	F32       float32            `protobuf:"fixed32,31,opt,name=f32,proto3" json:"f32,omitempty"`
	F32List   []float32          `protobuf:"fixed32,32,rep,packed,name=f32_list,json=f32List,proto3" json:"f32_list,omitempty"`
	F32Map    map[string]float32 `protobuf:"bytes,33,rep,name=f32_map,json=f32Map,proto3" json:"f32_map,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"fixed32,2,opt,name=value,proto3"`
	F64       float64            `protobuf:"fixed64,34,opt,name=f64,proto3" json:"f64,omitempty"`
	F64List   []float64          `protobuf:"fixed64,35,rep,packed,name=f64_list,json=f64List,proto3" json:"f64_list,omitempty"`
	F64Map    map[string]float64 `protobuf:"bytes,36,rep,name=f64_map,json=f64Map,proto3" json:"f64_map,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"fixed64,2,opt,name=value,proto3"`
	Str       string             `protobuf:"bytes,37,opt,name=str,proto3" json:"str,omitempty"`
	StrList   []string           `protobuf:"bytes,38,rep,name=str_list,json=strList,proto3" json:"str_list,omitempty"`
	StrMap    map[string]string  `protobuf:"bytes,39,rep,name=str_map,json=strMap,proto3" json:"str_map,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Byt       []byte             `protobuf:"bytes,40,opt,name=byt,proto3" json:"byt,omitempty"`
	BytList   [][]byte           `protobuf:"bytes,41,rep,name=byt_list,json=bytList,proto3" json:"byt_list,omitempty"`
	BytMap    map[string][]byte  `protobuf:"bytes,42,rep,name=byt_map,json=bytMap,proto3" json:"byt_map,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	B         bool               `protobuf:"varint,43,opt,name=b,proto3" json:"b,omitempty"`
	BList     []bool             `protobuf:"varint,44,rep,packed,name=b_list,json=bList,proto3" json:"b_list,omitempty"`
	BMap      map[bool]bool      `protobuf:"bytes,45,rep,name=b_map,json=bMap,proto3" json:"b_map,omitempty" protobuf_key:"varint,1,opt,name=key,proto3" protobuf_val:"varint,2,opt,name=value,proto3"`
}

func (x *Scalars) Reset() {
	*x = Scalars{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_knittest_scalars_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Scalars) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Scalars) ProtoMessage() {}

func (x *Scalars) ProtoReflect() protoreflect.Message {
	mi := &file_buf_knittest_scalars_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Scalars.ProtoReflect.Descriptor instead.
func (*Scalars) Descriptor() ([]byte, []int) {
	return file_buf_knittest_scalars_proto_rawDescGZIP(), []int{0}
}

func (x *Scalars) GetI32() int32 {
	if x != nil {
		return x.I32
	}
	return 0
}

func (x *Scalars) GetI32List() []int32 {
	if x != nil {
		return x.I32List
	}
	return nil
}

func (x *Scalars) GetI32Map() map[int32]int32 {
	if x != nil {
		return x.I32Map
	}
	return nil
}

func (x *Scalars) GetI64() int64 {
	if x != nil {
		return x.I64
	}
	return 0
}

func (x *Scalars) GetI64List() []int64 {
	if x != nil {
		return x.I64List
	}
	return nil
}

func (x *Scalars) GetI64Map() map[int64]int64 {
	if x != nil {
		return x.I64Map
	}
	return nil
}

func (x *Scalars) GetU32() uint32 {
	if x != nil {
		return x.U32
	}
	return 0
}

func (x *Scalars) GetU32List() []uint32 {
	if x != nil {
		return x.U32List
	}
	return nil
}

func (x *Scalars) GetU32Map() map[uint32]uint32 {
	if x != nil {
		return x.U32Map
	}
	return nil
}

func (x *Scalars) GetU64() uint64 {
	if x != nil {
		return x.U64
	}
	return 0
}

func (x *Scalars) GetU64List() []uint64 {
	if x != nil {
		return x.U64List
	}
	return nil
}

func (x *Scalars) GetU64Map() map[uint64]uint64 {
	if x != nil {
		return x.U64Map
	}
	return nil
}

func (x *Scalars) GetS32() int32 {
	if x != nil {
		return x.S32
	}
	return 0
}

func (x *Scalars) GetS32List() []int32 {
	if x != nil {
		return x.S32List
	}
	return nil
}

func (x *Scalars) GetS32Map() map[int32]int32 {
	if x != nil {
		return x.S32Map
	}
	return nil
}

func (x *Scalars) GetS64() int64 {
	if x != nil {
		return x.S64
	}
	return 0
}

func (x *Scalars) GetS64List() []int64 {
	if x != nil {
		return x.S64List
	}
	return nil
}

func (x *Scalars) GetS64Map() map[int64]int64 {
	if x != nil {
		return x.S64Map
	}
	return nil
}

func (x *Scalars) GetFx32() uint32 {
	if x != nil {
		return x.Fx32
	}
	return 0
}

func (x *Scalars) GetFx32List() []uint32 {
	if x != nil {
		return x.Fx32List
	}
	return nil
}

func (x *Scalars) GetFx32Map() map[uint32]uint32 {
	if x != nil {
		return x.Fx32Map
	}
	return nil
}

func (x *Scalars) GetFx64() uint64 {
	if x != nil {
		return x.Fx64
	}
	return 0
}

func (x *Scalars) GetFx64List() []uint64 {
	if x != nil {
		return x.Fx64List
	}
	return nil
}

func (x *Scalars) GetFx64Map() map[uint64]uint64 {
	if x != nil {
		return x.Fx64Map
	}
	return nil
}

func (x *Scalars) GetSfx32() int32 {
	if x != nil {
		return x.Sfx32
	}
	return 0
}

func (x *Scalars) GetSfx32List() []int32 {
	if x != nil {
		return x.Sfx32List
	}
	return nil
}

func (x *Scalars) GetSfx32Map() map[int32]int32 {
	if x != nil {
		return x.Sfx32Map
	}
	return nil
}

func (x *Scalars) GetSfx64() int64 {
	if x != nil {
		return x.Sfx64
	}
	return 0
}

func (x *Scalars) GetSfx64List() []int64 {
	if x != nil {
		return x.Sfx64List
	}
	return nil
}

func (x *Scalars) GetSfx64Map() map[int64]int64 {
	if x != nil {
		return x.Sfx64Map
	}
	return nil
}

func (x *Scalars) GetF32() float32 {
	if x != nil {
		return x.F32
	}
	return 0
}

func (x *Scalars) GetF32List() []float32 {
	if x != nil {
		return x.F32List
	}
	return nil
}

func (x *Scalars) GetF32Map() map[string]float32 {
	if x != nil {
		return x.F32Map
	}
	return nil
}

func (x *Scalars) GetF64() float64 {
	if x != nil {
		return x.F64
	}
	return 0
}

func (x *Scalars) GetF64List() []float64 {
	if x != nil {
		return x.F64List
	}
	return nil
}

func (x *Scalars) GetF64Map() map[string]float64 {
	if x != nil {
		return x.F64Map
	}
	return nil
}

func (x *Scalars) GetStr() string {
	if x != nil {
		return x.Str
	}
	return ""
}

func (x *Scalars) GetStrList() []string {
	if x != nil {
		return x.StrList
	}
	return nil
}

func (x *Scalars) GetStrMap() map[string]string {
	if x != nil {
		return x.StrMap
	}
	return nil
}

func (x *Scalars) GetByt() []byte {
	if x != nil {
		return x.Byt
	}
	return nil
}

func (x *Scalars) GetBytList() [][]byte {
	if x != nil {
		return x.BytList
	}
	return nil
}

func (x *Scalars) GetBytMap() map[string][]byte {
	if x != nil {
		return x.BytMap
	}
	return nil
}

func (x *Scalars) GetB() bool {
	if x != nil {
		return x.B
	}
	return false
}

func (x *Scalars) GetBList() []bool {
	if x != nil {
		return x.BList
	}
	return nil
}

func (x *Scalars) GetBMap() map[bool]bool {
	if x != nil {
		return x.BMap
	}
	return nil
}

var File_buf_knittest_scalars_proto protoreflect.FileDescriptor

var file_buf_knittest_scalars_proto_rawDesc = []byte{
	0x0a, 0x1a, 0x62, 0x75, 0x66, 0x2f, 0x6b, 0x6e, 0x69, 0x74, 0x74, 0x65, 0x73, 0x74, 0x2f, 0x73,
	0x63, 0x61, 0x6c, 0x61, 0x72, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0c, 0x62, 0x75,
	0x66, 0x2e, 0x6b, 0x6e, 0x69, 0x74, 0x74, 0x65, 0x73, 0x74, 0x22, 0xc5, 0x13, 0x0a, 0x07, 0x53,
	0x63, 0x61, 0x6c, 0x61, 0x72, 0x73, 0x12, 0x10, 0x0a, 0x03, 0x69, 0x33, 0x32, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x05, 0x52, 0x03, 0x69, 0x33, 0x32, 0x12, 0x19, 0x0a, 0x08, 0x69, 0x33, 0x32, 0x5f,
	0x6c, 0x69, 0x73, 0x74, 0x18, 0x02, 0x20, 0x03, 0x28, 0x05, 0x52, 0x07, 0x69, 0x33, 0x32, 0x4c,
	0x69, 0x73, 0x74, 0x12, 0x3a, 0x0a, 0x07, 0x69, 0x33, 0x32, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x03,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x6b, 0x6e, 0x69, 0x74, 0x74,
	0x65, 0x73, 0x74, 0x2e, 0x53, 0x63, 0x61, 0x6c, 0x61, 0x72, 0x73, 0x2e, 0x49, 0x33, 0x32, 0x4d,
	0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x69, 0x33, 0x32, 0x4d, 0x61, 0x70, 0x12,
	0x10, 0x0a, 0x03, 0x69, 0x36, 0x34, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52, 0x03, 0x69, 0x36,
	0x34, 0x12, 0x19, 0x0a, 0x08, 0x69, 0x36, 0x34, 0x5f, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x05, 0x20,
	0x03, 0x28, 0x03, 0x52, 0x07, 0x69, 0x36, 0x34, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x3a, 0x0a, 0x07,
	0x69, 0x36, 0x34, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x06, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x21, 0x2e,
	0x62, 0x75, 0x66, 0x2e, 0x6b, 0x6e, 0x69, 0x74, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x63, 0x61,
	0x6c, 0x61, 0x72, 0x73, 0x2e, 0x49, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x52, 0x06, 0x69, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x33, 0x32, 0x18,
	0x07, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x03, 0x75, 0x33, 0x32, 0x12, 0x19, 0x0a, 0x08, 0x75, 0x33,
	0x32, 0x5f, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x08, 0x20, 0x03, 0x28, 0x0d, 0x52, 0x07, 0x75, 0x33,
	0x32, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x3a, 0x0a, 0x07, 0x75, 0x33, 0x32, 0x5f, 0x6d, 0x61, 0x70,
	0x18, 0x09, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x6b, 0x6e, 0x69,
	0x74, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x63, 0x61, 0x6c, 0x61, 0x72, 0x73, 0x2e, 0x55, 0x33,
	0x32, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x75, 0x33, 0x32, 0x4d, 0x61,
	0x70, 0x12, 0x10, 0x0a, 0x03, 0x75, 0x36, 0x34, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x04, 0x52, 0x03,
	0x75, 0x36, 0x34, 0x12, 0x19, 0x0a, 0x08, 0x75, 0x36, 0x34, 0x5f, 0x6c, 0x69, 0x73, 0x74, 0x18,
	0x0b, 0x20, 0x03, 0x28, 0x04, 0x52, 0x07, 0x75, 0x36, 0x34, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x3a,
	0x0a, 0x07, 0x75, 0x36, 0x34, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x0c, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x21, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x6b, 0x6e, 0x69, 0x74, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53,
	0x63, 0x61, 0x6c, 0x61, 0x72, 0x73, 0x2e, 0x55, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74,
	0x72, 0x79, 0x52, 0x06, 0x75, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x12, 0x10, 0x0a, 0x03, 0x73, 0x33,
	0x32, 0x18, 0x0d, 0x20, 0x01, 0x28, 0x11, 0x52, 0x03, 0x73, 0x33, 0x32, 0x12, 0x19, 0x0a, 0x08,
	0x73, 0x33, 0x32, 0x5f, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x0e, 0x20, 0x03, 0x28, 0x11, 0x52, 0x07,
	0x73, 0x33, 0x32, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x3a, 0x0a, 0x07, 0x73, 0x33, 0x32, 0x5f, 0x6d,
	0x61, 0x70, 0x18, 0x0f, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x6b,
	0x6e, 0x69, 0x74, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x63, 0x61, 0x6c, 0x61, 0x72, 0x73, 0x2e,
	0x53, 0x33, 0x32, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x73, 0x33, 0x32,
	0x4d, 0x61, 0x70, 0x12, 0x10, 0x0a, 0x03, 0x73, 0x36, 0x34, 0x18, 0x10, 0x20, 0x01, 0x28, 0x12,
	0x52, 0x03, 0x73, 0x36, 0x34, 0x12, 0x19, 0x0a, 0x08, 0x73, 0x36, 0x34, 0x5f, 0x6c, 0x69, 0x73,
	0x74, 0x18, 0x11, 0x20, 0x03, 0x28, 0x12, 0x52, 0x07, 0x73, 0x36, 0x34, 0x4c, 0x69, 0x73, 0x74,
	0x12, 0x3a, 0x0a, 0x07, 0x73, 0x36, 0x34, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x12, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x21, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x6b, 0x6e, 0x69, 0x74, 0x74, 0x65, 0x73, 0x74,
	0x2e, 0x53, 0x63, 0x61, 0x6c, 0x61, 0x72, 0x73, 0x2e, 0x53, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x45,
	0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x73, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x12, 0x12, 0x0a, 0x04,
	0x66, 0x78, 0x33, 0x32, 0x18, 0x13, 0x20, 0x01, 0x28, 0x07, 0x52, 0x04, 0x66, 0x78, 0x33, 0x32,
	0x12, 0x1b, 0x0a, 0x09, 0x66, 0x78, 0x33, 0x32, 0x5f, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x14, 0x20,
	0x03, 0x28, 0x07, 0x52, 0x08, 0x66, 0x78, 0x33, 0x32, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x3d, 0x0a,
	0x08, 0x66, 0x78, 0x33, 0x32, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x15, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x22, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x6b, 0x6e, 0x69, 0x74, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53,
	0x63, 0x61, 0x6c, 0x61, 0x72, 0x73, 0x2e, 0x46, 0x78, 0x33, 0x32, 0x4d, 0x61, 0x70, 0x45, 0x6e,
	0x74, 0x72, 0x79, 0x52, 0x07, 0x66, 0x78, 0x33, 0x32, 0x4d, 0x61, 0x70, 0x12, 0x12, 0x0a, 0x04,
	0x66, 0x78, 0x36, 0x34, 0x18, 0x16, 0x20, 0x01, 0x28, 0x06, 0x52, 0x04, 0x66, 0x78, 0x36, 0x34,
	0x12, 0x1b, 0x0a, 0x09, 0x66, 0x78, 0x36, 0x34, 0x5f, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x17, 0x20,
	0x03, 0x28, 0x06, 0x52, 0x08, 0x66, 0x78, 0x36, 0x34, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x3d, 0x0a,
	0x08, 0x66, 0x78, 0x36, 0x34, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x18, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x22, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x6b, 0x6e, 0x69, 0x74, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53,
	0x63, 0x61, 0x6c, 0x61, 0x72, 0x73, 0x2e, 0x46, 0x78, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x45, 0x6e,
	0x74, 0x72, 0x79, 0x52, 0x07, 0x66, 0x78, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x12, 0x14, 0x0a, 0x05,
	0x73, 0x66, 0x78, 0x33, 0x32, 0x18, 0x19, 0x20, 0x01, 0x28, 0x0f, 0x52, 0x05, 0x73, 0x66, 0x78,
	0x33, 0x32, 0x12, 0x1d, 0x0a, 0x0a, 0x73, 0x66, 0x78, 0x33, 0x32, 0x5f, 0x6c, 0x69, 0x73, 0x74,
	0x18, 0x1a, 0x20, 0x03, 0x28, 0x0f, 0x52, 0x09, 0x73, 0x66, 0x78, 0x33, 0x32, 0x4c, 0x69, 0x73,
	0x74, 0x12, 0x40, 0x0a, 0x09, 0x73, 0x66, 0x78, 0x33, 0x32, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x1b,
	0x20, 0x03, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x6b, 0x6e, 0x69, 0x74, 0x74,
	0x65, 0x73, 0x74, 0x2e, 0x53, 0x63, 0x61, 0x6c, 0x61, 0x72, 0x73, 0x2e, 0x53, 0x66, 0x78, 0x33,
	0x32, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x08, 0x73, 0x66, 0x78, 0x33, 0x32,
	0x4d, 0x61, 0x70, 0x12, 0x14, 0x0a, 0x05, 0x73, 0x66, 0x78, 0x36, 0x34, 0x18, 0x1c, 0x20, 0x01,
	0x28, 0x10, 0x52, 0x05, 0x73, 0x66, 0x78, 0x36, 0x34, 0x12, 0x1d, 0x0a, 0x0a, 0x73, 0x66, 0x78,
	0x36, 0x34, 0x5f, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x1d, 0x20, 0x03, 0x28, 0x10, 0x52, 0x09, 0x73,
	0x66, 0x78, 0x36, 0x34, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x40, 0x0a, 0x09, 0x73, 0x66, 0x78, 0x36,
	0x34, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x1e, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x62, 0x75,
	0x66, 0x2e, 0x6b, 0x6e, 0x69, 0x74, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x63, 0x61, 0x6c, 0x61,
	0x72, 0x73, 0x2e, 0x53, 0x66, 0x78, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x52, 0x08, 0x73, 0x66, 0x78, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x12, 0x10, 0x0a, 0x03, 0x66, 0x33,
	0x32, 0x18, 0x1f, 0x20, 0x01, 0x28, 0x02, 0x52, 0x03, 0x66, 0x33, 0x32, 0x12, 0x19, 0x0a, 0x08,
	0x66, 0x33, 0x32, 0x5f, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x20, 0x20, 0x03, 0x28, 0x02, 0x52, 0x07,
	0x66, 0x33, 0x32, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x3a, 0x0a, 0x07, 0x66, 0x33, 0x32, 0x5f, 0x6d,
	0x61, 0x70, 0x18, 0x21, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x6b,
	0x6e, 0x69, 0x74, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x63, 0x61, 0x6c, 0x61, 0x72, 0x73, 0x2e,
	0x46, 0x33, 0x32, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x66, 0x33, 0x32,
	0x4d, 0x61, 0x70, 0x12, 0x10, 0x0a, 0x03, 0x66, 0x36, 0x34, 0x18, 0x22, 0x20, 0x01, 0x28, 0x01,
	0x52, 0x03, 0x66, 0x36, 0x34, 0x12, 0x19, 0x0a, 0x08, 0x66, 0x36, 0x34, 0x5f, 0x6c, 0x69, 0x73,
	0x74, 0x18, 0x23, 0x20, 0x03, 0x28, 0x01, 0x52, 0x07, 0x66, 0x36, 0x34, 0x4c, 0x69, 0x73, 0x74,
	0x12, 0x3a, 0x0a, 0x07, 0x66, 0x36, 0x34, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x24, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x21, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x6b, 0x6e, 0x69, 0x74, 0x74, 0x65, 0x73, 0x74,
	0x2e, 0x53, 0x63, 0x61, 0x6c, 0x61, 0x72, 0x73, 0x2e, 0x46, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x45,
	0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x66, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x12, 0x10, 0x0a, 0x03,
	0x73, 0x74, 0x72, 0x18, 0x25, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x73, 0x74, 0x72, 0x12, 0x19,
	0x0a, 0x08, 0x73, 0x74, 0x72, 0x5f, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x26, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x07, 0x73, 0x74, 0x72, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x3a, 0x0a, 0x07, 0x73, 0x74, 0x72,
	0x5f, 0x6d, 0x61, 0x70, 0x18, 0x27, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x62, 0x75, 0x66,
	0x2e, 0x6b, 0x6e, 0x69, 0x74, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x53, 0x63, 0x61, 0x6c, 0x61, 0x72,
	0x73, 0x2e, 0x53, 0x74, 0x72, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x73,
	0x74, 0x72, 0x4d, 0x61, 0x70, 0x12, 0x10, 0x0a, 0x03, 0x62, 0x79, 0x74, 0x18, 0x28, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x03, 0x62, 0x79, 0x74, 0x12, 0x19, 0x0a, 0x08, 0x62, 0x79, 0x74, 0x5f, 0x6c,
	0x69, 0x73, 0x74, 0x18, 0x29, 0x20, 0x03, 0x28, 0x0c, 0x52, 0x07, 0x62, 0x79, 0x74, 0x4c, 0x69,
	0x73, 0x74, 0x12, 0x3a, 0x0a, 0x07, 0x62, 0x79, 0x74, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x2a, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x21, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x6b, 0x6e, 0x69, 0x74, 0x74, 0x65,
	0x73, 0x74, 0x2e, 0x53, 0x63, 0x61, 0x6c, 0x61, 0x72, 0x73, 0x2e, 0x42, 0x79, 0x74, 0x4d, 0x61,
	0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x06, 0x62, 0x79, 0x74, 0x4d, 0x61, 0x70, 0x12, 0x0c,
	0x0a, 0x01, 0x62, 0x18, 0x2b, 0x20, 0x01, 0x28, 0x08, 0x52, 0x01, 0x62, 0x12, 0x15, 0x0a, 0x06,
	0x62, 0x5f, 0x6c, 0x69, 0x73, 0x74, 0x18, 0x2c, 0x20, 0x03, 0x28, 0x08, 0x52, 0x05, 0x62, 0x4c,
	0x69, 0x73, 0x74, 0x12, 0x34, 0x0a, 0x05, 0x62, 0x5f, 0x6d, 0x61, 0x70, 0x18, 0x2d, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x6b, 0x6e, 0x69, 0x74, 0x74, 0x65, 0x73,
	0x74, 0x2e, 0x53, 0x63, 0x61, 0x6c, 0x61, 0x72, 0x73, 0x2e, 0x42, 0x4d, 0x61, 0x70, 0x45, 0x6e,
	0x74, 0x72, 0x79, 0x52, 0x04, 0x62, 0x4d, 0x61, 0x70, 0x1a, 0x39, 0x0a, 0x0b, 0x49, 0x33, 0x32,
	0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x3a, 0x02, 0x38, 0x01, 0x1a, 0x39, 0x0a, 0x0b, 0x49, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x45, 0x6e,
	0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a,
	0x39, 0x0a, 0x0b, 0x55, 0x33, 0x32, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10,
	0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x03, 0x6b, 0x65, 0x79,
	0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0d, 0x52,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x39, 0x0a, 0x0b, 0x55, 0x36,
	0x34, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x39, 0x0a, 0x0b, 0x53, 0x33, 0x32, 0x4d, 0x61, 0x70, 0x45,
	0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x11, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x11, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01,
	0x1a, 0x39, 0x0a, 0x0b, 0x53, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12,
	0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x12, 0x52, 0x03, 0x6b, 0x65,
	0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x12,
	0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x3a, 0x0a, 0x0c, 0x46,
	0x78, 0x33, 0x32, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b,
	0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x07, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x07, 0x52, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x3a, 0x0a, 0x0c, 0x46, 0x78, 0x36, 0x34, 0x4d,
	0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x06, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x06, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a,
	0x02, 0x38, 0x01, 0x1a, 0x3b, 0x0a, 0x0d, 0x53, 0x66, 0x78, 0x33, 0x32, 0x4d, 0x61, 0x70, 0x45,
	0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0f, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0f, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01,
	0x1a, 0x3b, 0x0a, 0x0d, 0x53, 0x66, 0x78, 0x36, 0x34, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x10, 0x52, 0x03,
	0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x10, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x39, 0x0a,
	0x0b, 0x46, 0x33, 0x32, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03,
	0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14,
	0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x02, 0x52, 0x05, 0x76,
	0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x39, 0x0a, 0x0b, 0x46, 0x36, 0x34, 0x4d,
	0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x01, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a,
	0x02, 0x38, 0x01, 0x1a, 0x39, 0x0a, 0x0b, 0x53, 0x74, 0x72, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74,
	0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x39,
	0x0a, 0x0b, 0x42, 0x79, 0x74, 0x4d, 0x61, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a,
	0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12,
	0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x1a, 0x37, 0x0a, 0x09, 0x42, 0x4d, 0x61,
	0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02,
	0x38, 0x01, 0x42, 0xa8, 0x01, 0x0a, 0x10, 0x63, 0x6f, 0x6d, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x6b,
	0x6e, 0x69, 0x74, 0x74, 0x65, 0x73, 0x74, 0x42, 0x0c, 0x53, 0x63, 0x61, 0x6c, 0x61, 0x72, 0x73,
	0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x35, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x75, 0x66, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x6b, 0x6e, 0x69,
	0x74, 0x2d, 0x67, 0x6f, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x67, 0x65,
	0x6e, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x6b, 0x6e, 0x69, 0x74, 0x74, 0x65, 0x73, 0x74, 0xa2, 0x02,
	0x03, 0x42, 0x4b, 0x58, 0xaa, 0x02, 0x0c, 0x42, 0x75, 0x66, 0x2e, 0x4b, 0x6e, 0x69, 0x74, 0x74,
	0x65, 0x73, 0x74, 0xca, 0x02, 0x0c, 0x42, 0x75, 0x66, 0x5c, 0x4b, 0x6e, 0x69, 0x74, 0x74, 0x65,
	0x73, 0x74, 0xe2, 0x02, 0x18, 0x42, 0x75, 0x66, 0x5c, 0x4b, 0x6e, 0x69, 0x74, 0x74, 0x65, 0x73,
	0x74, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x0d,
	0x42, 0x75, 0x66, 0x3a, 0x3a, 0x4b, 0x6e, 0x69, 0x74, 0x74, 0x65, 0x73, 0x74, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_buf_knittest_scalars_proto_rawDescOnce sync.Once
	file_buf_knittest_scalars_proto_rawDescData = file_buf_knittest_scalars_proto_rawDesc
)

func file_buf_knittest_scalars_proto_rawDescGZIP() []byte {
	file_buf_knittest_scalars_proto_rawDescOnce.Do(func() {
		file_buf_knittest_scalars_proto_rawDescData = protoimpl.X.CompressGZIP(file_buf_knittest_scalars_proto_rawDescData)
	})
	return file_buf_knittest_scalars_proto_rawDescData
}

var file_buf_knittest_scalars_proto_msgTypes = make([]protoimpl.MessageInfo, 16)
var file_buf_knittest_scalars_proto_goTypes = []interface{}{
	(*Scalars)(nil), // 0: buf.knittest.Scalars
	nil,             // 1: buf.knittest.Scalars.I32MapEntry
	nil,             // 2: buf.knittest.Scalars.I64MapEntry
	nil,             // 3: buf.knittest.Scalars.U32MapEntry
	nil,             // 4: buf.knittest.Scalars.U64MapEntry
	nil,             // 5: buf.knittest.Scalars.S32MapEntry
	nil,             // 6: buf.knittest.Scalars.S64MapEntry
	nil,             // 7: buf.knittest.Scalars.Fx32MapEntry
	nil,             // 8: buf.knittest.Scalars.Fx64MapEntry
	nil,             // 9: buf.knittest.Scalars.Sfx32MapEntry
	nil,             // 10: buf.knittest.Scalars.Sfx64MapEntry
	nil,             // 11: buf.knittest.Scalars.F32MapEntry
	nil,             // 12: buf.knittest.Scalars.F64MapEntry
	nil,             // 13: buf.knittest.Scalars.StrMapEntry
	nil,             // 14: buf.knittest.Scalars.BytMapEntry
	nil,             // 15: buf.knittest.Scalars.BMapEntry
}
var file_buf_knittest_scalars_proto_depIdxs = []int32{
	1,  // 0: buf.knittest.Scalars.i32_map:type_name -> buf.knittest.Scalars.I32MapEntry
	2,  // 1: buf.knittest.Scalars.i64_map:type_name -> buf.knittest.Scalars.I64MapEntry
	3,  // 2: buf.knittest.Scalars.u32_map:type_name -> buf.knittest.Scalars.U32MapEntry
	4,  // 3: buf.knittest.Scalars.u64_map:type_name -> buf.knittest.Scalars.U64MapEntry
	5,  // 4: buf.knittest.Scalars.s32_map:type_name -> buf.knittest.Scalars.S32MapEntry
	6,  // 5: buf.knittest.Scalars.s64_map:type_name -> buf.knittest.Scalars.S64MapEntry
	7,  // 6: buf.knittest.Scalars.fx32_map:type_name -> buf.knittest.Scalars.Fx32MapEntry
	8,  // 7: buf.knittest.Scalars.fx64_map:type_name -> buf.knittest.Scalars.Fx64MapEntry
	9,  // 8: buf.knittest.Scalars.sfx32_map:type_name -> buf.knittest.Scalars.Sfx32MapEntry
	10, // 9: buf.knittest.Scalars.sfx64_map:type_name -> buf.knittest.Scalars.Sfx64MapEntry
	11, // 10: buf.knittest.Scalars.f32_map:type_name -> buf.knittest.Scalars.F32MapEntry
	12, // 11: buf.knittest.Scalars.f64_map:type_name -> buf.knittest.Scalars.F64MapEntry
	13, // 12: buf.knittest.Scalars.str_map:type_name -> buf.knittest.Scalars.StrMapEntry
	14, // 13: buf.knittest.Scalars.byt_map:type_name -> buf.knittest.Scalars.BytMapEntry
	15, // 14: buf.knittest.Scalars.b_map:type_name -> buf.knittest.Scalars.BMapEntry
	15, // [15:15] is the sub-list for method output_type
	15, // [15:15] is the sub-list for method input_type
	15, // [15:15] is the sub-list for extension type_name
	15, // [15:15] is the sub-list for extension extendee
	0,  // [0:15] is the sub-list for field type_name
}

func init() { file_buf_knittest_scalars_proto_init() }
func file_buf_knittest_scalars_proto_init() {
	if File_buf_knittest_scalars_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_buf_knittest_scalars_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Scalars); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_buf_knittest_scalars_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   16,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_buf_knittest_scalars_proto_goTypes,
		DependencyIndexes: file_buf_knittest_scalars_proto_depIdxs,
		MessageInfos:      file_buf_knittest_scalars_proto_msgTypes,
	}.Build()
	File_buf_knittest_scalars_proto = out.File
	file_buf_knittest_scalars_proto_rawDesc = nil
	file_buf_knittest_scalars_proto_goTypes = nil
	file_buf_knittest_scalars_proto_depIdxs = nil
}