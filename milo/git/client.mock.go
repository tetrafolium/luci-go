// Code generated by MockGen. DO NOT EDIT.
// Source: client.go

// Package git is a generated GoMock package.
package git

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	git "github.com/tetrafolium/luci-go/common/proto/git"
	reflect "reflect"
)

// MockClient is a mock of Client interface.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient.
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// Log mocks base method.
func (m *MockClient) Log(c context.Context, host, project, commitish string, inputOptions *LogOptions) ([]*git.Commit, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Log", c, host, project, commitish, inputOptions)
	ret0, _ := ret[0].([]*git.Commit)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Log indicates an expected call of Log.
func (mr *MockClientMockRecorder) Log(c, host, project, commitish, inputOptions interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Log", reflect.TypeOf((*MockClient)(nil).Log), c, host, project, commitish, inputOptions)
}

// CombinedLogs mocks base method.
func (m *MockClient) CombinedLogs(c context.Context, host, project, excludeRef string, refs []string, limit int) ([]*git.Commit, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CombinedLogs", c, host, project, excludeRef, refs, limit)
	ret0, _ := ret[0].([]*git.Commit)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CombinedLogs indicates an expected call of CombinedLogs.
func (mr *MockClientMockRecorder) CombinedLogs(c, host, project, excludeRef, refs, limit interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CombinedLogs", reflect.TypeOf((*MockClient)(nil).CombinedLogs), c, host, project, excludeRef, refs, limit)
}

// CLEmail mocks base method.
func (m *MockClient) CLEmail(c context.Context, host string, changeNumber int64) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CLEmail", c, host, changeNumber)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CLEmail indicates an expected call of CLEmail.
func (mr *MockClientMockRecorder) CLEmail(c, host, changeNumber interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CLEmail", reflect.TypeOf((*MockClient)(nil).CLEmail), c, host, changeNumber)
}
