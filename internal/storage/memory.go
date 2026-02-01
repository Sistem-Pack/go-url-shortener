package storage

type URLStorage interface {
	Set(id, original string)
	Get(id string) (string, bool)
}

type Memory struct {
	data map[string]string
}

func NewMemory() *Memory {
	return &Memory{data: make(map[string]string)}
}

func (m *Memory) Set(id, original string) {
	m.data[id] = original
}

func (m *Memory) Get(id string) (string, bool) {
	val, ok := m.data[id]
	return val, ok
}
