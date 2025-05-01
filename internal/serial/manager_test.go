package serial

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockPort demonstrates using testify mocks for serial.Port
// It can be used to intercept Read and Write calls in more advanced tests.
type MockPort struct {
	mock.Mock
}

func (m *MockPort) Read(p []byte) (int, error) {
	args := m.Called(p)
	// Optionally copy data into p if needed for tests
	if d := args.Get(0); d != nil {
		copy(p, d.([]byte))
	}
	return args.Int(1), args.Error(2)
}

func (m *MockPort) Write(p []byte) (int, error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

func (m *MockPort) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestRun_ImmediateCancel(t *testing.T) {
	// When context is already canceled, Run should return nil immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Add buffer size parameter to NewManager call
	mgr := NewManager("/dev/nonexistent", 9600, 4096)
	err := mgr.Run(ctx, make(chan []byte), make(chan []byte))
	require.NoError(t, err)
}

func TestOpenPort_InvalidDevice(t *testing.T) {
	m := NewManager("/dev/nonexistent", 9600, 4096).(*manager)
	_, err := m.openPort()
	require.Error(t, err)
}

func TestBufferPool(t *testing.T) {
	bufSize := 1024
	m := NewManager("/dev/test", 9600, bufSize).(*manager)

	// Get a buffer from the pool and verify its size
	buf := m.bufferPool.Get().([]byte)
	require.Equal(t, bufSize, len(buf))

	// Put it back and get another to confirm reuse
	m.bufferPool.Put(buf)
	buf2 := m.bufferPool.Get().([]byte)

	// Cannot guarantee the same buffer is returned, but size should match
	require.Equal(t, bufSize, len(buf2))
}

func TestMockPort_Write(t *testing.T) {
	mp := new(MockPort)
	data := []byte("hello")
	mp.On("Write", data).Return(len(data), nil)
	n, err := mp.Write(data)
	require.NoError(t, err)
	require.Equal(t, len(data), n)
	mp.AssertExpectations(t)
}
