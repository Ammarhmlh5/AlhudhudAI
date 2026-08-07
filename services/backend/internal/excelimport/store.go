package excelimport

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// PreviewRow is a single normalized row shown to the user before confirmation.
type PreviewRow struct {
	Phone   string `json:"phone"`
	Content string `json:"content"`
	Valid   bool   `json:"valid"`
	Reason  string `json:"reason,omitempty"`
}

// Preview holds the full normalized result of an upload so the app can confirm
// before anything is actually ingested.
type Preview struct {
	Token      string
	FileName   string
	Rows       []PreviewRow
	TotalRows  int
	Valid      int
	Invalid    int
	Duplicates int
	CreatedAt  time.Time
}

// Store is an in-memory, TTL-bounded store for upload previews. No database
// schema changes are needed for this feature.
type Store struct {
	mu   sync.Mutex
	m    map[string]*Preview
	ttl  time.Duration
	stop chan struct{}
}

func NewStore() *Store {
	s := &Store{
		m:    make(map[string]*Preview),
		ttl:  30 * time.Minute,
		stop: make(chan struct{}),
	}
	go s.cleanupLoop()
	return s
}

func newToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "ex_" + hex.EncodeToString([]byte(time.Now().Format("150405.000000")))
	}
	return "ex_" + hex.EncodeToString(b)
}

func (s *Store) Set(p Preview) *Preview {
	p.Token = newToken()
	p.CreatedAt = time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := p
	s.m[p.Token] = &cp
	return &cp
}

func (s *Store) Get(token string) (*Preview, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.m[token]
	if !ok {
		return nil, false
	}
	if time.Since(p.CreatedAt) > s.ttl {
		delete(s.m, token)
		return nil, false
	}
	return p, true
}

func (s *Store) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, token)
}

func (s *Store) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			now := time.Now()
			s.mu.Lock()
			for k, p := range s.m {
				if now.Sub(p.CreatedAt) > s.ttl {
					delete(s.m, k)
				}
			}
			s.mu.Unlock()
		}
	}
}
