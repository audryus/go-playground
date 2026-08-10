package mpt

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"sync"
)

// Store é onde os nós da trie vivem, endereçados pelo próprio hash.
// Como os nós são imutáveis (toda escrita cria um nó novo — estilo
// "copy-on-write" / persistent data structure), múltiplas versões da
// trie (múltiplas roots ao longo do tempo) podem compartilhar a mesma
// Store sem conflito: nós idênticos entre versões têm o mesmo hash e
// não precisam ser duplicados.
type Store interface {
	Get(h Hash) (*TrieNode, bool)
	Put(n *TrieNode) Hash
}

// MemStore é um Store em memória, seguro para uso concorrente.
type MemStore struct {
	mu    sync.RWMutex
	nodes map[Hash][]byte // guarda o gob já serializado
}

func NewMemStore() *MemStore {
	return &MemStore{nodes: make(map[Hash][]byte)}
}

func (s *MemStore) Get(h Hash) (*TrieNode, bool) {
	s.mu.RLock()
	raw, ok := s.nodes[h]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	n, err := decodeNode(raw)
	if err != nil {
		return nil, false
	}
	return n, true
}

func (s *MemStore) Put(n *TrieNode) Hash {
	raw := encodeNode(n)
	h := hashOf(raw)
	s.mu.Lock()
	s.nodes[h] = raw
	s.mu.Unlock()
	return h
}

// Len retorna quantos nós distintos existem na store — útil para observar
// o quanto de reaproveitamento (deduplicação) está acontecendo entre
// versões sucessivas da trie.
func (s *MemStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.nodes)
}

// --- Persistência --------------------------------------------------------

// snapshot é o formato gravado em disco: todos os nós da store, mais os
// "roots nomeados" que você quiser lembrar (ex.: "latest", ou um root por
// tick de jogo).
type storeSnapshot struct {
	Version int
	Nodes   map[Hash][]byte
	Roots   map[string]Hash
}

const storeSnapshotVersion = 1

// Save grava toda a store em disco, junto com um conjunto de roots
// nomeados (por exemplo {"latest": rootAtual}) para que o chamador saiba
// por onde recomeçar a navegar a trie após o reload.
func (s *MemStore) Save(path string, roots map[string]Hash) error {
	s.mu.RLock()
	nodesCopy := make(map[Hash][]byte, len(s.nodes))
	for h, raw := range s.nodes {
		nodesCopy[h] = raw
	}
	s.mu.RUnlock()

	snap := storeSnapshot{
		Version: storeSnapshotVersion,
		Nodes:   nodesCopy,
		Roots:   roots,
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(snap); err != nil {
		return fmt.Errorf("mpt: falha ao serializar store: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("mpt: falha ao escrever %q: %w", path, err)
	}
	return nil
}

// LoadMemStore lê uma store persistida e devolve os roots nomeados salvos
// junto com ela (ex.: para retomar de onde o servidor parou).
func LoadMemStore(path string) (*MemStore, map[string]Hash, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("mpt: falha ao ler %q: %w", path, err)
	}
	var snap storeSnapshot
	if err := gob.NewDecoder(bytes.NewReader(raw)).Decode(&snap); err != nil {
		return nil, nil, fmt.Errorf("mpt: falha ao desserializar store: %w", err)
	}
	if snap.Version != storeSnapshotVersion {
		return nil, nil, fmt.Errorf("mpt: versão de snapshot desconhecida: %d", snap.Version)
	}

	s := &MemStore{nodes: snap.Nodes}
	return s, snap.Roots, nil
}
