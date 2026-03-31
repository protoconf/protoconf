package otelkv

import (
	"context"
	"errors"
	"testing"

	"github.com/kvtools/valkeyrie/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStore is a store.Store implementation that records method invocations
// and returns configurable results for testing delegation behavior.
type mockStore struct {
	putCalled       bool
	getCalled       bool
	deleteCalled    bool
	existsCalled    bool
	watchCalled     bool
	watchTreeCalled bool
	newLockCalled   bool
	listCalled      bool
	deleteTreeCalled bool
	atomicPutCalled  bool
	atomicDelCalled  bool
	closeCalled     bool

	getReturn *store.KVPair
	getErr    error
	putErr    error
}

func (m *mockStore) Put(_ context.Context, _ string, _ []byte, _ *store.WriteOptions) error {
	m.putCalled = true
	return m.putErr
}

func (m *mockStore) Get(_ context.Context, _ string, _ *store.ReadOptions) (*store.KVPair, error) {
	m.getCalled = true
	return m.getReturn, m.getErr
}

func (m *mockStore) Delete(_ context.Context, _ string) error {
	m.deleteCalled = true
	return nil
}

func (m *mockStore) Exists(_ context.Context, _ string, _ *store.ReadOptions) (bool, error) {
	m.existsCalled = true
	return true, nil
}

func (m *mockStore) Watch(_ context.Context, _ string, _ *store.ReadOptions) (<-chan *store.KVPair, error) {
	m.watchCalled = true
	ch := make(chan *store.KVPair)
	return ch, nil
}

func (m *mockStore) WatchTree(_ context.Context, _ string, _ *store.ReadOptions) (<-chan []*store.KVPair, error) {
	m.watchTreeCalled = true
	ch := make(chan []*store.KVPair)
	return ch, nil
}

func (m *mockStore) NewLock(_ context.Context, _ string, _ *store.LockOptions) (store.Locker, error) {
	m.newLockCalled = true
	return nil, nil
}

func (m *mockStore) List(_ context.Context, _ string, _ *store.ReadOptions) ([]*store.KVPair, error) {
	m.listCalled = true
	return nil, nil
}

func (m *mockStore) DeleteTree(_ context.Context, _ string) error {
	m.deleteTreeCalled = true
	return nil
}

func (m *mockStore) AtomicPut(_ context.Context, _ string, _ []byte, _ *store.KVPair, _ *store.WriteOptions) (bool, *store.KVPair, error) {
	m.atomicPutCalled = true
	return false, nil, nil
}

func (m *mockStore) AtomicDelete(_ context.Context, _ string, _ *store.KVPair) (bool, error) {
	m.atomicDelCalled = true
	return false, nil
}

func (m *mockStore) Close() error {
	m.closeCalled = true
	return nil
}

func TestNew(t *testing.T) {
	ctx := context.Background()
	mock := &mockStore{}
	s := New(ctx, mock)
	require.NotNil(t, s)
	assert.NotNil(t, s.tracer)
	assert.Equal(t, mock, s.next)
}

func TestPut_Delegation(t *testing.T) {
	ctx := context.Background()
	mock := &mockStore{}
	s := New(ctx, mock)

	err := s.Put(ctx, "key", []byte("value"), nil)
	require.NoError(t, err)
	assert.True(t, mock.putCalled, "Put should delegate to inner store")
}

func TestPut_PropagatesError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("put failed")
	mock := &mockStore{putErr: expectedErr}
	s := New(ctx, mock)

	err := s.Put(ctx, "key", []byte("value"), nil)
	assert.ErrorIs(t, err, expectedErr)
}

func TestGet_Delegation(t *testing.T) {
	ctx := context.Background()
	expected := &store.KVPair{Key: "key", Value: []byte("value")}
	mock := &mockStore{getReturn: expected}
	s := New(ctx, mock)

	got, err := s.Get(ctx, "key", nil)
	require.NoError(t, err)
	assert.True(t, mock.getCalled, "Get should delegate to inner store")
	assert.Equal(t, expected, got)
}

func TestGet_PropagatesError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("not found")
	mock := &mockStore{getErr: expectedErr}
	s := New(ctx, mock)

	got, err := s.Get(ctx, "missing", nil)
	assert.ErrorIs(t, err, expectedErr)
	assert.Nil(t, got)
}

func TestDelete_Delegation(t *testing.T) {
	ctx := context.Background()
	mock := &mockStore{}
	s := New(ctx, mock)

	err := s.Delete(ctx, "key")
	require.NoError(t, err)
	assert.True(t, mock.deleteCalled, "Delete should delegate to inner store")
}

func TestExists_Delegation(t *testing.T) {
	ctx := context.Background()
	mock := &mockStore{}
	s := New(ctx, mock)

	exists, err := s.Exists(ctx, "key", nil)
	require.NoError(t, err)
	assert.True(t, exists)
	assert.True(t, mock.existsCalled, "Exists should delegate to inner store")
}

func TestWatch_Delegation(t *testing.T) {
	ctx := context.Background()
	mock := &mockStore{}
	s := New(ctx, mock)

	ch, err := s.Watch(ctx, "key", nil)
	require.NoError(t, err)
	assert.NotNil(t, ch)
	assert.True(t, mock.watchCalled, "Watch should delegate to inner store")
}

func TestWatchTree_Delegation(t *testing.T) {
	ctx := context.Background()
	mock := &mockStore{}
	s := New(ctx, mock)

	ch, err := s.WatchTree(ctx, "dir", nil)
	require.NoError(t, err)
	assert.NotNil(t, ch)
	assert.True(t, mock.watchTreeCalled, "WatchTree should delegate to inner store")
}

func TestNewLock_Delegation(t *testing.T) {
	ctx := context.Background()
	mock := &mockStore{}
	s := New(ctx, mock)

	_, err := s.NewLock(ctx, "key", nil)
	require.NoError(t, err)
	assert.True(t, mock.newLockCalled, "NewLock should delegate to inner store")
}

func TestList_Delegation(t *testing.T) {
	ctx := context.Background()
	mock := &mockStore{}
	s := New(ctx, mock)

	_, err := s.List(ctx, "dir", nil)
	require.NoError(t, err)
	assert.True(t, mock.listCalled, "List should delegate to inner store")
}

func TestDeleteTree_Delegation(t *testing.T) {
	ctx := context.Background()
	mock := &mockStore{}
	s := New(ctx, mock)

	err := s.DeleteTree(ctx, "dir")
	require.NoError(t, err)
	assert.True(t, mock.deleteTreeCalled, "DeleteTree should delegate to inner store")
}

func TestAtomicPut_Delegation(t *testing.T) {
	ctx := context.Background()
	mock := &mockStore{}
	s := New(ctx, mock)

	_, _, err := s.AtomicPut(ctx, "key", []byte("value"), nil, nil)
	require.NoError(t, err)
	assert.True(t, mock.atomicPutCalled, "AtomicPut should delegate to inner store")
}

func TestAtomicDelete_Delegation(t *testing.T) {
	ctx := context.Background()
	mock := &mockStore{}
	s := New(ctx, mock)

	_, err := s.AtomicDelete(ctx, "key", nil)
	require.NoError(t, err)
	assert.True(t, mock.atomicDelCalled, "AtomicDelete should delegate to inner store")
}

func TestClose_Delegation(t *testing.T) {
	ctx := context.Background()
	mock := &mockStore{}
	s := New(ctx, mock)

	err := s.Close()
	require.NoError(t, err)
	assert.True(t, mock.closeCalled, "Close should delegate to inner store")
}

func TestDelegation_AllMethods(t *testing.T) {
	// Table-driven test verifying every method delegates to the inner store
	ctx := context.Background()

	tests := []struct {
		name     string
		fn       func(s *Store, mock *mockStore)
		called   func(mock *mockStore) bool
	}{
		{
			name:   "Put",
			fn:     func(s *Store, _ *mockStore) { _ = s.Put(ctx, "k", []byte("v"), nil) },
			called: func(m *mockStore) bool { return m.putCalled },
		},
		{
			name:   "Get",
			fn:     func(s *Store, _ *mockStore) { _, _ = s.Get(ctx, "k", nil) },
			called: func(m *mockStore) bool { return m.getCalled },
		},
		{
			name:   "Delete",
			fn:     func(s *Store, _ *mockStore) { _ = s.Delete(ctx, "k") },
			called: func(m *mockStore) bool { return m.deleteCalled },
		},
		{
			name:   "Exists",
			fn:     func(s *Store, _ *mockStore) { _, _ = s.Exists(ctx, "k", nil) },
			called: func(m *mockStore) bool { return m.existsCalled },
		},
		{
			name:   "Watch",
			fn:     func(s *Store, _ *mockStore) { _, _ = s.Watch(ctx, "k", nil) },
			called: func(m *mockStore) bool { return m.watchCalled },
		},
		{
			name:   "WatchTree",
			fn:     func(s *Store, _ *mockStore) { _, _ = s.WatchTree(ctx, "dir", nil) },
			called: func(m *mockStore) bool { return m.watchTreeCalled },
		},
		{
			name:   "NewLock",
			fn:     func(s *Store, _ *mockStore) { _, _ = s.NewLock(ctx, "k", nil) },
			called: func(m *mockStore) bool { return m.newLockCalled },
		},
		{
			name:   "List",
			fn:     func(s *Store, _ *mockStore) { _, _ = s.List(ctx, "dir", nil) },
			called: func(m *mockStore) bool { return m.listCalled },
		},
		{
			name:   "DeleteTree",
			fn:     func(s *Store, _ *mockStore) { _ = s.DeleteTree(ctx, "dir") },
			called: func(m *mockStore) bool { return m.deleteTreeCalled },
		},
		{
			name:   "AtomicPut",
			fn:     func(s *Store, _ *mockStore) { _, _, _ = s.AtomicPut(ctx, "k", nil, nil, nil) },
			called: func(m *mockStore) bool { return m.atomicPutCalled },
		},
		{
			name:   "AtomicDelete",
			fn:     func(s *Store, _ *mockStore) { _, _ = s.AtomicDelete(ctx, "k", nil) },
			called: func(m *mockStore) bool { return m.atomicDelCalled },
		},
		{
			name:   "Close",
			fn:     func(s *Store, _ *mockStore) { _ = s.Close() },
			called: func(m *mockStore) bool { return m.closeCalled },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockStore{}
			s := New(ctx, mock)
			tc.fn(s, mock)
			assert.True(t, tc.called(mock), "%s should delegate to inner store", tc.name)
		})
	}
}
