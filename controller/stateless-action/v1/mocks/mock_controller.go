// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/ondat/operator-toolkit/controller/stateless-action/v1 (interfaces: Controller)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	action "github.com/ondat/operator-toolkit/controller/stateless-action/v1/action"
	types "k8s.io/apimachinery/pkg/types"
)

// MockController is a mock of Controller interface.
type MockController struct {
	ctrl     *gomock.Controller
	recorder *MockControllerMockRecorder
}

// MockControllerMockRecorder is the mock recorder for MockController.
type MockControllerMockRecorder struct {
	mock *MockController
}

// NewMockController creates a new mock instance.
func NewMockController(ctrl *gomock.Controller) *MockController {
	mock := &MockController{ctrl: ctrl}
	mock.recorder = &MockControllerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockController) EXPECT() *MockControllerMockRecorder {
	return m.recorder
}

// BuildActionManager mocks base method.
func (m *MockController) BuildActionManager(arg0 interface{}) (action.Manager, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BuildActionManager", arg0)
	ret0, _ := ret[0].(action.Manager)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// BuildActionManager indicates an expected call of BuildActionManager.
func (mr *MockControllerMockRecorder) BuildActionManager(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BuildActionManager", reflect.TypeOf((*MockController)(nil).BuildActionManager), arg0)
}

// GetObject mocks base method.
func (m *MockController) GetObject(arg0 context.Context, arg1 types.NamespacedName) (interface{}, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetObject", arg0, arg1)
	ret0, _ := ret[0].(interface{})
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetObject indicates an expected call of GetObject.
func (mr *MockControllerMockRecorder) GetObject(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetObject", reflect.TypeOf((*MockController)(nil).GetObject), arg0, arg1)
}

// RequireAction mocks base method.
func (m *MockController) RequireAction(arg0 context.Context, arg1 interface{}) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RequireAction", arg0, arg1)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RequireAction indicates an expected call of RequireAction.
func (mr *MockControllerMockRecorder) RequireAction(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RequireAction", reflect.TypeOf((*MockController)(nil).RequireAction), arg0, arg1)
}
