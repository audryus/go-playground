package main

import (
	"fmt"
	"strings"
)

func main() {
	exemplo1RootHashBasico()
	exemplo2ProvaDeInclusao()
	exemplo3DeteccaoDeAdulteracao()
	exemplo4DiffEntreDuasArvores()
	exemplo5SimulandoBlocoDeTransacoes()
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

func section(title string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", len(title)))
	fmt.Println(title)
	fmt.Println(strings.Repeat("=", len(title)))
}
