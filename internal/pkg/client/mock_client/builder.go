// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client (interfaces: Builder)
//
// Generated by this command:
//
//	mockgen -build_flags=--mod=mod -package mock_client -destination=mock_client/builder.go . Builder
//

// Package mock_client is a generated GoMock package.
package mock_client

import (
	context "context"
	reflect "reflect"

	client "github.com/yandex-cloud/cluster-api-provider-yandex/internal/pkg/client"
	gomock "go.uber.org/mock/gomock"
	client0 "sigs.k8s.io/controller-runtime/pkg/client"
)

// MockBuilder is a mock of Builder interface.
type MockBuilder struct {
	ctrl     *gomock.Controller
	recorder *MockBuilderMockRecorder
}

// MockBuilderMockRecorder is the mock recorder for MockBuilder.
type MockBuilderMockRecorder struct {
	mock *MockBuilder
}

// NewMockBuilder creates a new mock instance.
func NewMockBuilder(ctrl *gomock.Controller) *MockBuilder {
	mock := &MockBuilder{ctrl: ctrl}
	mock.recorder = &MockBuilderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBuilder) EXPECT() *MockBuilderMockRecorder {
	return m.recorder
}

// GetClientFromKey mocks base method.
func (m *MockBuilder) GetClientFromKey(arg0 context.Context, arg1 string) (client.Client, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetClientFromKey", arg0, arg1)
	ret0, _ := ret[0].(client.Client)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetClientFromKey indicates an expected call of GetClientFromKey.
func (mr *MockBuilderMockRecorder) GetClientFromKey(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetClientFromKey", reflect.TypeOf((*MockBuilder)(nil).GetClientFromKey), arg0, arg1)
}

// GetClientFromSecret mocks base method.
func (m *MockBuilder) GetClientFromSecret(arg0 context.Context, arg1 client0.Client, arg2, arg3, arg4 string) (client.Client, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetClientFromSecret", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(client.Client)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetClientFromSecret indicates an expected call of GetClientFromSecret.
func (mr *MockBuilderMockRecorder) GetClientFromSecret(arg0, arg1, arg2, arg3, arg4 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetClientFromSecret", reflect.TypeOf((*MockBuilder)(nil).GetClientFromSecret), arg0, arg1, arg2, arg3, arg4)
}

// GetDefaultClient mocks base method.
func (m *MockBuilder) GetDefaultClient(arg0 context.Context) (client.Client, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDefaultClient", arg0)
	ret0, _ := ret[0].(client.Client)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDefaultClient indicates an expected call of GetDefaultClient.
func (mr *MockBuilderMockRecorder) GetDefaultClient(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDefaultClient", reflect.TypeOf((*MockBuilder)(nil).GetDefaultClient), arg0)
}
