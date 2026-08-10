package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/pt"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	pt_translations "github.com/go-playground/validator/v10/translations/pt"
)

// ---------------------------------------------------------------------------
// Testes originais (mantidos)
// ---------------------------------------------------------------------------

func TestMarshalUnmarshalStruct(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterEnum("order_status", map[string]map[string]string{
		"approved": {"pt": "aprovado"},
	})

	o := Order{ID: "123", Status: APPROVED, Customer: Person{FirstName: "Ana"}}
	ptb, err := reg.Marshal(o, "pt")
	if err != nil {
		t.Fatalf("Marshal pt error: %v", err)
	}
	enb, err := reg.Marshal(o, "en")
	if err != nil {
		t.Fatalf("Marshal en error: %v", err)
	}

	ptStr := string(ptb)
	enStr := string(enb)

	if !strings.Contains(ptStr, "situacao_pedido") {
		t.Fatalf("pt marshal missing localized key: %s", ptStr)
	}
	if !strings.Contains(enStr, "status") {
		t.Fatalf("en marshal missing status key: %s", enStr)
	}

	var got Order
	body := []byte(`{"id":"123","customer":{"firstName":"Bob"},"status":"approved"}`)
	if err := reg.Unmarshal(body, &got, "en"); err != nil {
		t.Fatalf("Unmarshal en failed: %v", err)
	}
	if got.Status != APPROVED {
		t.Fatalf("expected status %v got %v", APPROVED, got.Status)
	}
}

func TestNestedSlicesAndMaps(t *testing.T) {
	reg := NewRegistry()
	if err := reg.RegisterField("authorization", map[string]string{"pt": "autorizacao"}); err != nil {
		t.Fatalf("RegisterField authorization: %v", err)
	}
	if err := reg.RegisterField("success", map[string]string{"pt": "sucesso"}); err != nil {
		t.Fatalf("RegisterField success: %v", err)
	}
	if err := reg.RegisterField("status", map[string]string{"pt": "situacao"}); err != nil {
		t.Fatalf("RegisterField status: %v", err)
	}
	reg.RegisterFieldEnum("status", "order_status")
	reg.RegisterEnum("order_status", map[string]map[string]string{"approved": {"pt": "aprovado"}})

	in := map[string]interface{}{
		"authorization": map[string]interface{}{"success": true, "status": "approved"},
	}
	localized, err := reg.canonicalToGeneric(in, "pt")
	if err != nil {
		t.Fatalf("canonicalToGeneric error: %v", err)
	}
	m, ok := localized.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map got %T", localized)
	}
	if _, ok := m["autorizacao"]; !ok {
		t.Fatalf("expected localized key autorizacao present")
	}

	canon, err := reg.genericToCanonical(localized, "pt")
	if err != nil {
		t.Fatalf("genericToCanonical error: %v", err)
	}
	cmap, ok := canon.(map[string]interface{})
	if !ok {
		t.Fatalf("expected canonical map after roundtrip")
	}
	inner, ok := cmap["authorization"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected nested map inside authorization")
	}
	if inner["status"] != "approved" {
		t.Fatalf("expected canonical enum value approved, got %v", inner["status"])
	}
}

func TestNestedStructLocaleRoundtrip(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterEnum("order_status", map[string]map[string]string{
		"approved": {"pt": "aprovado"},
	})

	order := Order{ID: "123", Status: APPROVED, Customer: Person{FirstName: "Ana"}}
	ptb, err := reg.Marshal(order, "pt")
	if err != nil {
		t.Fatalf("Marshal pt failed: %v", err)
	}

	var decoded Order
	if err := reg.Unmarshal(ptb, &decoded, "pt"); err != nil {
		t.Fatalf("Unmarshal pt failed: %v", err)
	}

	enb, err := reg.Marshal(decoded, "en")
	if err != nil {
		t.Fatalf("Marshal en failed: %v", err)
	}
	if !strings.Contains(string(enb), `"firstName"`) || !strings.Contains(string(enb), `"status"`) {
		t.Fatalf("expected english keys in roundtrip, got %s", string(enb))
	}
}

func TestNumericConversion(t *testing.T) {
	type NumStruct struct {
		N int `json:"n"`
	}
	reg := NewRegistry()
	var n NumStruct
	if err := reg.Unmarshal([]byte(`{"n":123.0}`), &n, "en"); err != nil {
		t.Fatalf("expected no error converting 123.0 -> int: %v", err)
	}
	if n.N != 123 {
		t.Fatalf("expected 123 got %d", n.N)
	}
	var n2 NumStruct
	if err := reg.Unmarshal([]byte(`{"n":123.4}`), &n2, "en"); err == nil {
		t.Fatalf("expected error converting fractional number to int")
	}
}

func TestPointersAndOmitEmpty(t *testing.T) {
	type PtrStruct struct {
		P *Person `json:"p,omitempty"`
	}
	reg := NewRegistry()
	var p PtrStruct
	if err := reg.Unmarshal([]byte(`{}`), &p, "en"); err != nil {
		t.Fatalf("unmarshal empty failed: %v", err)
	}
	if p.P != nil {
		t.Fatalf("expected nil pointer for missing field")
	}
	if err := reg.Unmarshal([]byte(`{"p":{"firstName":"Bob"}}`), &p, "en"); err != nil {
		t.Fatalf("unmarshal pointer failed: %v", err)
	}
	if p.P == nil || p.P.FirstName != "Bob" {
		t.Fatalf("expected pointer set with FirstName Bob, got %+v", p.P)
	}
}

// ---------------------------------------------------------------------------
// Item 1 — race entre RegisterFieldEnum e Marshal/Unmarshal do campo genérico.
//
// Roda com: go test -race -run TestConcurrentRegisterFieldEnumWithMarshal
// Antes da correção, este teste falhava de forma consistente sob -race,
// porque genericToCanonical/canonicalToGeneric copiavam a referência de
// r.fieldEnum sob RLock e liam o mapa DEPOIS de soltar o lock, enquanto
// RegisterFieldEnum escrevia na mesma instância do mapa sob Lock.
// ---------------------------------------------------------------------------

func TestConcurrentRegisterFieldEnumWithMarshal(t *testing.T) {
	runtime.GOMAXPROCS(8)

	reg := NewRegistry()
	reg.RegisterEnum("order_status", map[string]map[string]string{"approved": {"pt": "aprovado"}})
	if err := reg.RegisterField("status", map[string]string{"pt": "situacao"}); err != nil {
		t.Fatalf("RegisterField: %v", err)
	}
	reg.RegisterFieldEnum("status", "order_status")

	o := Order{
		ID:     "1",
		Status: APPROVED,
		Generic: map[string]json.RawMessage{
			"authorization": json.RawMessage(`{"status":"approved","success":true}`),
		},
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					if _, err := reg.Marshal(o, "pt"); err != nil {
						t.Errorf("Marshal concurrent failed: %v", err)
						return
					}
				}
			}
		}()
	}

	for i := 0; i < 20000; i++ {
		reg.RegisterFieldEnum("dyn_field_x", "order_status")
	}
	close(stop)
	wg.Wait()
}

// ---------------------------------------------------------------------------
// Item 2 — desacoplamento: campo de struct tipada sem tag `enum` explícita
// NÃO deve ser afetado por um RegisterFieldEnum feito para o campo genérico,
// mesmo que os nomes coincidam.
// ---------------------------------------------------------------------------

func TestStructFieldNotAffectedByGenericFieldRegistry(t *testing.T) {
	type Simple struct {
		Status string `json:"status"` // sem tag `enum` de propósito
	}
	reg := NewRegistry()
	reg.RegisterEnum("order_status", map[string]map[string]string{"approved": {"pt": "aprovado"}})
	if err := reg.RegisterField("status", map[string]string{"pt": "situacao"}); err != nil {
		t.Fatalf("RegisterField: %v", err)
	}
	reg.RegisterFieldEnum("status", "order_status")

	s := Simple{Status: "approved"}
	b, err := reg.Marshal(s, "pt")
	if err != nil {
		t.Fatalf("Marshal pt failed: %v", err)
	}
	// Sem tag i18n/enum no campo, deve permanecer "status":"approved" (chave e valor
	// em inglês), mesmo que "status"->"situacao" e o enum estejam registrados
	// globalmente para o campo genérico.
	if !strings.Contains(string(b), `"status":"approved"`) {
		t.Fatalf("campo de struct tipada foi afetado indevidamente pelo registro genérico: %s", string(b))
	}
}

func TestStructFieldWithExplicitEnumTagStillWorks(t *testing.T) {
	type Tagged struct {
		Status string `json:"status" i18n:"pt=situacao" enum:"order_status"`
	}
	reg := NewRegistry()
	reg.RegisterEnum("order_status", map[string]map[string]string{"approved": {"pt": "aprovado"}})

	s := Tagged{Status: "approved"}
	b, err := reg.Marshal(s, "pt")
	if err != nil {
		t.Fatalf("Marshal pt failed: %v", err)
	}
	if !strings.Contains(string(b), `"situacao":"aprovado"`) {
		t.Fatalf("expected localized enum value via explicit tag, got %s", string(b))
	}
}

// ---------------------------------------------------------------------------
// Item 3 — validação estrita de enum.
// ---------------------------------------------------------------------------

func TestStrictEnumRejectsUnknownLocalizedValue(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterEnum("order_status", map[string]map[string]string{"approved": {"pt": "aprovado"}})

	body := []byte(`{"id":"1","cliente":{"primeiro_nome":"Ana"},"situacao_pedido":"aprovadoo"}`) // typo
	var o Order
	err := reg.Unmarshal(body, &o, "pt")
	if err == nil {
		t.Fatalf("esperava erro para valor de enum inválido, mas Unmarshal aceitou silenciosamente")
	}
}

func TestStrictEnumRejectsUnknownCanonicalValue(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterEnum("order_status", map[string]map[string]string{"approved": {"pt": "aprovado"}})

	// Status contém um valor que nunca foi registrado no enum.
	o := Order{ID: "1", Status: OrderStatus("refunded")}
	_, err := reg.Marshal(o, "pt")
	if err == nil {
		t.Fatalf("esperava erro ao serializar valor canônico desconhecido do enum")
	}
}

func TestStrictEnumRejectsWrongJSONType(t *testing.T) {
	reg := NewRegistry()
	reg.RegisterEnum("order_status", map[string]map[string]string{"approved": {"pt": "aprovado"}})

	body := []byte(`{"id":"1","status":123}`) // status deveria ser string
	var o Order
	if err := reg.Unmarshal(body, &o, "en"); err == nil {
		t.Fatalf("esperava erro para tipo de valor incompatível com enum")
	}
}

func TestEnumIdentityFallbackForBaseLocale(t *testing.T) {
	// "en" nunca é explicitamente registrado (canônico == inglês por convenção),
	// e mesmo assim deve funcionar via identidade, sem erro.
	reg := NewRegistry()
	reg.RegisterEnum("order_status", map[string]map[string]string{"approved": {"pt": "aprovado"}})

	o := Order{ID: "1", Status: APPROVED}
	b, err := reg.Marshal(o, "en")
	if err != nil {
		t.Fatalf("locale base (en) sem tradução explícita deveria funcionar por identidade: %v", err)
	}
	if !strings.Contains(string(b), `"status":"approved"`) {
		t.Fatalf("esperado status=approved via identidade, got %s", string(b))
	}
}

// ---------------------------------------------------------------------------
// Item 4 — índice reverso / colisão de nomes no dicionário genérico.
// ---------------------------------------------------------------------------

func TestRegisterFieldReverseLookupIsFast(t *testing.T) {
	reg := NewRegistry()
	for i := 0; i < 500; i++ {
		canon := fmt.Sprintf("field_%d", i)
		if err := reg.RegisterField(canon, map[string]string{"pt": fmt.Sprintf("campo_%d", i)}); err != nil {
			t.Fatalf("RegisterField %s: %v", canon, err)
		}
	}
	// canonicalFieldName deve resolver via índice reverso, não varredura linear.
	got := reg.canonicalFieldName("campo_499", "pt")
	if got != "field_499" {
		t.Fatalf("expected field_499, got %s", got)
	}
}

func TestRegisterFieldDetectsCollision(t *testing.T) {
	reg := NewRegistry()
	if err := reg.RegisterField("status", map[string]string{"pt": "situacao"}); err != nil {
		t.Fatalf("first RegisterField should succeed: %v", err)
	}
	err := reg.RegisterField("state", map[string]string{"pt": "situacao"}) // mesmo nome pt, canônico diferente
	if err == nil {
		t.Fatalf("esperava erro de colisão ao registrar dois campos com o mesmo nome localizado")
	}
}

// ---------------------------------------------------------------------------
// Item 5 — demais correções: campos embedded/anônimos promovidos como no
// encoding/json padrão.
// ---------------------------------------------------------------------------

type Audited struct {
	CreatedBy string `json:"createdBy" i18n:"pt=criado_por"`
}

type AuditedOrder struct {
	Audited        // embedded, sem tag json explícita -> deve ser promovido
	ID      string `json:"id"`
}

func TestEmbeddedStructPromotion(t *testing.T) {
	reg := NewRegistry()

	o := AuditedOrder{ID: "1", Audited: Audited{CreatedBy: "Ana"}}
	b, err := reg.Marshal(o, "pt")
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	// campo do embedded deve aparecer no nível raiz, traduzido, sem aninhar
	// sob uma chave "Audited".
	if !strings.Contains(string(b), `"criado_por":"Ana"`) {
		t.Fatalf("esperado campo promovido do embedded, got %s", string(b))
	}
	if strings.Contains(string(b), `"Audited"`) {
		t.Fatalf("campo embedded não deveria aparecer aninhado, got %s", string(b))
	}

	var decoded AuditedOrder
	body := []byte(`{"id":"1","criado_por":"Bob"}`)
	if err := reg.Unmarshal(body, &decoded, "pt"); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if decoded.CreatedBy != "Bob" {
		t.Fatalf("esperado CreatedBy=Bob, got %q", decoded.CreatedBy)
	}
}

// ---------------------------------------------------------------------------
// Tradução de erros do go-playground/validator, usando a infraestrutura de
// tradução nativa dele (ut.UniversalTranslator + translations/{locale}).
// ---------------------------------------------------------------------------

// setupValidators monta um *validator.Validate + ut.Translator por locale,
// como uma aplicação real faria: cada validator usa reg.I18nTagNameFunc(locale)
// pra reportar nomes de campo na tag i18n, e cada translator usa o pacote
// oficial de traduções daquele idioma.
func setupValidators(t *testing.T, reg *Registry) (vEn, vPt *validator.Validate) {
	t.Helper()

	enLoc := en.New()
	ptLoc := pt.New()
	uni := ut.New(enLoc, enLoc, ptLoc) // fallback do universal-translator: en

	vEn = validator.New()
	vEn.RegisterTagNameFunc(reg.I18nTagNameFunc("en"))
	transEn, _ := uni.GetTranslator("en")
	if err := en_translations.RegisterDefaultTranslations(vEn, transEn); err != nil {
		t.Fatalf("registrar traduções en: %v", err)
	}
	reg.RegisterValidatorTranslator("en", transEn)

	vPt = validator.New()
	vPt.RegisterTagNameFunc(reg.I18nTagNameFunc("pt"))
	transPt, _ := uni.GetTranslator("pt")
	if err := pt_translations.RegisterDefaultTranslations(vPt, transPt); err != nil {
		t.Fatalf("registrar traduções pt: %v", err)
	}
	reg.RegisterValidatorTranslator("pt", transPt)

	return vEn, vPt
}

func TestTranslateValidationErrors_TopLevelField(t *testing.T) {
	reg := NewRegistry()
	vEn, vPt := setupValidators(t, reg)

	o := &Order{}

	errPt := vPt.Struct(o)
	verrsPt, ok := errPt.(validator.ValidationErrors)
	if !ok || len(verrsPt) == 0 {
		t.Fatalf("esperava validator.ValidationErrors não vazio, got %v", errPt)
	}
	pt, err := reg.TranslateValidationErrors(verrsPt, reflect.TypeOf(*o), "pt")
	if err != nil {
		t.Fatalf("TranslateValidationErrors pt: %v", err)
	}
	if len(pt) != 1 || pt[0].Field != "situacao_pedido" {
		t.Fatalf("esperava campo pt=situacao_pedido, got %+v", pt)
	}
	if !strings.Contains(pt[0].Message, "situacao_pedido") {
		t.Fatalf("mensagem pt deveria conter o campo traduzido, got %q", pt[0].Message)
	}

	errEn := vEn.Struct(o)
	verrsEn := errEn.(validator.ValidationErrors)
	en, err := reg.TranslateValidationErrors(verrsEn, reflect.TypeOf(*o), "en")
	if err != nil {
		t.Fatalf("TranslateValidationErrors en: %v", err)
	}
	if len(en) != 1 || en[0].Field != "status" {
		t.Fatalf("esperava campo en=status (fallback pro json tag), got %+v", en)
	}
	if !strings.Contains(en[0].Message, "status") {
		t.Fatalf("mensagem en deveria conter o campo traduzido, got %q", en[0].Message)
	}
}

type validationItem struct {
	Name string `json:"name" i18n:"pt=nome" validate:"required"`
}

type validationOrderWithItems struct {
	ID    string           `json:"id"`
	Items []validationItem `json:"items" i18n:"pt=items" validate:"dive"`
}

func TestTranslateValidationErrors_NestedSlice(t *testing.T) {
	reg := NewRegistry()
	vEn, vPt := setupValidators(t, reg)

	o := validationOrderWithItems{Items: []validationItem{{}, {Name: "ok"}}}
	errPt := vPt.Struct(o)
	verrs, ok := errPt.(validator.ValidationErrors)
	if !ok || len(verrs) != 1 {
		t.Fatalf("esperava exatamente 1 erro de validação, got %v", errPt)
	}

	pt, err := reg.TranslateValidationErrors(verrs, reflect.TypeOf(o), "pt")
	if err != nil {
		t.Fatalf("TranslateValidationErrors: %v", err)
	}
	if len(pt) != 1 || pt[0].Field != "items[0].nome" {
		t.Fatalf("esperava items[0].nome, got %+v", pt)
	}

	errEn := vEn.Struct(o)
	verrs, ok = errEn.(validator.ValidationErrors)
	if !ok || len(verrs) != 1 {
		t.Fatalf("esperava exatamente 1 erro de validação, got %v", errEn)
	}

	en, err := reg.TranslateValidationErrors(verrs, reflect.TypeOf(o), "en")
	if err != nil {
		t.Fatalf("TranslateValidationErrors: %v", err)
	}
	if len(en) != 1 || en[0].Field != "itens[0].name" {
		t.Fatalf("esperava itens[0].name, got %+v", pt)
	}
}

type ValidationEmbeddedBase struct {
	CreatedBy string `json:"createdBy" i18n:"pt=criado_por" validate:"required"`
}

type validationEmbeddedOrder struct {
	ValidationEmbeddedBase
	ID string `json:"id"`
}

func TestTranslateValidationErrors_EmbeddedField(t *testing.T) {
	reg := NewRegistry()
	_, vPt := setupValidators(t, reg)

	o := validationEmbeddedOrder{}
	errPt := vPt.Struct(o)
	verrs, ok := errPt.(validator.ValidationErrors)
	if !ok || len(verrs) != 1 {
		t.Fatalf("esperava 1 erro, got %v", errPt)
	}

	pt, err := reg.TranslateValidationErrors(verrs, reflect.TypeOf(o), "pt")
	if err != nil {
		t.Fatalf("TranslateValidationErrors: %v", err)
	}
	if len(pt) != 1 || pt[0].Field != "criado_por" {
		t.Fatalf("esperava campo criado_por (embedded, sem prefixo de tipo), got %+v", pt)
	}
}

func TestTranslateValidationErrors_FallbackToCanonicalLocale(t *testing.T) {
	reg := NewRegistry()
	vEn, _ := setupValidators(t, reg) // "es" nunca é registrado de propósito

	o := &Order{}
	errEn := vEn.Struct(o)
	verrs := errEn.(validator.ValidationErrors)

	// pedimos "es", que não tem tradutor registrado -> cai pro fallback (en)
	es, err := reg.TranslateValidationErrors(verrs, reflect.TypeOf(*o), "es")
	if err != nil {
		t.Fatalf("esperava fallback bem-sucedido pro locale canônico, got err: %v", err)
	}
	if len(es) != 1 || es[0].Field != "status" {
		t.Fatalf("esperava fallback pro campo en=status, got %+v", es)
	}
}

func TestTranslateValidationErrors_NoTranslatorRegisteredAtAll(t *testing.T) {
	reg := NewRegistry() // nenhum RegisterValidatorTranslator chamado

	fakeErrs := validator.ValidationErrors{}
	if _, err := reg.TranslateValidationErrors(fakeErrs, reflect.TypeOf(Order{}), "pt"); err == nil {
		t.Fatalf("esperava erro quando nenhum translator foi registrado, nem pro locale nem pro fallback")
	}
}
