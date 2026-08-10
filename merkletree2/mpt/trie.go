package mpt

import "bytes"

// EmptyRoot é o root de uma trie vazia (nenhuma chave inserida).
var EmptyRoot = Hash{}

// Put insere ou atualiza key->value na trie cujo root atual é `root`, e
// retorna o novo root. Não modifica nós existentes — cria nós novos ao
// longo do caminho da chave e reaproveita (por hash) tudo que não mudou.
// Custo: O(len(key)) nós novos, não O(n) como reconstruir uma flat tree.
func Put(store Store, root Hash, key, value []byte) Hash {
	path := bytesToNibbles(key)
	return insert(store, root, path, value)
}

func insert(store Store, nodeHash Hash, path, value []byte) Hash {
	if nodeHash.IsZero() {
		// Não há nó aqui ainda: cria uma folha nova com o caminho restante.
		leaf := &TrieNode{Type: NodeLeaf, Path: append([]byte(nil), path...), Value: value}
		return store.Put(leaf)
	}

	node, ok := store.Get(nodeHash)
	if !ok {
		// Hash referenciado mas ausente na store: dado corrompido/faltando.
		// Em produção isso mereceria um erro tipado; aqui simplificamos
		// tratando como se fosse um nó vazio (comportamento definido, mas
		// sinaliza um bug de integração se acontecer).
		leaf := &TrieNode{Type: NodeLeaf, Path: append([]byte(nil), path...), Value: value}
		return store.Put(leaf)
	}

	switch node.Type {
	case NodeLeaf:
		return insertAtLeaf(store, node, path, value)
	case NodeExtension:
		return insertAtExtension(store, node, path, value)
	case NodeBranch:
		return insertAtBranch(store, node, path, value)
	default:
		panic("mpt: tipo de nó desconhecido")
	}
}

func insertAtLeaf(store Store, leaf *TrieNode, path, value []byte) Hash {
	common := commonPrefixLen(leaf.Path, path)

	// Mesma chave exata: só substitui o valor.
	if common == len(leaf.Path) && common == len(path) {
		newLeaf := &TrieNode{Type: NodeLeaf, Path: leaf.Path, Value: value}
		return store.Put(newLeaf)
	}

	// Caminhos divergem em algum ponto: precisa de um branch aqui.
	branch := &TrieNode{Type: NodeBranch}

	placeAt(store, branch, leaf.Path, leaf.Value, common, true)
	placeAt(store, branch, path, value, common, false)

	branchHash := store.Put(branch)
	if common == 0 {
		return branchHash
	}
	ext := &TrieNode{Type: NodeExtension, Path: path[:common], Child: branchHash}
	return store.Put(ext)
}

func insertAtExtension(store Store, ext *TrieNode, path, value []byte) Hash {
	common := commonPrefixLen(ext.Path, path)

	if common == len(ext.Path) {
		// O caminho novo cobre toda a extensão: desce e recursa no filho
		// com o restante do caminho.
		remaining := path[common:]
		newChildHash := insert(store, ext.Child, remaining, value)
		newExt := &TrieNode{Type: NodeExtension, Path: ext.Path, Child: newChildHash}
		return store.Put(newExt)
	}

	// Diverge no meio da extensão: precisa quebrar a extensão em um branch.
	branch := &TrieNode{Type: NodeBranch}

	// Parte restante da extensão original aponta pro filho antigo dela.
	extRemaining := ext.Path[common+1:]
	extIdx := ext.Path[common]
	var childForExt Hash
	if len(extRemaining) == 0 {
		childForExt = ext.Child
	} else {
		childForExt = store.Put(&TrieNode{Type: NodeExtension, Path: extRemaining, Child: ext.Child})
	}
	branch.Children[extIdx] = childForExt

	placeAt(store, branch, path, value, common, false)

	branchHash := store.Put(branch)
	if common == 0 {
		return branchHash
	}
	newExt := &TrieNode{Type: NodeExtension, Path: path[:common], Child: branchHash}
	return store.Put(newExt)
}

func insertAtBranch(store Store, branch *TrieNode, path, value []byte) Hash {
	newBranch := cloneNode(branch)

	if len(path) == 0 {
		// A chave termina exatamente neste branch.
		newBranch.Value = value
		return store.Put(newBranch)
	}

	idx := path[0]
	remaining := path[1:]
	childHash := branch.Children[idx] // Hash{} se não existir ainda
	newChildHash := insert(store, childHash, remaining, value)
	newBranch.Children[idx] = newChildHash
	return store.Put(newBranch)
}

// placeAt insere (path, value) dentro de um branch recém-criado, a partir
// da posição `common` do caminho em diante. isExisting indica se este é o
// nó que já existia (usado só para clareza; o comportamento é o mesmo).
func placeAt(store Store, branch *TrieNode, path, value []byte, common int, isExisting bool) {
	if common == len(path) {
		branch.Value = value
		return
	}
	idx := path[common]
	remaining := path[common+1:]
	leafHash := store.Put(&TrieNode{Type: NodeLeaf, Path: append([]byte(nil), remaining...), Value: value})
	branch.Children[idx] = leafHash
}

// Get busca o valor associado à chave, navegando a trie a partir de root.
func Get(store Store, root Hash, key []byte) ([]byte, bool) {
	path := bytesToNibbles(key)
	current := root

	for {
		if current.IsZero() {
			return nil, false
		}
		node, ok := store.Get(current)
		if !ok {
			return nil, false
		}

		switch node.Type {
		case NodeLeaf:
			if bytes.Equal(node.Path, path) {
				return node.Value, true
			}
			return nil, false

		case NodeExtension:
			if len(path) < len(node.Path) || !bytes.Equal(path[:len(node.Path)], node.Path) {
				return nil, false
			}
			path = path[len(node.Path):]
			current = node.Child

		case NodeBranch:
			if len(path) == 0 {
				if node.Value != nil {
					return node.Value, true
				}
				return nil, false
			}
			idx := path[0]
			path = path[1:]
			current = node.Children[idx]
		}
	}
}

// Proof é o conjunto de nós, do root até a folha, necessário para provar
// (ou refutar) que uma chave tem um determinado valor numa trie com um
// dado root — sem precisar consultar a Store inteira.
type Proof struct {
	Nodes [][]byte // cada elemento é a codificação bruta (gob) de um nó
}

// Prove monta a prova de inclusão (ou de ausência) de `key`.
func Prove(store Store, root Hash, key []byte) (*Proof, bool) {
	path := bytesToNibbles(key)
	current := root
	var proof Proof

	for {
		if current.IsZero() {
			return &proof, false
		}
		node, ok := store.Get(current)
		if !ok {
			return &proof, false
		}
		proof.Nodes = append(proof.Nodes, encodeNode(node))

		switch node.Type {
		case NodeLeaf:
			return &proof, bytes.Equal(node.Path, path)

		case NodeExtension:
			if len(path) < len(node.Path) || !bytes.Equal(path[:len(node.Path)], node.Path) {
				return &proof, false
			}
			path = path[len(node.Path):]
			current = node.Child

		case NodeBranch:
			if len(path) == 0 {
				return &proof, node.Value != nil
			}
			idx := path[0]
			path = path[1:]
			current = node.Children[idx]
		}
	}
}

// VerifyProof confere, de forma independente da Store, que `key` tem
// exatamente `value` numa trie cujo root é `expectedRoot` — recomputando
// os hashes a partir dos nós fornecidos na prova. Isso é o que permite a
// um cliente (ex.: outro servidor de jogo, ou um auditor externo) validar
// uma alegação sem ter a trie inteira.
func VerifyProof(expectedRoot Hash, key, value []byte, proof *Proof) bool {
	path := bytesToNibbles(key)
	current := expectedRoot

	for _, raw := range proof.Nodes {
		if hashOf(raw) != current {
			return false // nó não bate com o hash esperado nesta posição
		}
		node, err := decodeNode(raw)
		if err != nil {
			return false
		}

		switch node.Type {
		case NodeLeaf:
			return bytes.Equal(node.Path, path) && bytes.Equal(node.Value, value)

		case NodeExtension:
			if len(path) < len(node.Path) || !bytes.Equal(path[:len(node.Path)], node.Path) {
				return false
			}
			path = path[len(node.Path):]
			current = node.Child

		case NodeBranch:
			if len(path) == 0 {
				return bytes.Equal(node.Value, value)
			}
			idx := path[0]
			path = path[1:]
			current = node.Children[idx]
		}
	}

	return false
}
