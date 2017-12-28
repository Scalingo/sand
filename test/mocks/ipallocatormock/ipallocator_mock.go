// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/Scalingo/sand/ipallocator (interfaces: IPAllocator)

// Package ipallocatormock is a generated GoMock package.
package ipallocatormock

import (
	context "context"
	net "net"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockIPAllocator is a mock of IPAllocator interface
type MockIPAllocator struct {
	ctrl     *gomock.Controller
	recorder *MockIPAllocatorMockRecorder
}

// MockIPAllocatorMockRecorder is the mock recorder for MockIPAllocator
type MockIPAllocatorMockRecorder struct {
	mock *MockIPAllocator
}

// NewMockIPAllocator creates a new mock instance
func NewMockIPAllocator(ctrl *gomock.Controller) *MockIPAllocator {
	mock := &MockIPAllocator{ctrl: ctrl}
	mock.recorder = &MockIPAllocatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockIPAllocator) EXPECT() *MockIPAllocatorMockRecorder {
	return m.recorder
}

// AllocateIP mocks base method
func (m *MockIPAllocator) AllocateIP(arg0 context.Context) (net.IP, uint, error) {
	ret := m.ctrl.Call(m, "AllocateIP", arg0)
	ret0, _ := ret[0].(net.IP)
	ret1, _ := ret[1].(uint)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// AllocateIP indicates an expected call of AllocateIP
func (mr *MockIPAllocatorMockRecorder) AllocateIP(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AllocateIP", reflect.TypeOf((*MockIPAllocator)(nil).AllocateIP), arg0)
}
