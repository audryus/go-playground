package main

import (
	"fmt"
	"myplayground/merkletree2/anticheat"
	"myplayground/merkletree2/mpt"
	"os"
	"runtime"
	"strings"
	"time"
)

func main() {
	exemplo1RootHashBasico()
	exemplo2ProvaDeInclusao()
	exemplo3DeteccaoDeAdulteracao()
	exemplo4DiffEntreDuasArvores()
	exemplo5SimulandoBlocoDeTransacoes()
	exemplo6Persistencia()
	exemplo7ParalelizacaoBenchmark()
	exemplo8MPTBasico()
	exemplo9AntiCheatDropDeItens()
	exemplo10PersistenciaEDisputaDeAuditoria()
}

// -----------------------------------------------------------------------
// Exemplo 1: construir a árvore e obter o hash raiz
// -----------------------------------------------------------------------
func exemplo1RootHashBasico() {
	section("Exemplo 1: Root Hash básico")

	dados := [][]byte{
		[]byte("arquivo1.txt"),
		[]byte("arquivo2.txt"),
		[]byte("arquivo3.txt"),
		[]byte("arquivo4.txt"),
	}

	tree, err := New(dados)
	if err != nil {
		panic(err)
	}

	fmt.Println("Merkle Root:", tree.RootHash())
	fmt.Println("Número de folhas:", len(tree.Leaves))
}

// -----------------------------------------------------------------------
// Exemplo 2: gerar e verificar uma Merkle Proof (prova de inclusão)
// Isso simula, por exemplo, um cliente leve (light client) que confia
// apenas na Merkle Root e quer verificar se um dado específico
// realmente pertence ao conjunto — sem baixar tudo.
// -----------------------------------------------------------------------
func exemplo2ProvaDeInclusao() {
	section("Exemplo 2: Prova de inclusão (Merkle Proof)")

	dados := [][]byte{
		[]byte("tx: Alice -> Bob: 10 BTC"),
		[]byte("tx: Bob -> Carol: 5 BTC"),
		[]byte("tx: Carol -> Dave: 2 BTC"),
		[]byte("tx: Dave -> Eve: 1 BTC"),
		[]byte("tx: Eve -> Frank: 0.5 BTC"),
	}

	tree, err := New(dados)
	if err != nil {
		panic(err)
	}

	root := tree.RootHash()

	// Queremos provar que a transação de índice 2 ("Carol -> Dave")
	// está incluída na árvore, sem revelar as outras transações.
	idx := 2
	proof, err := tree.GenerateProof(idx)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Provando inclusão de: %q\n", dados[idx])
	fmt.Println("Passos na prova (log2(n) aprox.):", len(proof.Steps))

	// O "verificador" só precisa: do dado, da prova e da root — não da árvore inteira.
	valido := VerifyProof(dados[idx], proof, root)
	fmt.Println("Prova válida?", valido)

	// Tentando provar um dado que foi adulterado -> deve falhar
	dadoFalso := []byte("tx: Carol -> Dave: 999999 BTC")
	valido2 := VerifyProof(dadoFalso, proof, root)
	fmt.Println("Prova válida para dado adulterado?", valido2)
}

// -----------------------------------------------------------------------
// Exemplo 3: detectar se algum dado foi alterado comparando apenas
// a Merkle Root (útil para verificação de integridade de arquivos/backups)
// -----------------------------------------------------------------------
func exemplo3DeteccaoDeAdulteracao() {
	section("Exemplo 3: Detecção de adulteração via Root Hash")

	original := [][]byte{
		[]byte("bloco de dados 1"),
		[]byte("bloco de dados 2"),
		[]byte("bloco de dados 3"),
	}

	adulterado := [][]byte{
		[]byte("bloco de dados 1"),
		[]byte("bloco de dados 2 - MODIFICADO"),
		[]byte("bloco de dados 3"),
	}

	treeOriginal, _ := New(original)
	treeAdulterado, _ := New(adulterado)

	fmt.Println("Root original:  ", treeOriginal.RootHash())
	fmt.Println("Root adulterada:", treeAdulterado.RootHash())
	fmt.Println("Dados íntegros?", treeOriginal.RootHash() == treeAdulterado.RootHash())
}

// -----------------------------------------------------------------------
// Exemplo 4: usar Diff para descobrir exatamente quais folhas mudaram
// entre duas versões de um conjunto de dados — útil para sincronização
// eficiente entre réplicas (ex.: Cassandra, DynamoDB, rsync-like tools)
// -----------------------------------------------------------------------
func exemplo4DiffEntreDuasArvores() {
	section("Exemplo 4: Diff eficiente entre duas árvores")

	replicaA := [][]byte{
		[]byte("registro-0"),
		[]byte("registro-1"),
		[]byte("registro-2"),
		[]byte("registro-3"),
		[]byte("registro-4"),
		[]byte("registro-5"),
		[]byte("registro-6"),
		[]byte("registro-7"),
	}

	// Simula uma réplica com apenas 2 registros divergentes
	replicaB := make([][]byte, len(replicaA))
	copy(replicaB, replicaA)
	replicaB[3] = []byte("registro-3-DIVERGENTE")
	replicaB[6] = []byte("registro-6-DIVERGENTE")

	treeA, _ := New(replicaA)
	treeB, _ := New(replicaB)

	indicesDivergentes, err := Diff(treeA, treeB)
	if err != nil {
		panic(err)
	}

	fmt.Println("Índices que divergem entre as réplicas:", indicesDivergentes)
	fmt.Println("(Em vez de comparar 8 registros, a árvore permite pular")
	fmt.Println(" subárvores inteiras que já batem, checando só o necessário.)")
}

// -----------------------------------------------------------------------
// Exemplo 5: simular um "bloco" estilo blockchain com várias transações,
// mostrando o fluxo completo: montar bloco -> calcular root -> auditor
// externo verifica uma transação específica sem precisar do bloco inteiro
// -----------------------------------------------------------------------
func exemplo5SimulandoBlocoDeTransacoes() {
	section("Exemplo 5: Simulando um bloco de transações (estilo blockchain)")

	transacoes := [][]byte{
		[]byte("Alice paga 100 para Bob"),
		[]byte("Bob paga 50 para Carol"),
		[]byte("Carol paga 25 para Dave"),
		[]byte("Dave paga 10 para Eve"),
		[]byte("Eve paga 5 para Frank"),
		[]byte("Frank paga 2 para Grace"),
		[]byte("Grace paga 1 para Heidi"),
	}

	bloco, err := New(transacoes)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== Cabeçalho do bloco (simplificado) ===")
	fmt.Println("Merkle Root:", bloco.RootHash())
	fmt.Println("Qtd. de transações:", len(transacoes))

	// Um "nó leve" (light node) da rede quer confirmar que a transação
	// "Dave paga 10 para Eve" está de fato no bloco, sem baixar as 7 transações.
	idxAlvo := 3
	prova, _ := bloco.GenerateProof(idxAlvo)

	fmt.Println("\n=== Light client verificando 1 transação ===")
	fmt.Printf("Transação a verificar: %q\n", transacoes[idxAlvo])
	ok := VerifyProof(transacoes[idxAlvo], prova, bloco.RootHash())
	fmt.Println("Transação confirmada no bloco?", ok)
}

// -----------------------------------------------------------------------
// Exemplo 6: persistir uma Merkle tree em disco e recarregar
// -----------------------------------------------------------------------
func exemplo6Persistencia() {
	section("Exemplo 6: Persistência da árvore em disco")

	dados := [][]byte{
		[]byte("save-slot-1"),
		[]byte("save-slot-2"),
		[]byte("save-slot-3"),
		[]byte("save-slot-4"),
	}

	tree, _ := New(dados)
	fmt.Println("Root antes de salvar:", tree.RootHash())

	path := "/tmp/merkle-snapshot.gob"
	if err := tree.Save(path); err != nil {
		panic(err)
	}
	defer os.Remove(path)

	// Simula reinício do processo: reconstruímos do zero a partir do disco.
	reloaded, err := Load(path)
	if err != nil {
		panic(err)
	}
	fmt.Println("Root após reload:  ", reloaded.RootHash())
	fmt.Println("Iguais?", tree.RootHash() == reloaded.RootHash())

	// A árvore recarregada é totalmente funcional: dá pra gerar provas normalmente.
	proof, _ := reloaded.GenerateProof(1)
	ok := VerifyProof(dados[1], proof, reloaded.RootHash())
	fmt.Println("Prova pós-reload ainda válida?", ok)

	// Versão "somente hash": persiste sem os dados originais (ex.: por
	// política de retenção — você guarda o rastro de auditoria, mas não
	// o conteúdo). A Root e as provas de ESTRUTURA continuam funcionando;
	// o que você perde é a capacidade de re-derivar o hash a partir de um
	// dado novo e comparar (porque não há mais dado nenhum salvo).
	hashes := make([]Hash, len(tree.Leaves))
	for i, leaf := range tree.Leaves {
		hashes[i] = leaf.Hash
	}
	onlyHashesTree := NewFromHashes(hashes)
	fmt.Println("Root reconstruída só com hashes (sem dados):", onlyHashesTree.RootHash())
	fmt.Println("Bate com a original?", onlyHashesTree.RootHash() == tree.RootHash())
}

// -----------------------------------------------------------------------
// Exemplo 7: comparar New() sequencial vs NewParallel() em dataset grande
// -----------------------------------------------------------------------
func exemplo7ParalelizacaoBenchmark() {
	section("Exemplo 7: Paralelização (sequencial vs. paralelo)")

	fmt.Println("CPUs disponíveis:", runtime.NumCPU())
	if runtime.NumCPU() == 1 {
		fmt.Println("(com 1 core só, a versão paralela tende a perder para a sequencial —")
		fmt.Println(" o ganho aparece com múltiplos cores, que é o caso normal em produção.)")
	}

	n := 300_000
	dados := make([][]byte, n)
	for i := 0; i < n; i++ {
		dados[i] = []byte(fmt.Sprintf("evento-de-jogo-%d", i))
	}
	fmt.Printf("Construindo árvore com %d folhas...\n", n)

	start := time.Now()
	treeSeq, _ := New(dados)
	seqDuration := time.Since(start)

	start = time.Now()
	treePar, _ := NewParallel(dados, 0) // 0 = usa runtime.NumCPU()
	parDuration := time.Since(start)

	fmt.Println("Sequencial: ", seqDuration)
	fmt.Println("Paralelo:   ", parDuration)
	fmt.Println("Roots iguais?", treeSeq.RootHash() == treePar.RootHash())
	fmt.Printf("Speedup: %.2fx\n", float64(seqDuration)/float64(parDuration))
	fmt.Println("(Para datasets pequenos, o overhead de coordenar goroutines")
	fmt.Println(" pode tornar a versão paralela mais lenta — teste com o seu volume real.)")
}

// -----------------------------------------------------------------------
// Exemplo 8: uso básico da Merkle Patricia Trie (chave -> valor,
// atualização incremental sem reconstruir tudo)
// -----------------------------------------------------------------------
func exemplo8MPTBasico() {
	section("Exemplo 8: Merkle Patricia Trie — básico")

	store := mpt.NewMemStore()
	root := mpt.EmptyRoot

	root = mpt.Put(store, root, []byte("player:1001"), []byte(`{"hp":100}`))
	rootAposPrimeiraInsercao := root

	root = mpt.Put(store, root, []byte("player:1002"), []byte(`{"hp":80}`))
	root = mpt.Put(store, root, []byte("player:1003"), []byte(`{"hp":95}`))

	v, ok := mpt.Get(store, root, []byte("player:1002"))
	fmt.Printf("player:1002 -> %s (encontrado: %v)\n", v, ok)

	// Atualizar 1 chave é O(profundidade), não O(n): a trie não é
	// reconstruída inteira, só o caminho de "player:1001" muda.
	root = mpt.Put(store, root, []byte("player:1001"), []byte(`{"hp":42}`))
	v, _ = mpt.Get(store, root, []byte("player:1001"))
	fmt.Printf("player:1001 após update -> %s\n", v)

	fmt.Println("\nRoot mudou a cada escrita (esperado):")
	fmt.Println("  root após 1ª inserção:      ", rootAposPrimeiraInsercao)
	fmt.Println("  root após update do hp:     ", root)
	fmt.Printf("Nós distintos armazenados na store: %d (nós antigos não usados por 'root' continuam lá — útil para navegar versões anteriores)\n", store.Len())
}

// -----------------------------------------------------------------------
// Exemplo 9: anti-cheat de drops de itens usando a MPT — o caso de uso
// que motivou tudo isso.
// -----------------------------------------------------------------------
func exemplo9AntiCheatDropDeItens() {
	section("Exemplo 9: Anti-cheat de drop de itens (MPT)")

	registry := anticheat.NewDropRegistry()

	// O servidor autoritativo registra os drops conforme acontecem no jogo.
	espadaLendaria := anticheat.ItemDrop{
		ItemID: "espada-solar", Rarity: "Lendário", Attack: 120, Defense: 0,
		DroppedBy: "boss-dragao-ancestral", MapZone: "vulcao-esquecido",
		DroppedAt: time.Date(2026, 8, 10, 21, 0, 0, 0, time.UTC),
	}
	escudoComum := anticheat.ItemDrop{
		ItemID: "escudo-madeira", Rarity: "Comum", Attack: 0, Defense: 5,
		DroppedBy: "goblin-batedor", MapZone: "floresta-sombria",
		DroppedAt: time.Date(2026, 8, 10, 21, 1, 0, 0, time.UTC),
	}

	root1 := registry.RegisterDrop("drop-9f8a2b", espadaLendaria)
	fmt.Println("Root após drop da espada lendária:", root1)

	root2 := registry.RegisterDrop("drop-1c3e77", escudoComum)
	fmt.Println("Root após drop do escudo comum:   ", root2)

	fmt.Println()
	fmt.Println("--- Cenário 1: cliente honesto pega a espada ---")
	real, ok := registry.VerifyPickup("drop-9f8a2b", espadaLendaria)
	fmt.Printf("Pickup válido? %v (item confirmado: %s, ataque %d)\n", ok, real.ItemID, real.Attack)

	fmt.Println()
	fmt.Println("--- Cenário 2: cliente tenta 'inflar' os stats do item ---")
	espadaAdulterada := espadaLendaria
	espadaAdulterada.Attack = 9999 // cliente editou memória/pacote de rede
	_, ok = registry.VerifyPickup("drop-9f8a2b", espadaAdulterada)
	fmt.Printf("Pickup válido com stats adulterados? %v  <- deve ser rejeitado\n", ok)

	fmt.Println()
	fmt.Println("--- Cenário 3: cliente tenta reivindicar um drop que nunca existiu ---")
	itemFantasma := anticheat.ItemDrop{ItemID: "espada-inexistente", Rarity: "Lendário", Attack: 500}
	_, ok = registry.VerifyPickup("drop-inexistente-000", itemFantasma)
	fmt.Printf("Pickup válido para dropID forjado? %v  <- deve ser rejeitado\n", ok)

	fmt.Println()
	fmt.Println("--- Cenário 4: gerar prova para auditoria/outro servidor ---")
	proof, item, found := registry.ProvePickup("drop-9f8a2b")
	fmt.Printf("Prova gerada para drop-9f8a2b (item=%s), passos: %d\n", item.ItemID, len(proof.Nodes))
	// Um serviço externo (ex.: auditoria central) verifica sem acessar a trie inteira:
	validoExternamente := mpt.VerifyProof(root2, []byte("drop-9f8a2b"), item.Serialize(), proof)
	fmt.Println("Auditor externo confirma a prova?", validoExternamente)

	_ = found
}

// -----------------------------------------------------------------------
// Exemplo 10: persistência do registro (crash recovery) + detecção de
// que o servidor não pode ser "resetado" silenciosamente por um restart.
// -----------------------------------------------------------------------
func exemplo10PersistenciaEDisputaDeAuditoria() {
	section("Exemplo 10: Crash recovery do registro de drops")

	path := "/tmp/drop-registry-snapshot.gob"
	defer os.Remove(path)

	registry := anticheat.NewDropRegistry()
	registry.RegisterDrop("drop-aaa111", anticheat.ItemDrop{
		ItemID: "adaga-sombria", Rarity: "Raro", Attack: 45,
		DroppedBy: "assassino-elite", MapZone: "catacumbas",
	})
	registry.RegisterDrop("drop-bbb222", anticheat.ItemDrop{
		ItemID: "elmo-guerra", Rarity: "Incomum", Defense: 20,
		DroppedBy: "orc-capitao", MapZone: "planicie-quebrada",
	})
	rootAntesDoCrash := registry.RootHash()
	fmt.Println("Root antes do 'crash':", rootAntesDoCrash)

	if err := registry.Save(path); err != nil {
		panic(err)
	}
	fmt.Println("Snapshot salvo em disco.")

	// --- Simula o servidor caindo e subindo de novo ---
	fmt.Println("\n(simulando reinício do servidor de jogo...)")

	recovered, err := anticheat.LoadDropRegistry(path)
	if err != nil {
		panic(err)
	}
	fmt.Println("Root após recuperação:", recovered.RootHash())
	fmt.Println("Histórico de drops preservado?", recovered.RootHash() == rootAntesDoCrash)

	// O registro recuperado continua 100% funcional para novos drops
	// e para validar pickups referentes a drops de ANTES do crash.
	_, ok := recovered.VerifyPickup("drop-aaa111", anticheat.ItemDrop{
		ItemID: "adaga-sombria", Rarity: "Raro", Attack: 45,
		DroppedBy: "assassino-elite", MapZone: "catacumbas",
	})
	fmt.Println("Pickup de item dropado antes do crash ainda verificável?", ok)
}

func section(title string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", len(title)))
	fmt.Println(title)
	fmt.Println(strings.Repeat("=", len(title)))
}
