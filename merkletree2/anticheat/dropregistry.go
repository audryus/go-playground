// Package anticheat demonstra um uso prático de Merkle Patricia Trie:
// um registro autoritativo de drops de itens em um jogo, mantido pelo
// servidor, onde:
//
//  1. Cada drop (item que cai de um monstro/baú/chão) é gravado com uma
//     chave única (DropID) e um valor determinístico (stats do item).
//  2. O servidor atualiza a trie a cada drop em O(profundidade da chave),
//     não O(total de drops) — importante porque isso acontece o tempo
//     todo, em todo o mapa.
//  3. O RootHash da trie num dado instante é um "resumo" de todo o
//     histórico de drops até ali. Pode ser logado, assinado, ou usado
//     como checkpoint de auditoria.
//  4. Quando um cliente afirma "eu peguei o item X com esses stats", o
//     servidor confere via Get() (mais simples, pois tem a trie completa)
//     ou, em uma arquitetura distribuída com múltiplos servidores de
//     mundo, via Merkle Proof contra o root que aquele shard publicou —
//     sem precisar replicar o histórico de drops inteiro entre shards.
//  5. Se o cliente reporta stats que não batem com o que foi realmente
//     dropado (ex.: tentando fazer um item "Comum" virar "Lendário"
//     editando memória/rede), a verificação falha e o servidor rejeita
//     o pickup / flag para revisão.
package anticheat

import (
	"encoding/json"
	"fmt"
	"myplayground/merkletree2/mpt"
	"time"
)

// ItemDrop é o valor associado a cada DropID na trie.
type ItemDrop struct {
	ItemID    string    `json:"item_id"`
	Rarity    string    `json:"rarity"`
	Attack    int       `json:"attack"`
	Defense   int       `json:"defense"`
	DroppedBy string    `json:"dropped_by"` // ex.: ID do monstro/baú
	DroppedAt time.Time `json:"dropped_at"`
	MapZone   string    `json:"map_zone"`
}

// Serialize converte o item para bytes de forma determinística (JSON com
// chaves fixas via struct tags), pois esse é o valor que entra na trie —
// precisa ser byte-idêntico sempre que o mesmo drop for serializado.
// Exportado porque um verificador externo (outro serviço, ou o exemplo de
// auditoria) precisa produzir o mesmo formato de bytes para conferir uma
// Merkle Proof com mpt.VerifyProof.
func (d ItemDrop) Serialize() []byte {
	raw, _ := json.Marshal(d)
	return raw
}

func (d ItemDrop) serialize() []byte {
	return d.Serialize()
}

func deserialize(raw []byte) (ItemDrop, error) {
	var d ItemDrop
	err := json.Unmarshal(raw, &d)
	return d, err
}

// DropRegistry é o registro autoritativo mantido pelo servidor de jogo.
type DropRegistry struct {
	store mpt.Store
	root  mpt.Hash
}

// NewDropRegistry cria um registro vazio, pronto para receber drops.
func NewDropRegistry() *DropRegistry {
	return &DropRegistry{
		store: mpt.NewMemStore(),
		root:  mpt.EmptyRoot,
	}
}

// RegisterDrop grava um novo drop autoritativo e retorna o novo RootHash
// (o "checkpoint" do estado do mundo após este drop). Custo O(len(dropID)),
// não O(total de drops já registrados).
func (r *DropRegistry) RegisterDrop(dropID string, item ItemDrop) mpt.Hash {
	r.root = mpt.Put(r.store, r.root, []byte(dropID), item.serialize())
	return r.root
}

// RootHash retorna o checkpoint atual — o resumo criptográfico de todos
// os drops já registrados.
func (r *DropRegistry) RootHash() mpt.Hash {
	return r.root
}

// VerifyPickup confere se o item que o cliente está tentando pegar
// corresponde exatamente ao que foi realmente dropado pelo servidor.
// Retorna (itemReal, ok). ok=false cobre tanto "esse dropID nunca existiu"
// quanto "o servidor tem esse dropID mas os dados batidos pelo cliente
// não conferem" — ambos os casos merecem rejeitar o pickup.
func (r *DropRegistry) VerifyPickup(dropID string, claimed ItemDrop) (ItemDrop, bool) {
	raw, found := mpt.Get(r.store, r.root, []byte(dropID))
	if !found {
		return ItemDrop{}, false
	}
	real, err := deserialize(raw)
	if err != nil {
		return ItemDrop{}, false
	}

	claimedRaw := claimed.serialize()
	realRaw := real.serialize()
	if string(claimedRaw) != string(realRaw) {
		return real, false
	}
	return real, true
}

// ProvePickup gera uma prova de que dropID tem exatamente os stats
// fornecidos, contra o root atual. Isso é o que um shard/servidor regional
// mandaria para outro sistema (ex.: um serviço de auditoria central, ou
// outro servidor de mundo) validar sem precisar replicar toda a trie.
func (r *DropRegistry) ProvePickup(dropID string) (*mpt.Proof, ItemDrop, bool) {
	raw, found := mpt.Get(r.store, r.root, []byte(dropID))
	if !found {
		return nil, ItemDrop{}, false
	}
	item, err := deserialize(raw)
	if err != nil {
		return nil, ItemDrop{}, false
	}
	proof, ok := mpt.Prove(r.store, r.root, []byte(dropID))
	return proof, item, ok
}

// Save persiste a store inteira + o root atual em disco — usado para
// recuperação após queda do servidor (o histórico de drops não pode ser
// perdido, senão os clientes ficam com itens que o servidor "esqueceu").
func (r *DropRegistry) Save(path string) error {
	return r.store.(*mpt.MemStore).Save(path, map[string]mpt.Hash{"latest": r.root})
}

// LoadDropRegistry recarrega um registro persistido anteriormente.
func LoadDropRegistry(path string) (*DropRegistry, error) {
	store, roots, err := mpt.LoadMemStore(path)
	if err != nil {
		return nil, fmt.Errorf("anticheat: falha ao carregar registro: %w", err)
	}
	root, ok := roots["latest"]
	if !ok {
		return nil, fmt.Errorf("anticheat: snapshot não contém root 'latest'")
	}
	return &DropRegistry{store: store, root: root}, nil
}
