// Code generated by MockGen. DO NOT EDIT.
// Source: code.cloudfoundry.org/cfdev/cmd/start (interfaces: Hypervisor)

// Package mocks is a generated GoMock package.
package mocks

import (
	hypervisor "code.cloudfoundry.org/cfdev/hypervisor"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockHypervisor is a mock of Hypervisor interface
type MockHypervisor struct {
	ctrl     *gomock.Controller
	recorder *MockHypervisorMockRecorder
}

// MockHypervisorMockRecorder is the mock recorder for MockHypervisor
type MockHypervisorMockRecorder struct {
	mock *MockHypervisor
}

// NewMockHypervisor creates a new mock instance
func NewMockHypervisor(ctrl *gomock.Controller) *MockHypervisor {
	mock := &MockHypervisor{ctrl: ctrl}
	mock.recorder = &MockHypervisorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockHypervisor) EXPECT() *MockHypervisorMockRecorder {
	return m.recorder
}

// CreateVM mocks base method
func (m *MockHypervisor) CreateVM(arg0 hypervisor.VM) error {
	ret := m.ctrl.Call(m, "CreateVM", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateVM indicates an expected call of CreateVM
func (mr *MockHypervisorMockRecorder) CreateVM(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVM", reflect.TypeOf((*MockHypervisor)(nil).CreateVM), arg0)
}

// IsRunning mocks base method
func (m *MockHypervisor) IsRunning(arg0 string) (bool, error) {
	ret := m.ctrl.Call(m, "IsRunning", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsRunning indicates an expected call of IsRunning
func (mr *MockHypervisorMockRecorder) IsRunning(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsRunning", reflect.TypeOf((*MockHypervisor)(nil).IsRunning), arg0)
}

// Start mocks base method
func (m *MockHypervisor) Start(arg0 string) error {
	ret := m.ctrl.Call(m, "Start", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start
func (mr *MockHypervisorMockRecorder) Start(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockHypervisor)(nil).Start), arg0)
}

// Stop mocks base method
func (m *MockHypervisor) Stop(arg0 string) error {
	ret := m.ctrl.Call(m, "Stop", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop
func (mr *MockHypervisorMockRecorder) Stop(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockHypervisor)(nil).Stop), arg0)
}
