// Package merkletree implementa uma Merkle Tree binária genérica,
// com suporte a construção, geração de Merkle Root, provas de inclusão
// (Merkle Proof) e verificação dessas provas.
package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
)

// Hash representa um hash SHA-256 (32 bytes).
type Hash [32]byte

// String facilita debug/print do hash em hexadecimal.
func (h Hash) String() string {
	return fmt.Sprintf("%x", h[:])
}

// Node representa um nó da árvore (folha ou interno).
type Node struct {
	Left, Right *Node
	Parent      *Node
	Hash        Hash
	// Data guarda o dado original apenas para folhas (opcional, útil para debug).
	Data []byte
	// IsLeaf indica se este nó é uma folha.
	IsLeaf bool
}

// MerkleTree representa a árvore completa.
type MerkleTree struct {
	Root   *Node
	Leaves []*Node
}

// leafHash calcula o hash de uma folha.
// Usamos um prefixo 0x00 para diferenciar hash de folha de hash de nó interno
// (0x01), prevenindo ataques de "second preimage" onde um nó interno poderia
// ser confundido com uma folha (ataque clássico em Merkle trees ingênuas).
func leafHash(data []byte) Hash {
	h := sha256.New()
	h.Write([]byte{0x00})
	h.Write(data)
	var out Hash
	copy(out[:], h.Sum(nil))
	return out
}

// nodeHash calcula o hash de um nó interno a partir dos hashes dos filhos.
func nodeHash(left, right Hash) Hash {
	h := sha256.New()
	h.Write([]byte{0x01})
	h.Write(left[:])
	h.Write(right[:])
	var out Hash
	copy(out[:], h.Sum(nil))
	return out
}

// New constrói uma Merkle Tree a partir de uma lista de blocos de dados.
// Retorna erro se a lista estiver vazia.
func New(data [][]byte) (*MerkleTree, error) {
	if len(data) == 0 {
		return nil, errors.New("merkletree: não é possível construir árvore sem dados")
	}

	// 1. Cria as folhas
	leaves := make([]*Node, len(data))
	for i, d := range data {
		leaves[i] = &Node{
			Hash:   leafHash(d),
			Data:   d,
			IsLeaf: true,
		}
	}

	// 2. Constrói os níveis acima até chegar na raiz
	level := leaves
	for len(level) > 1 {
		var nextLevel []*Node

		for i := 0; i < len(level); i += 2 {
			left := level[i]

			// Se o nível tem número ímpar de nós, duplicamos o último
			// (estratégia comum usada pelo Bitcoin para lidar com folhas ímpares).
			var right *Node
			if i+1 < len(level) {
				right = level[i+1]
			} else {
				right = level[i]
			}

			parent := &Node{
				Left:  left,
				Right: right,
				Hash:  nodeHash(left.Hash, right.Hash),
			}
			left.Parent = parent
			if right != left {
				right.Parent = parent
			}

			nextLevel = append(nextLevel, parent)
		}

		level = nextLevel
	}

	return &MerkleTree{
		Root:   level[0],
		Leaves: leaves,
	}, nil
}

// RootHash retorna o hash raiz (o "resumo" de todos os dados).
func (t *MerkleTree) RootHash() Hash {
	return t.Root.Hash
}

// ProofStep representa um passo de uma Merkle Proof: um hash "irmão"
// e de que lado ele fica (para saber a ordem de concatenação).
type ProofStep struct {
	Hash    Hash
	IsRight bool // true se este hash é o irmão à direita do nó atual
}

// MerkleProof é o caminho de hashes necessário para provar que um dado
// específico pertence à árvore, sem revelar os outros dados.
type MerkleProof struct {
	LeafHash Hash
	Steps    []ProofStep
}

// GenerateProof gera a prova de inclusão para o dado na posição `index`.
func (t *MerkleTree) GenerateProof(index int) (*MerkleProof, error) {
	if index < 0 || index >= len(t.Leaves) {
		return nil, fmt.Errorf("merkletree: índice %d fora do intervalo [0, %d)", index, len(t.Leaves))
	}

	node := t.Leaves[index]
	proof := &MerkleProof{LeafHash: node.Hash}

	for node.Parent != nil {
		parent := node.Parent

		if parent.Left == node {
			// nosso nó é o filho esquerdo; o irmão é o direito
			proof.Steps = append(proof.Steps, ProofStep{
				Hash:    parent.Right.Hash,
				IsRight: true,
			})
		} else {
			proof.Steps = append(proof.Steps, ProofStep{
				Hash:    parent.Left.Hash,
				IsRight: false,
			})
		}

		node = parent
	}

	return proof, nil
}

// VerifyProof verifica se uma prova é válida para um determinado dado
// e uma raiz esperada — sem precisar da árvore inteira em memória.
func VerifyProof(data []byte, proof *MerkleProof, expectedRoot Hash) bool {
	current := leafHash(data)

	if !bytes.Equal(current[:], proof.LeafHash[:]) {
		return false
	}

	for _, step := range proof.Steps {
		if step.IsRight {
			current = nodeHash(current, step.Hash)
		} else {
			current = nodeHash(step.Hash, current)
		}
	}

	return bytes.Equal(current[:], expectedRoot[:])
}

// Diff compara duas Merkle trees e retorna os índices das folhas que
// diferem, aproveitando a estrutura da árvore para pular subárvores
// idênticas rapidamente (sincronização eficiente).
// Assume que ambas as árvores têm o mesmo número de folhas.
func Diff(a, b *MerkleTree) ([]int, error) {
	if len(a.Leaves) != len(b.Leaves) {
		return nil, errors.New("merkletree: árvores com número diferente de folhas não são comparáveis por esta função")
	}

	if a.Root.Hash == b.Root.Hash {
		return nil, nil // raízes iguais => dados idênticos, nada a fazer
	}

	var diffIdx []int
	var walk func(na, nb *Node, leafIdx *int)
	walk = func(na, nb *Node, leafIdx *int) {
		if na.Hash == nb.Hash {
			// Subárvores idênticas: pula todas as folhas dela de uma vez.
			*leafIdx += countLeaves(na)
			return
		}
		if na.IsLeaf || nb.IsLeaf {
			diffIdx = append(diffIdx, *leafIdx)
			*leafIdx++
			return
		}
		walk(na.Left, nb.Left, leafIdx)
		walk(na.Right, nb.Right, leafIdx)
	}

	idx := 0
	walk(a.Root, b.Root, &idx)
	return diffIdx, nil
}

func countLeaves(n *Node) int {
	if n.IsLeaf {
		return 1
	}
	total := 0
	if n.Left != nil {
		total += countLeaves(n.Left)
	}
	if n.Right != nil && n.Right != n.Left {
		total += countLeaves(n.Right)
	}
	return total
}
