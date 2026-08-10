package mpt

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestPutGetManyKeys(t *testing.T) {
	store := NewMemStore()
	root := EmptyRoot

	ref := make(map[string]string)
	rng := rand.New(rand.NewSource(42))

	// Insere 5000 chaves aleatórias e confere get após cada bloco de inserts.
	for i := 0; i < 5000; i++ {
		key := fmt.Sprintf("item-%d-%d", i, rng.Intn(1000))
		val := fmt.Sprintf("valor-%d", i)
		root = Put(store, root, []byte(key), []byte(val))
		ref[key] = val
	}

	for key, want := range ref {
		got, ok := Get(store, root, []byte(key))
		if !ok {
			t.Fatalf("chave %q não encontrada", key)
		}
		if string(got) != want {
			t.Fatalf("chave %q: got %q, want %q", key, got, want)
		}
	}

	missing, ok := Get(store, root, []byte("chave-que-nao-existe"))
	if ok || missing != nil {
		t.Fatalf("chave inexistente deveria retornar not-found")
	}
}

func TestUpdateExistingKey(t *testing.T) {
	store := NewMemStore()
	root := Put(store, EmptyRoot, []byte("a"), []byte("1"))
	root = Put(store, root, []byte("b"), []byte("2"))
	root = Put(store, root, []byte("a"), []byte("999")) // update

	v, ok := Get(store, root, []byte("a"))
	if !ok || string(v) != "999" {
		t.Fatalf("update falhou: got %q ok=%v", v, ok)
	}
	v, ok = Get(store, root, []byte("b"))
	if !ok || string(v) != "2" {
		t.Fatalf("chave não relacionada foi afetada: got %q ok=%v", v, ok)
	}
}

func TestProveAndVerify(t *testing.T) {
	store := NewMemStore()
	root := EmptyRoot
	keys := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"}
	for i, k := range keys {
		root = Put(store, root, []byte(k), []byte(fmt.Sprintf("v%d", i)))
	}

	for i, k := range keys {
		proof, found := Prove(store, root, []byte(k))
		if !found {
			t.Fatalf("prova para %q não encontrou a chave", k)
		}
		want := []byte(fmt.Sprintf("v%d", i))
		if !VerifyProof(root, []byte(k), want, proof) {
			t.Fatalf("VerifyProof falhou para chave legítima %q", k)
		}
		// Valor adulterado deve falhar.
		if VerifyProof(root, []byte(k), []byte("valor-forjado"), proof) {
			t.Fatalf("VerifyProof aceitou valor forjado para %q", k)
		}
	}

	// Chave nunca inserida.
	proof, found := Prove(store, root, []byte("nao-existe"))
	if found {
		t.Fatalf("prova encontrou chave que não foi inserida")
	}
	if VerifyProof(root, []byte("nao-existe"), []byte("qualquer"), proof) {
		t.Fatalf("VerifyProof aceitou prova de chave inexistente")
	}
}

func TestDeterministicRootIndependentOfInsertOrder(t *testing.T) {
	pairs := map[string]string{
		"k1": "v1", "k2": "v2", "k3": "v3", "k4": "v4", "k5": "v5",
	}

	build := func(order []string) Hash {
		store := NewMemStore()
		root := EmptyRoot
		for _, k := range order {
			root = Put(store, root, []byte(k), []byte(pairs[k]))
		}
		return root
	}

	rootA := build([]string{"k1", "k2", "k3", "k4", "k5"})
	rootB := build([]string{"k5", "k4", "k3", "k2", "k1"})

	if rootA != rootB {
		t.Fatalf("root deveria ser igual independente da ordem de inserção: %s vs %s", rootA, rootB)
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/trie.gob"

	store := NewMemStore()
	root := Put(store, EmptyRoot, []byte("item-1"), []byte("espada-lendaria"))
	root = Put(store, root, []byte("item-2"), []byte("escudo-comum"))

	if err := store.Save(path, map[string]Hash{"latest": root}); err != nil {
		t.Fatalf("save falhou: %v", err)
	}

	loaded, roots, err := LoadMemStore(path)
	if err != nil {
		t.Fatalf("load falhou: %v", err)
	}
	reloadedRoot := roots["latest"]
	if reloadedRoot != root {
		t.Fatalf("root recarregado difere do original")
	}

	v, ok := Get(loaded, reloadedRoot, []byte("item-1"))
	if !ok || string(v) != "espada-lendaria" {
		t.Fatalf("get pós-reload falhou: %q ok=%v", v, ok)
	}
}
