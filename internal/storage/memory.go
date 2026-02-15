package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type URLStorage interface {
	Set(id, original string) error
	Get(id string) (string, bool)
}

type Memory struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewMemory() *Memory {
	return &Memory{
		data: make(map[string]string),
	}
}

func (m *Memory) Set(id, original string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[id] = original
	return nil
}

func (m *Memory) Get(id string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.data[id]
	return val, ok
}

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type FileStorage struct {
	mu     sync.RWMutex
	data   map[string]string
	file   string
	lastID int
}

func NewFileStorage(filePath string) (*FileStorage, error) {
	fs := &FileStorage{
		data: make(map[string]string),
		file: filePath,
	}

	if err := fs.loadFromFile(); err != nil {
		return nil, err
	}

	return fs, nil
}

func (fs *FileStorage) Get(short string) (string, bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	val, ok := fs.data[short]
	return val, ok
}

func (fs *FileStorage) Set(short string, original string) error {
	fs.mu.Lock()

	fs.lastID++
	fs.data[short] = original

	dataCopy := make(map[string]string, len(fs.data))
	for k, v := range fs.data {
		dataCopy[k] = v
	}
	fs.mu.Unlock()

	return fs.saveToFile(dataCopy)
}

func (fs *FileStorage) saveToFile(data map[string]string) error {
	records := make([]URLRecord, 0, len(data))
	counter := 1
	for short, original := range data {
		records = append(records, URLRecord{
			UUID:        fmt.Sprintf("%d", counter),
			ShortURL:    short,
			OriginalURL: original,
		})
		counter++
	}

	file, err := os.Create(fs.file)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(records)
}

func (fs *FileStorage) loadFromFile() error {
	file, err := os.Open(fs.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	var records []URLRecord
	if err := json.NewDecoder(file).Decode(&records); err != nil {
		return err
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	maxID := 0
	for _, rec := range records {
		fs.data[rec.ShortURL] = rec.OriginalURL

		var uid int
		fmt.Sscanf(rec.UUID, "%d", &uid)
		if uid > maxID {
			maxID = uid
		}
	}
	fs.lastID = maxID

	return nil
}
