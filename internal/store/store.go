package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Category string

const (
	CategoryFact         Category = "fact"
	CategoryDiscovery    Category = "discovery"
	CategoryPreference   Category = "preference"
	CategoryInstruction  Category = "instruction"
)

type MemoryItem struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Category  Category  `json:"category"`
	Source    string    `json:"source"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Store struct {
	mu       sync.RWMutex
	filePath string
	items    map[string]MemoryItem
}

func New(filePath string) (*Store, error) {
	if filePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("home dir: %w", err)
		}
		filePath = filepath.Join(home, ".webcli", "memory.json")
	}

	s := &Store{
		filePath: filePath,
		items:    make(map[string]MemoryItem),
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}

	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load store: %w", err)
	}

	return s, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.items)
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return os.WriteFile(s.filePath, data, 0644)
}

func (s *Store) Save(key, value string, opts ...SaveOption) (MemoryItem, error) {
	if key == "" {
		return MemoryItem{}, fmt.Errorf("key cannot be empty")
	}
	if value == "" {
		return MemoryItem{}, fmt.Errorf("value cannot be empty")
	}

	cfg := saveConfig{
		category: CategoryFact,
		source:   "",
		tags:     nil,
	}
	for _, o := range opts {
		o(&cfg)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	item := MemoryItem{
		Key:       key,
		Value:     value,
		Category:  cfg.category,
		Source:    cfg.source,
		Tags:      cfg.tags,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if existing, ok := s.items[key]; ok {
		item.CreatedAt = existing.CreatedAt
	}

	if item.Tags == nil {
		item.Tags = []string{}
	}

	s.items[key] = item

	if err := s.save(); err != nil {
		delete(s.items, key)
		return MemoryItem{}, fmt.Errorf("persist: %w", err)
	}

	return item, nil
}

func (s *Store) SaveDiscovery(key, value, source string, tags []string) (MemoryItem, error) {
	return s.Save(key, value, WithCategory(CategoryDiscovery), WithSource(source), WithTags(tags))
}

func (s *Store) Get(key string) (MemoryItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[key]
	if !ok {
		return MemoryItem{}, fmt.Errorf("memory '%s' not found", key)
	}
	return item, nil
}

func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[key]; !ok {
		return fmt.Errorf("memory '%s' not found", key)
	}

	delete(s.items, key)
	return s.save()
}

func (s *Store) List() []MemoryItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]MemoryItem, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return items
}

func (s *Store) Search(query string) []MemoryItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(query)
	var matches []MemoryItem
	for _, item := range s.items {
		if strings.Contains(strings.ToLower(item.Key), query) ||
			strings.Contains(strings.ToLower(item.Value), query) ||
			stringContains(strings.ToLower(item.Source), query) ||
			tagContains(item.Tags, query) {
			matches = append(matches, item)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].UpdatedAt.After(matches[j].UpdatedAt)
	})
	return matches
}

func (s *Store) GetContext(topic string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if topic == "" {
		items := s.List()
		if len(items) == 0 {
			return ""
		}
		var b strings.Builder
		b.WriteString("Here is the shared knowledge I have gathered:\n\n")
		for _, item := range items {
			writeContextItem(&b, item)
		}
		return b.String()
	}

	query := strings.ToLower(topic)
	var relevant []MemoryItem
	for _, item := range s.items {
		if strings.Contains(strings.ToLower(item.Key), query) ||
			strings.Contains(strings.ToLower(item.Value), query) ||
			strings.Contains(strings.ToLower(item.Source), query) ||
			tagContains(item.Tags, query) {
			relevant = append(relevant, item)
		}
	}

	if len(relevant) == 0 {
		return ""
	}

	sort.Slice(relevant, func(i, j int) bool {
		return relevant[i].UpdatedAt.After(relevant[j].UpdatedAt)
	})

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Relevant shared knowledge about \"%s\":\n\n", topic))
	for _, item := range relevant {
		writeContextItem(&b, item)
	}
	return b.String()
}

func writeContextItem(b *strings.Builder, item MemoryItem) {
	b.WriteString(fmt.Sprintf("[%s] %s", item.Category, item.Key))
	if item.Source != "" {
		b.WriteString(fmt.Sprintf(" (source: %s)", item.Source))
	}
	b.WriteString("\n")
	b.WriteString(item.Value)
	if !strings.HasSuffix(item.Value, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

func (s *Store) Path() string {
	return s.filePath
}

type saveConfig struct {
	category Category
	source   string
	tags     []string
}

type SaveOption func(*saveConfig)

func WithCategory(c Category) SaveOption {
	return func(cfg *saveConfig) {
		cfg.category = c
	}
}

func WithSource(source string) SaveOption {
	return func(cfg *saveConfig) {
		cfg.source = source
	}
}

func WithTags(tags []string) SaveOption {
	return func(cfg *saveConfig) {
		cfg.tags = tags
	}
}

func stringContains(s, substr string) bool {
	return s != "" && strings.Contains(s, substr)
}

func tagContains(tags []string, query string) bool {
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}
