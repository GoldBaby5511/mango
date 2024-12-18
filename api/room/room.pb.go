// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.35.2
// 	protoc        v3.20.3
// source: room.proto

package room

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	types "mango/api/types"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type CMDRoom int32

const (
	CMDRoom_IDJoinReq         CMDRoom = 1 //进入房间
	CMDRoom_IDJoinRsp         CMDRoom = 2 //进入房间
	CMDRoom_IDUserActionReq   CMDRoom = 3 //用户动作
	CMDRoom_IDUserActionRsp   CMDRoom = 4 //用户动作
	CMDRoom_IDExitReq         CMDRoom = 5 //离开房间
	CMDRoom_IDExitRsp         CMDRoom = 6 //离开房间
	CMDRoom_IDUserStateChange CMDRoom = 7 //状态变化
)

// Enum value maps for CMDRoom.
var (
	CMDRoom_name = map[int32]string{
		1: "IDJoinReq",
		2: "IDJoinRsp",
		3: "IDUserActionReq",
		4: "IDUserActionRsp",
		5: "IDExitReq",
		6: "IDExitRsp",
		7: "IDUserStateChange",
	}
	CMDRoom_value = map[string]int32{
		"IDJoinReq":         1,
		"IDJoinRsp":         2,
		"IDUserActionReq":   3,
		"IDUserActionRsp":   4,
		"IDExitReq":         5,
		"IDExitRsp":         6,
		"IDUserStateChange": 7,
	}
)

func (x CMDRoom) Enum() *CMDRoom {
	p := new(CMDRoom)
	*p = x
	return p
}

func (x CMDRoom) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (CMDRoom) Descriptor() protoreflect.EnumDescriptor {
	return file_room_proto_enumTypes[0].Descriptor()
}

func (CMDRoom) Type() protoreflect.EnumType {
	return &file_room_proto_enumTypes[0]
}

func (x CMDRoom) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *CMDRoom) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = CMDRoom(num)
	return nil
}

// Deprecated: Use CMDRoom.Descriptor instead.
func (CMDRoom) EnumDescriptor() ([]byte, []int) {
	return file_room_proto_rawDescGZIP(), []int{0}
}

type ActionType int32

const (
	ActionType_Ready  ActionType = 1
	ActionType_Cancel ActionType = 2
)

// Enum value maps for ActionType.
var (
	ActionType_name = map[int32]string{
		1: "Ready",
		2: "Cancel",
	}
	ActionType_value = map[string]int32{
		"Ready":  1,
		"Cancel": 2,
	}
)

func (x ActionType) Enum() *ActionType {
	p := new(ActionType)
	*p = x
	return p
}

func (x ActionType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ActionType) Descriptor() protoreflect.EnumDescriptor {
	return file_room_proto_enumTypes[1].Descriptor()
}

func (ActionType) Type() protoreflect.EnumType {
	return &file_room_proto_enumTypes[1]
}

func (x ActionType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *ActionType) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = ActionType(num)
	return nil
}

// Deprecated: Use ActionType.Descriptor instead.
func (ActionType) EnumDescriptor() ([]byte, []int) {
	return file_room_proto_rawDescGZIP(), []int{1}
}

type JoinReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *JoinReq) Reset() {
	*x = JoinReq{}
	mi := &file_room_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *JoinReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JoinReq) ProtoMessage() {}

func (x *JoinReq) ProtoReflect() protoreflect.Message {
	mi := &file_room_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use JoinReq.ProtoReflect.Descriptor instead.
func (*JoinReq) Descriptor() ([]byte, []int) {
	return file_room_proto_rawDescGZIP(), []int{0}
}

type JoinRsp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AppId   *uint32          `protobuf:"varint,1,opt,name=app_id,json=appId" json:"app_id,omitempty"`
	ErrInfo *types.ErrorInfo `protobuf:"bytes,99,opt,name=err_info,json=errInfo" json:"err_info,omitempty"`
}

func (x *JoinRsp) Reset() {
	*x = JoinRsp{}
	mi := &file_room_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *JoinRsp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*JoinRsp) ProtoMessage() {}

func (x *JoinRsp) ProtoReflect() protoreflect.Message {
	mi := &file_room_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use JoinRsp.ProtoReflect.Descriptor instead.
func (*JoinRsp) Descriptor() ([]byte, []int) {
	return file_room_proto_rawDescGZIP(), []int{1}
}

func (x *JoinRsp) GetAppId() uint32 {
	if x != nil && x.AppId != nil {
		return *x.AppId
	}
	return 0
}

func (x *JoinRsp) GetErrInfo() *types.ErrorInfo {
	if x != nil {
		return x.ErrInfo
	}
	return nil
}

type UserActionReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Action *ActionType `protobuf:"varint,1,opt,name=action,enum=bs.room.ActionType" json:"action,omitempty"`
}

func (x *UserActionReq) Reset() {
	*x = UserActionReq{}
	mi := &file_room_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserActionReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserActionReq) ProtoMessage() {}

func (x *UserActionReq) ProtoReflect() protoreflect.Message {
	mi := &file_room_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserActionReq.ProtoReflect.Descriptor instead.
func (*UserActionReq) Descriptor() ([]byte, []int) {
	return file_room_proto_rawDescGZIP(), []int{2}
}

func (x *UserActionReq) GetAction() ActionType {
	if x != nil && x.Action != nil {
		return *x.Action
	}
	return ActionType_Ready
}

type UserActionRsp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Action  *ActionType      `protobuf:"varint,1,opt,name=action,enum=bs.room.ActionType" json:"action,omitempty"`
	ErrInfo *types.ErrorInfo `protobuf:"bytes,99,opt,name=err_info,json=errInfo" json:"err_info,omitempty"`
}

func (x *UserActionRsp) Reset() {
	*x = UserActionRsp{}
	mi := &file_room_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserActionRsp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserActionRsp) ProtoMessage() {}

func (x *UserActionRsp) ProtoReflect() protoreflect.Message {
	mi := &file_room_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserActionRsp.ProtoReflect.Descriptor instead.
func (*UserActionRsp) Descriptor() ([]byte, []int) {
	return file_room_proto_rawDescGZIP(), []int{3}
}

func (x *UserActionRsp) GetAction() ActionType {
	if x != nil && x.Action != nil {
		return *x.Action
	}
	return ActionType_Ready
}

func (x *UserActionRsp) GetErrInfo() *types.ErrorInfo {
	if x != nil {
		return x.ErrInfo
	}
	return nil
}

type ExitReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ExitReq) Reset() {
	*x = ExitReq{}
	mi := &file_room_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ExitReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExitReq) ProtoMessage() {}

func (x *ExitReq) ProtoReflect() protoreflect.Message {
	mi := &file_room_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExitReq.ProtoReflect.Descriptor instead.
func (*ExitReq) Descriptor() ([]byte, []int) {
	return file_room_proto_rawDescGZIP(), []int{4}
}

type ExitRsp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ErrInfo *types.ErrorInfo `protobuf:"bytes,99,opt,name=err_info,json=errInfo" json:"err_info,omitempty"`
}

func (x *ExitRsp) Reset() {
	*x = ExitRsp{}
	mi := &file_room_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ExitRsp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExitRsp) ProtoMessage() {}

func (x *ExitRsp) ProtoReflect() protoreflect.Message {
	mi := &file_room_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExitRsp.ProtoReflect.Descriptor instead.
func (*ExitRsp) Descriptor() ([]byte, []int) {
	return file_room_proto_rawDescGZIP(), []int{5}
}

func (x *ExitRsp) GetErrInfo() *types.ErrorInfo {
	if x != nil {
		return x.ErrInfo
	}
	return nil
}

type UserStateChange struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	UserId         *uint64 `protobuf:"varint,1,opt,name=user_id,json=userId" json:"user_id,omitempty"`                           //用户ID
	UserState      *uint32 `protobuf:"varint,2,opt,name=user_state,json=userState" json:"user_state,omitempty"`                  //用户状态
	TableServiceId *uint32 `protobuf:"varint,3,opt,name=table_service_id,json=tableServiceId" json:"table_service_id,omitempty"` //
	TableId        *uint64 `protobuf:"varint,4,opt,name=table_id,json=tableId" json:"table_id,omitempty"`                        //
	SeatId         *uint32 `protobuf:"varint,5,opt,name=seat_id,json=seatId" json:"seat_id,omitempty"`                           //
}

func (x *UserStateChange) Reset() {
	*x = UserStateChange{}
	mi := &file_room_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserStateChange) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserStateChange) ProtoMessage() {}

func (x *UserStateChange) ProtoReflect() protoreflect.Message {
	mi := &file_room_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserStateChange.ProtoReflect.Descriptor instead.
func (*UserStateChange) Descriptor() ([]byte, []int) {
	return file_room_proto_rawDescGZIP(), []int{6}
}

func (x *UserStateChange) GetUserId() uint64 {
	if x != nil && x.UserId != nil {
		return *x.UserId
	}
	return 0
}

func (x *UserStateChange) GetUserState() uint32 {
	if x != nil && x.UserState != nil {
		return *x.UserState
	}
	return 0
}

func (x *UserStateChange) GetTableServiceId() uint32 {
	if x != nil && x.TableServiceId != nil {
		return *x.TableServiceId
	}
	return 0
}

func (x *UserStateChange) GetTableId() uint64 {
	if x != nil && x.TableId != nil {
		return *x.TableId
	}
	return 0
}

func (x *UserStateChange) GetSeatId() uint32 {
	if x != nil && x.SeatId != nil {
		return *x.SeatId
	}
	return 0
}

var File_room_proto protoreflect.FileDescriptor

var file_room_proto_rawDesc = []byte{
	0x0a, 0x0a, 0x72, 0x6f, 0x6f, 0x6d, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x07, 0x62, 0x73,
	0x2e, 0x72, 0x6f, 0x6f, 0x6d, 0x1a, 0x0b, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0x09, 0x0a, 0x07, 0x4a, 0x6f, 0x69, 0x6e, 0x52, 0x65, 0x71, 0x22, 0x50, 0x0a,
	0x07, 0x4a, 0x6f, 0x69, 0x6e, 0x52, 0x73, 0x70, 0x12, 0x15, 0x0a, 0x06, 0x61, 0x70, 0x70, 0x5f,
	0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x05, 0x61, 0x70, 0x70, 0x49, 0x64, 0x12,
	0x2e, 0x0a, 0x08, 0x65, 0x72, 0x72, 0x5f, 0x69, 0x6e, 0x66, 0x6f, 0x18, 0x63, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x13, 0x2e, 0x62, 0x73, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x45, 0x72, 0x72,
	0x6f, 0x72, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x07, 0x65, 0x72, 0x72, 0x49, 0x6e, 0x66, 0x6f, 0x22,
	0x3c, 0x0a, 0x0d, 0x55, 0x73, 0x65, 0x72, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71,
	0x12, 0x2b, 0x0a, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e,
	0x32, 0x13, 0x2e, 0x62, 0x73, 0x2e, 0x72, 0x6f, 0x6f, 0x6d, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x54, 0x79, 0x70, 0x65, 0x52, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x6c, 0x0a,
	0x0d, 0x55, 0x73, 0x65, 0x72, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x73, 0x70, 0x12, 0x2b,
	0x0a, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x13,
	0x2e, 0x62, 0x73, 0x2e, 0x72, 0x6f, 0x6f, 0x6d, 0x2e, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x54,
	0x79, 0x70, 0x65, 0x52, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x2e, 0x0a, 0x08, 0x65,
	0x72, 0x72, 0x5f, 0x69, 0x6e, 0x66, 0x6f, 0x18, 0x63, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e,
	0x62, 0x73, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x49, 0x6e,
	0x66, 0x6f, 0x52, 0x07, 0x65, 0x72, 0x72, 0x49, 0x6e, 0x66, 0x6f, 0x22, 0x09, 0x0a, 0x07, 0x45,
	0x78, 0x69, 0x74, 0x52, 0x65, 0x71, 0x22, 0x39, 0x0a, 0x07, 0x45, 0x78, 0x69, 0x74, 0x52, 0x73,
	0x70, 0x12, 0x2e, 0x0a, 0x08, 0x65, 0x72, 0x72, 0x5f, 0x69, 0x6e, 0x66, 0x6f, 0x18, 0x63, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x62, 0x73, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x45,
	0x72, 0x72, 0x6f, 0x72, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x07, 0x65, 0x72, 0x72, 0x49, 0x6e, 0x66,
	0x6f, 0x22, 0xa7, 0x01, 0x0a, 0x0f, 0x55, 0x73, 0x65, 0x72, 0x53, 0x74, 0x61, 0x74, 0x65, 0x43,
	0x68, 0x61, 0x6e, 0x67, 0x65, 0x12, 0x17, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x12, 0x1d,
	0x0a, 0x0a, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0d, 0x52, 0x09, 0x75, 0x73, 0x65, 0x72, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x28, 0x0a,
	0x10, 0x74, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x69,
	0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0e, 0x74, 0x61, 0x62, 0x6c, 0x65, 0x53, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x49, 0x64, 0x12, 0x19, 0x0a, 0x08, 0x74, 0x61, 0x62, 0x6c, 0x65,
	0x5f, 0x69, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x04, 0x52, 0x07, 0x74, 0x61, 0x62, 0x6c, 0x65,
	0x49, 0x64, 0x12, 0x17, 0x0a, 0x07, 0x73, 0x65, 0x61, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x05, 0x20,
	0x01, 0x28, 0x0d, 0x52, 0x06, 0x73, 0x65, 0x61, 0x74, 0x49, 0x64, 0x2a, 0x86, 0x01, 0x0a, 0x07,
	0x43, 0x4d, 0x44, 0x52, 0x6f, 0x6f, 0x6d, 0x12, 0x0d, 0x0a, 0x09, 0x49, 0x44, 0x4a, 0x6f, 0x69,
	0x6e, 0x52, 0x65, 0x71, 0x10, 0x01, 0x12, 0x0d, 0x0a, 0x09, 0x49, 0x44, 0x4a, 0x6f, 0x69, 0x6e,
	0x52, 0x73, 0x70, 0x10, 0x02, 0x12, 0x13, 0x0a, 0x0f, 0x49, 0x44, 0x55, 0x73, 0x65, 0x72, 0x41,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x65, 0x71, 0x10, 0x03, 0x12, 0x13, 0x0a, 0x0f, 0x49, 0x44,
	0x55, 0x73, 0x65, 0x72, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x73, 0x70, 0x10, 0x04, 0x12,
	0x0d, 0x0a, 0x09, 0x49, 0x44, 0x45, 0x78, 0x69, 0x74, 0x52, 0x65, 0x71, 0x10, 0x05, 0x12, 0x0d,
	0x0a, 0x09, 0x49, 0x44, 0x45, 0x78, 0x69, 0x74, 0x52, 0x73, 0x70, 0x10, 0x06, 0x12, 0x15, 0x0a,
	0x11, 0x49, 0x44, 0x55, 0x73, 0x65, 0x72, 0x53, 0x74, 0x61, 0x74, 0x65, 0x43, 0x68, 0x61, 0x6e,
	0x67, 0x65, 0x10, 0x07, 0x2a, 0x23, 0x0a, 0x0a, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x09, 0x0a, 0x05, 0x52, 0x65, 0x61, 0x64, 0x79, 0x10, 0x01, 0x12, 0x0a, 0x0a,
	0x06, 0x43, 0x61, 0x6e, 0x63, 0x65, 0x6c, 0x10, 0x02, 0x42, 0x07, 0x5a, 0x05, 0x2f, 0x72, 0x6f,
	0x6f, 0x6d,
}

var (
	file_room_proto_rawDescOnce sync.Once
	file_room_proto_rawDescData = file_room_proto_rawDesc
)

func file_room_proto_rawDescGZIP() []byte {
	file_room_proto_rawDescOnce.Do(func() {
		file_room_proto_rawDescData = protoimpl.X.CompressGZIP(file_room_proto_rawDescData)
	})
	return file_room_proto_rawDescData
}

var file_room_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_room_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_room_proto_goTypes = []any{
	(CMDRoom)(0),            // 0: bs.room.CMDRoom
	(ActionType)(0),         // 1: bs.room.ActionType
	(*JoinReq)(nil),         // 2: bs.room.JoinReq
	(*JoinRsp)(nil),         // 3: bs.room.JoinRsp
	(*UserActionReq)(nil),   // 4: bs.room.UserActionReq
	(*UserActionRsp)(nil),   // 5: bs.room.UserActionRsp
	(*ExitReq)(nil),         // 6: bs.room.ExitReq
	(*ExitRsp)(nil),         // 7: bs.room.ExitRsp
	(*UserStateChange)(nil), // 8: bs.room.UserStateChange
	(*types.ErrorInfo)(nil), // 9: bs.types.ErrorInfo
}
var file_room_proto_depIdxs = []int32{
	9, // 0: bs.room.JoinRsp.err_info:type_name -> bs.types.ErrorInfo
	1, // 1: bs.room.UserActionReq.action:type_name -> bs.room.ActionType
	1, // 2: bs.room.UserActionRsp.action:type_name -> bs.room.ActionType
	9, // 3: bs.room.UserActionRsp.err_info:type_name -> bs.types.ErrorInfo
	9, // 4: bs.room.ExitRsp.err_info:type_name -> bs.types.ErrorInfo
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_room_proto_init() }
func file_room_proto_init() {
	if File_room_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_room_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_room_proto_goTypes,
		DependencyIndexes: file_room_proto_depIdxs,
		EnumInfos:         file_room_proto_enumTypes,
		MessageInfos:      file_room_proto_msgTypes,
	}.Build()
	File_room_proto = out.File
	file_room_proto_rawDesc = nil
	file_room_proto_goTypes = nil
	file_room_proto_depIdxs = nil
}
