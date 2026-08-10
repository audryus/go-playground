// Package mpt implementa uma Merkle Patricia Trie (MPT) simplificada,
// no estilo da usada pelo Ethereum como "state trie" / "storage trie".
//
// A diferença fundamental para a Merkle tree "flat" (pacote merkletree)
// é que a MPT é uma estrutura chave->valor. Isso muda o que ela resolve:
//
//   - Merkle tree flat: você monta a árvore inteira de uma vez a partir de
//     uma lista fixa de dados. Para mudar 1 item, em teoria refaz a árvore
//     inteira (O(n) hashes) — a menos que você reimplemente a lógica de
//     atualização manualmente, o que a torna, na prática, quase uma MPT.
//   - MPT: update(key, value) é O(profundidade da chave), tipicamente
//     O(log n), porque só os nós no caminho da chave mudam. Todo o resto
//     da árvore permanece com o mesmo hash e é reaproveitado.
//
// Isso é exatamente o que você quer para um registro de drops de itens
// que muda a cada segundo: nunca reconstruir do zero.
package mpt

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
)

// Hash é um hash SHA-256 (32 bytes). O valor zero (Hash{}) é usado como
// sentinela de "nó vazio / não existe".
type Hash [32]byte

func (h Hash) String() string {
	return fmt.Sprintf("%x", h[:])
}

// IsZero indica se este é o hash sentinela de "nó ausente".
func (h Hash) IsZero() bool {
	return h == Hash{}
}

// NodeType identifica o tipo de nó da trie.
type NodeType uint8

const (
	// NodeLeaf guarda o restante do caminho de nibbles e o valor final.
	NodeLeaf NodeType = iota
	// NodeExtension comprime um trecho de caminho compartilhado por um
	// único filho (evita uma cadeia longa de branches com 1 filho só).
	NodeExtension
	// NodeBranch tem até 16 filhos (um por nibble 0x0-0xF) e,
	// opcionalmente, um valor terminal para quando uma chave termina
	// exatamente neste branch.
	NodeBranch
)

// TrieNode é a unidade de armazenamento da trie. O hash de um nó é
// calculado sobre sua codificação gob — cada nó é "content-addressed":
// ele é identificado e localizado pelo seu próprio hash.
//
// Nota de produção: implementações reais (ex.: go-ethereum) usam RLP em
// vez de gob para ter uma codificação canônica estável entre versões e
// linguagens diferentes. Aqui usamos gob por simplicidade didática; ele é
// determinístico o suficiente dentro do mesmo binário/versão de struct,
// mas não é um padrão de interoperabilidade entre sistemas.
type TrieNode struct {
	Type NodeType

	// Usado por Leaf e Extension: os nibbles restantes do caminho.
	Path []byte

	// Usado por Leaf (valor da chave) e por Branch (valor terminal,
	// quando uma chave termina exatamente neste branch).
	Value []byte

	// Usado por Extension: hash do único filho.
	Child Hash

	// Usado por Branch: hash de cada um dos 16 filhos possíveis
	// (Hash{} = filho ausente naquele nibble).
	Children [16]Hash
}

// encode serializa o nó de forma determinística (gob) para fins de hash e
// de armazenamento/prova.
func encodeNode(n *TrieNode) []byte {
	var buf bytes.Buffer
	// gob.Encode não falha para estes tipos (sem canais/funcs/interfaces),
	// então ignoramos o erro aqui de propósito.
	_ = gob.NewEncoder(&buf).Encode(n)
	return buf.Bytes()
}

func decodeNode(raw []byte) (*TrieNode, error) {
	var n TrieNode
	if err := gob.NewDecoder(bytes.NewReader(raw)).Decode(&n); err != nil {
		return nil, fmt.Errorf("mpt: falha ao decodificar nó: %w", err)
	}
	return &n, nil
}

func hashOf(raw []byte) Hash {
	return Hash(sha256.Sum256(raw))
}

// bytesToNibbles converte uma chave (bytes) em uma sequência de nibbles
// (metades de byte, valores 0-15). A trie opera sobre nibbles para ter
// fan-out 16 nos branches em vez de 256, o que reduz o custo de cada nó.
func bytesToNibbles(key []byte) []byte {
	nibbles := make([]byte, len(key)*2)
	for i, b := range key {
		nibbles[i*2] = b >> 4
		nibbles[i*2+1] = b & 0x0F
	}
	return nibbles
}

// commonPrefixLen retorna quantos nibbles iniciais são idênticos entre a e b.
func commonPrefixLen(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

func cloneNode(n *TrieNode) *TrieNode {
	c := *n
	if n.Path != nil {
		c.Path = append([]byte(nil), n.Path...)
	}
	if n.Value != nil {
		c.Value = append([]byte(nil), n.Value...)
	}
	// Children é um array (não slice), então `c := *n` já copia por valor.
	return &c
}
