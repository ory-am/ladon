// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/ory-am/ladon (interfaces: Manager)

package ladon_test

import (
	gomock "github.com/golang/mock/gomock"
	ladon "github.com/ory-am/ladon"
)

// Mock of Manager interface
type MockManager struct {
	ctrl     *gomock.Controller
	recorder *_MockManagerRecorder
}

// Recorder for MockManager (not exported)
type _MockManagerRecorder struct {
	mock *MockManager
}

func NewMockManager(ctrl *gomock.Controller) *MockManager {
	mock := &MockManager{ctrl: ctrl}
	mock.recorder = &_MockManagerRecorder{mock}
	return mock
}

func (_m *MockManager) EXPECT() *_MockManagerRecorder {
	return _m.recorder
}

func (_m *MockManager) Create(_param0 ladon.Policy) error {
	ret := _m.ctrl.Call(_m, "Create", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockManagerRecorder) Create(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Create", arg0)
}

func (_m *MockManager) Delete(_param0 string) error {
	ret := _m.ctrl.Call(_m, "Delete", _param0)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockManagerRecorder) Delete(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Delete", arg0)
}

func (_m *MockManager) FindRequestCandidates(_param0 *ladon.Request) (ladon.Policies, error) {
	ret := _m.ctrl.Call(_m, "FindRequestCandidates", _param0)
	ret0, _ := ret[0].(ladon.Policies)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockManagerRecorder) FindRequestCandidates(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "FindRequestCandidates", arg0)
}

func (_m *MockManager) Get(_param0 string) (ladon.Policy, error) {
	ret := _m.ctrl.Call(_m, "Get", _param0)
	ret0, _ := ret[0].(ladon.Policy)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (_mr *_MockManagerRecorder) Get(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Get", arg0)
}
