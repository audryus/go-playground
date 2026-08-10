package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
)

// leafRecord é a unidade persistida por folha. Data pode ser nil se o
// chamador optar por persistir apenas os hashes (ver NewFromHashes) —
// útil quando você quer manter um "audit trail" verificável sem guardar
// o payload original (ex.: por espaço ou por política de retenção de dados).
type leafRecord struct {
	Hash Hash
	Data []byte
}

// snapshot é o formato serializado gravado em disco.
type snapshot struct {
	Version int
	Leaves  []leafRecord
}

const snapshotVersion = 1

// Save grava a árvore em disco. Se as folhas tiverem o campo Data
// preenchido (árvore criada via New), ele é persistido junto — permitindo
// reload completo com New(). Se a árvore foi criada via NewFromHashes
// (sem dados originais), apenas os hashes são gravados.
func (t *MerkleTree) Save(path string) error {
	snap := snapshot{
		Version: snapshotVersion,
		Leaves:  make([]leafRecord, len(t.Leaves)),
	}
	for i, leaf := range t.Leaves {
		snap.Leaves[i] = leafRecord{Hash: leaf.Hash, Data: leaf.Data}
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(snap); err != nil {
		return fmt.Errorf("merkletree: falha ao serializar snapshot: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("merkletree: falha ao escrever arquivo %q: %w", path, err)
	}
	return nil
}

// Load reconstrói uma MerkleTree a partir de um snapshot em disco.
//
// Se todos os registros tiverem Data presente, a árvore é reconstruída via
// New() (recalculando os hashes a partir dos dados — mais seguro, pois
// valida que os dados batem com os hashes gravados).
//
// Se algum registro não tiver Data (persistência "somente hash"), a árvore
// é reconstruída via NewFromHashes() — a estrutura e a Root ficam idênticas
// à original, mas GenerateProof funciona normalmente; só não dá pra
// verificar o conteúdo original porque ele não foi guardado.
func Load(path string) (*MerkleTree, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("merkletree: falha ao ler arquivo %q: %w", path, err)
	}

	var snap snapshot
	if err := gob.NewDecoder(bytes.NewReader(raw)).Decode(&snap); err != nil {
		return nil, fmt.Errorf("merkletree: falha ao desserializar snapshot: %w", err)
	}
	if snap.Version != snapshotVersion {
		return nil, fmt.Errorf("merkletree: versão de snapshot desconhecida: %d", snap.Version)
	}
	if len(snap.Leaves) == 0 {
		return nil, fmt.Errorf("merkletree: snapshot vazio")
	}

	allHaveData := true
	for _, l := range snap.Leaves {
		if l.Data == nil {
			allHaveData = false
			break
		}
	}

	if allHaveData {
		data := make([][]byte, len(snap.Leaves))
		for i, l := range snap.Leaves {
			data[i] = l.Data
		}
		tree, err := New(data)
		if err != nil {
			return nil, err
		}
		// Sanidade: os hashes recalculados devem bater com os gravados.
		for i, l := range snap.Leaves {
			if tree.Leaves[i].Hash != l.Hash {
				return nil, fmt.Errorf("merkletree: hash da folha %d não confere com o snapshot (dado corrompido ou adulterado)", i)
			}
		}
		return tree, nil
	}

	hashes := make([]Hash, len(snap.Leaves))
	for i, l := range snap.Leaves {
		hashes[i] = l.Hash
	}
	return NewFromHashes(hashes), nil
}
