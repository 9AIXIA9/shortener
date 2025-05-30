// Code generated by MockGen. DO NOT EDIT.
// Source: cache.go
//
// Generated by this command:
//
//	mockgen -source=cache.go -destination=./mock/cache_mock.go -package=cachex
//

// Package cachex is a generated GoMock package.
package cachex

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockSequenceCache is a mock of SequenceCache interface.
type MockSequenceCache struct {
	ctrl     *gomock.Controller
	recorder *MockSequenceCacheMockRecorder
	isgomock struct{}
}

// MockSequenceCacheMockRecorder is the mock recorder for MockSequenceCache.
type MockSequenceCacheMockRecorder struct {
	mock *MockSequenceCache
}

// NewMockSequenceCache creates a new mock instance.
func NewMockSequenceCache(ctrl *gomock.Controller) *MockSequenceCache {
	mock := &MockSequenceCache{ctrl: ctrl}
	mock.recorder = &MockSequenceCacheMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSequenceCache) EXPECT() *MockSequenceCacheMockRecorder {
	return m.recorder
}

// FillIDs mocks base method.
func (m *MockSequenceCache) FillIDs(ctx context.Context, ids []uint64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FillIDs", ctx, ids)
	ret0, _ := ret[0].(error)
	return ret0
}

// FillIDs indicates an expected call of FillIDs.
func (mr *MockSequenceCacheMockRecorder) FillIDs(ctx, ids any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FillIDs", reflect.TypeOf((*MockSequenceCache)(nil).FillIDs), ctx, ids)
}

// GetSingleID mocks base method.
func (m *MockSequenceCache) GetSingleID(ctx context.Context) (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSingleID", ctx)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSingleID indicates an expected call of GetSingleID.
func (mr *MockSequenceCacheMockRecorder) GetSingleID(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSingleID", reflect.TypeOf((*MockSequenceCache)(nil).GetSingleID), ctx)
}

// IsLessThanThreshold mocks base method.
func (m *MockSequenceCache) IsLessThanThreshold(ctx context.Context, threshold int) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsLessThanThreshold", ctx, threshold)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IsLessThanThreshold indicates an expected call of IsLessThanThreshold.
func (mr *MockSequenceCacheMockRecorder) IsLessThanThreshold(ctx, threshold any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsLessThanThreshold", reflect.TypeOf((*MockSequenceCache)(nil).IsLessThanThreshold), ctx, threshold)
}

// IsOK mocks base method.
func (m *MockSequenceCache) IsOK(ctx context.Context) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsOK", ctx)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsOK indicates an expected call of IsOK.
func (mr *MockSequenceCacheMockRecorder) IsOK(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsOK", reflect.TypeOf((*MockSequenceCache)(nil).IsOK), ctx)
}
