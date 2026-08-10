package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

type OrderStatus string

const APPROVED OrderStatus = "approved"

type Order struct {
	ID       string                     `json:"id"`
	Customer Person                     `json:"customer" i18n:"pt=cliente"`
	Status   OrderStatus                `json:"status" i18n:"pt=situacao_pedido" enum:"order_status" validate:"required"`
	Generic  map[string]json.RawMessage `json:"generic" i18n:"pt=generico"`
}

type Person struct {
	FirstName string `json:"firstName" i18n:"pt=primeiro_nome"`
}

func main() {
	reg := NewRegistry()
	reg.RegisterEnum("order_status", map[string]map[string]string{
		"approved": {"pt": "aprovado"},
		"pending":  {"pt": "pendente"},
	})

	o := Order{ID: "123", Status: APPROVED, Customer: Person{FirstName: "approved"}}
	pt, err := reg.Marshal(o, "pt")
	if err != nil {
		panic(err)
	}
	en, err := reg.Marshal(o, "en")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(pt))
	fmt.Println(string(en))

	var order Order
	bode := []byte(`{
  "id": "123",
  "customer": { "firstName": "Audryus" },
  "status": "approved"
}`)
	if err := reg.Unmarshal(bode, &order, "en"); err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", order)

	pt, err = reg.Marshal(order, "pt")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(pt))
}

// ---------------------------------------------------------------------------
// Enum registry (canônico <-> localizado), com validação estrita.
// ---------------------------------------------------------------------------

type enumTable struct {
	toLocale map[string]map[string]string // canônico -> locale -> localizado
	toCanon  map[string]map[string]string // locale -> localizado -> canônico
}

// ---------------------------------------------------------------------------
// Cache de metadados de struct (reflect.Type -> []fieldMeta), com suporte a
// múltiplos nomes de locale por campo, enum, e promoção de campos embedded.
// ---------------------------------------------------------------------------

type fieldMeta struct {
	Index        int
	LocaleNames  map[string]string // locale -> chave (da tag i18n)
	DefaultName  string            // fallback: tag json, ou nome do campo Go
	Hidden       bool              // json:"-"
	EnumName     string            // "" se não for enum (vem só da tag `enum`)
	Kind         reflect.Kind
	EmbedPromote bool // campo anônimo (embedded) sem nome json explícito: promove os campos filhos pro nível do pai
}

var structCache sync.Map // reflect.Type -> []fieldMeta

func getFieldMeta(typ reflect.Type) []fieldMeta {
	if cached, ok := structCache.Load(typ); ok {
		return cached.([]fieldMeta)
	}

	var fields []fieldMeta
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.PkgPath != "" {
			continue // não exportado
		}

		fm := fieldMeta{
			Index:       i,
			LocaleNames: map[string]string{},
			EnumName:    f.Tag.Get("enum"),
			Kind:        f.Type.Kind(),
		}

		if tag, ok := f.Tag.Lookup("i18n"); ok {
			for _, part := range strings.Split(tag, ",") {
				kv := strings.SplitN(part, "=", 2)
				if len(kv) == 2 {
					fm.LocaleNames[kv[0]] = kv[1]
				}
			}
		}

		jsonTag := f.Tag.Get("json")
		explicitName := strings.Split(jsonTag, ",")[0]

		switch explicitName {
		case "-":
			fm.Hidden = true
		case "":
			fm.DefaultName = f.Name
			underlying := f.Type
			if underlying.Kind() == reflect.Ptr {
				underlying = underlying.Elem()
			}
			// campo embedded (anônimo) sem nome json explícito: promove como o encoding/json faz
			if f.Anonymous && underlying.Kind() == reflect.Struct {
				fm.EmbedPromote = true
			}
		default:
			fm.DefaultName = explicitName
		}

		fields = append(fields, fm)
	}

	structCache.Store(typ, fields)
	return fields
}

// resolveFieldKey resolve o nome de chave JSON de um campo de struct conhecida
// (schema fixo) para um locale. Usa SÓ a tag `i18n`/`json` do próprio campo —
// propositalmente NÃO cai pro dicionário global de campos genéricos
// (RegisterField/RegisterFieldEnum), que é reservado ao campo dinâmico
// (map[string]json.RawMessage). Isso evita que registrar uma tradução para
// o conteúdo genérico de uma capability afete, por coincidência de nome,
// campos de contratos tipados que nada têm a ver com ela.
func resolveFieldKey(fm fieldMeta, locale string) (string, bool) {
	if fm.Hidden {
		return "", false
	}
	if name, ok := fm.LocaleNames[locale]; ok {
		return name, true
	}
	return fm.DefaultName, true
}

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

type Registry struct {
	mu sync.RWMutex

	enums map[string]*enumTable

	// dicionário do campo GENÉRICO (map[string]json.RawMessage) apenas.
	// Não tem relação com a tag i18n das structs tipadas (ver resolveFieldKey).
	fieldNames               map[string]map[string]string // canônico -> locale -> localizado
	fieldNamesReverse        map[string]map[string]string // locale -> localizado -> canônico (índice reverso, O(1))
	fieldEnum                map[string]string            // campo canônico (genérico) -> nome do enum associado
	validatorTranslators     map[string]ut.Translator     // locale -> ut.Translator já configurado (RegisterDefaultTranslations)
	validationFallbackLocale string                       // locale usado quando o pedido não tem tradutor (padrão: "en")
}

func NewRegistry() *Registry {
	return &Registry{enums: map[string]*enumTable{}}
}

func (r *Registry) RegisterEnum(name string, values map[string]map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t := &enumTable{toLocale: values, toCanon: map[string]map[string]string{}}
	for canon, locales := range values {
		for locale, localized := range locales {
			if t.toCanon[locale] == nil {
				t.toCanon[locale] = map[string]string{}
			}
			t.toCanon[locale][localized] = canon
		}
	}
	r.enums[name] = t
}

// RegisterField define o nome canônico de um campo do payload GENÉRICO e suas
// traduções por locale. Mantém um índice reverso (locale+localizado -> canônico)
// para lookup O(1), reconstruído a cada chamada. Detecta colisão: se duas
// entradas canônicas diferentes acabarem mapeando pro mesmo nome localizado no
// mesmo locale, a chamada falha em vez de silenciosamente produzir um
// resultado não-determinístico.
func (r *Registry) RegisterField(canonical string, locales map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	candidate := make(map[string]map[string]string, len(r.fieldNames)+1)
	for c, ls := range r.fieldNames {
		candidate[c] = ls
	}
	candidate[canonical] = locales

	reverse := map[string]map[string]string{}
	for canon, ls := range candidate {
		for locale, localized := range ls {
			if reverse[locale] == nil {
				reverse[locale] = map[string]string{}
			}
			if existing, ok := reverse[locale][localized]; ok && existing != canon {
				return fmt.Errorf("i18n: colisão de nome de campo: %q e %q usam o mesmo nome %q no locale %q", existing, canon, localized, locale)
			}
			reverse[locale][localized] = canon
		}
	}

	r.fieldNames = candidate
	r.fieldNamesReverse = reverse
	return nil
}

// RegisterFieldEnum associa um campo canônico do payload GENÉRICO a um enum já
// registrado. Só afeta a tradução dentro de map[string]json.RawMessage — não
// afeta campos de structs tipadas (essas só usam a tag `enum` explícita).
func (r *Registry) RegisterFieldEnum(canonicalField, enumName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fieldEnum == nil {
		r.fieldEnum = map[string]string{}
	}
	r.fieldEnum[canonicalField] = enumName
}

func (r *Registry) canonicalFieldName(localized, locale string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if rev, ok := r.fieldNamesReverse[locale]; ok {
		if canon, ok := rev[localized]; ok {
			return canon
		}
	}
	return localized
}

func (r *Registry) localizedFieldName(canonical, locale string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if locales, ok := r.fieldNames[canonical]; ok {
		if name, ok := locales[locale]; ok {
			return name
		}
	}
	return canonical
}

// lookupGenericFieldEnum é a ÚNICA forma de ler r.fieldEnum. Nunca copie o mapa
// e leia depois de soltar o lock — isso foi a causa da race original entre
// RegisterFieldEnum (escreve) e Marshal/Unmarshal do campo genérico (lia sem lock).
func (r *Registry) lookupGenericFieldEnum(canonicalField string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.fieldEnum == nil {
		return "", false
	}
	name, ok := r.fieldEnum[canonicalField]
	return name, ok
}

// toLocale traduz um valor canônico de enum para o locale pedido.
// Regras (validação estrita):
//   - enum não registrado -> erro
//   - valor canônico desconhecido pro enum -> erro (dado inválido na aplicação)
//   - locale sem tradução explícita registrada -> devolve o valor canônico
//     sem erro (identidade). Isso é intencional: o padrão adotado é o valor
//     canônico coincidir com o idioma "base" (ex: inglês), então não é
//     necessário registrar entradas para ele.
func (r *Registry) toLocale(enum, canonical, locale string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.enums[enum]
	if !ok {
		return "", fmt.Errorf("enum %q não registrado", enum)
	}
	locales, ok := t.toLocale[canonical]
	if !ok {
		return "", fmt.Errorf("valor %q não é um valor canônico válido do enum %q", canonical, enum)
	}
	if v, ok := locales[locale]; ok {
		return v, nil
	}
	return canonical, nil
}

// toCanonical traduz um valor localizado para o canônico.
// Regras (validação estrita):
//   - enum não registrado -> erro
//   - valor não encontrado nem como tradução localizada nem como valor
//     canônico usado diretamente (identidade) -> erro
func (r *Registry) toCanonical(enum, localized, locale string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.enums[enum]
	if !ok {
		return "", fmt.Errorf("enum %q não registrado", enum)
	}
	if m, ok := t.toCanon[locale]; ok {
		if canon, ok := m[localized]; ok {
			return canon, nil
		}
	}
	if _, ok := t.toLocale[localized]; ok {
		return localized, nil // já é um valor canônico válido (identidade)
	}
	return "", fmt.Errorf("valor %q inválido para o enum %q no locale %q", localized, enum, locale)
}

// genericToCanonical traduz recursivamente um valor decodificado do payload
// GENÉRICO (localizado -> canônico), incluindo tradução de valores de enum
// quando o campo canônico correspondente estiver associado via RegisterFieldEnum.
func (r *Registry) genericToCanonical(v interface{}, locale string) (interface{}, error) {
	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, vv := range val {
			canonKey := r.canonicalFieldName(k, locale)
			translated, err := r.genericToCanonical(vv, locale)
			if err != nil {
				return nil, fmt.Errorf("campo %q: %w", canonKey, err)
			}
			if enumName, hasEnum := r.lookupGenericFieldEnum(canonKey); hasEnum {
				if s, ok := translated.(string); ok {
					canon, err := r.toCanonical(enumName, s, locale)
					if err != nil {
						return nil, fmt.Errorf("campo %q: %w", canonKey, err)
					}
					translated = canon
				}
			}
			out[canonKey] = translated
		}
		return out, nil
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, item := range val {
			translated, err := r.genericToCanonical(item, locale)
			if err != nil {
				return nil, err
			}
			out[i] = translated
		}
		return out, nil
	default:
		return val, nil
	}
}

// canonicalToGeneric é o espelho de genericToCanonical (canônico -> localizado).
func (r *Registry) canonicalToGeneric(v interface{}, locale string) (interface{}, error) {
	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, vv := range val {
			localizedVal, err := r.canonicalToGeneric(vv, locale)
			if err != nil {
				return nil, fmt.Errorf("campo %q: %w", k, err)
			}
			if enumName, hasEnum := r.lookupGenericFieldEnum(k); hasEnum {
				if s, ok := localizedVal.(string); ok {
					loc, err := r.toLocale(enumName, s, locale)
					if err != nil {
						return nil, fmt.Errorf("campo %q: %w", k, err)
					}
					localizedVal = loc
				}
			}
			out[r.localizedFieldName(k, locale)] = localizedVal
		}
		return out, nil
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, item := range val {
			translated, err := r.canonicalToGeneric(item, locale)
			if err != nil {
				return nil, err
			}
			out[i] = translated
		}
		return out, nil
	default:
		return val, nil
	}
}

// ---------------------------------------------------------------------------
// Conversão de tipos simples (usada no Unmarshal)
// ---------------------------------------------------------------------------

func safeEnumString(fv reflect.Value) string {
	if fv.Kind() == reflect.Ptr {
		if fv.IsNil() {
			return ""
		}
		return safeEnumString(fv.Elem())
	}
	if fv.Kind() == reflect.String {
		return fv.String()
	}
	if fv.CanInterface() {
		return fmt.Sprintf("%v", fv.Interface())
	}
	return fmt.Sprintf("%v", fv)
}

func convertSimple(rawVal interface{}, targetType reflect.Type) (reflect.Value, error) {
	if rawVal == nil {
		return reflect.Zero(targetType), nil
	}

	if targetType.Kind() == reflect.Ptr {
		converted, err := convertSimple(rawVal, targetType.Elem())
		if err != nil {
			return reflect.Zero(targetType), err
		}
		ptr := reflect.New(targetType.Elem())
		ptr.Elem().Set(converted)
		return ptr, nil
	}

	switch targetType.Kind() {
	case reflect.String:
		switch v := rawVal.(type) {
		case string:
			return reflect.ValueOf(v).Convert(targetType), nil
		case json.Number:
			return reflect.ValueOf(v.String()).Convert(targetType), nil
		default:
			return reflect.ValueOf(fmt.Sprint(v)).Convert(targetType), nil
		}
	case reflect.Bool:
		switch v := rawVal.(type) {
		case bool:
			return reflect.ValueOf(v).Convert(targetType), nil
		case string:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return reflect.Zero(targetType), err
			}
			return reflect.ValueOf(b).Convert(targetType), nil
		default:
			return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to bool", rawVal)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var iv int64
		switch v := rawVal.(type) {
		case float64:
			if math.Mod(v, 1) != 0 {
				return reflect.Zero(targetType), fmt.Errorf("cannot convert non-integer numeric %v to int", v)
			}
			iv = int64(v)
		case json.Number:
			s := v.String()
			if strings.ContainsAny(s, ".eE") {
				f, err := v.Float64()
				if err != nil {
					return reflect.Zero(targetType), err
				}
				if math.Mod(f, 1) != 0 {
					return reflect.Zero(targetType), fmt.Errorf("cannot convert non-integer numeric %v to int", f)
				}
				iv = int64(f)
			} else {
				i, err := v.Int64()
				if err != nil {
					return reflect.Zero(targetType), err
				}
				iv = i
			}
		case string:
			parsed, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				if f, err2 := strconv.ParseFloat(v, 64); err2 == nil && math.Mod(f, 1) == 0 {
					iv = int64(f)
				} else {
					return reflect.Zero(targetType), err
				}
			} else {
				iv = parsed
			}
		default:
			rv := reflect.ValueOf(rawVal)
			if rv.Type().ConvertibleTo(targetType) {
				return rv.Convert(targetType), nil
			}
			return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to %s", rawVal, targetType)
		}
		out := reflect.New(targetType).Elem()
		out.SetInt(iv)
		return out, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var uv uint64
		switch v := rawVal.(type) {
		case float64:
			if math.Mod(v, 1) != 0 {
				return reflect.Zero(targetType), fmt.Errorf("cannot convert non-integer numeric %v to uint", v)
			}
			uv = uint64(v)
		case json.Number:
			s := v.String()
			if strings.ContainsAny(s, ".eE") {
				f, err := v.Float64()
				if err != nil {
					return reflect.Zero(targetType), err
				}
				if math.Mod(f, 1) != 0 || f < 0 {
					return reflect.Zero(targetType), fmt.Errorf("cannot convert non-integer numeric %v to uint", f)
				}
				uv = uint64(f)
			} else {
				parsed, err := v.Int64()
				if err != nil || parsed < 0 {
					return reflect.Zero(targetType), fmt.Errorf("cannot convert %v to uint", v)
				}
				uv = uint64(parsed)
			}
		case string:
			parsed, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				if f, err2 := strconv.ParseFloat(v, 64); err2 == nil && math.Mod(f, 1) == 0 && f >= 0 {
					uv = uint64(f)
				} else {
					return reflect.Zero(targetType), err
				}
			} else {
				uv = parsed
			}
		default:
			rv := reflect.ValueOf(rawVal)
			if rv.Type().ConvertibleTo(targetType) {
				return rv.Convert(targetType), nil
			}
			return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to %s", rawVal, targetType)
		}
		out := reflect.New(targetType).Elem()
		out.SetUint(uv)
		return out, nil
	case reflect.Float32, reflect.Float64:
		var fv float64
		switch v := rawVal.(type) {
		case float64:
			fv = v
		case json.Number:
			parsed, err := v.Float64()
			if err != nil {
				return reflect.Zero(targetType), err
			}
			fv = parsed
		case string:
			parsed, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return reflect.Zero(targetType), err
			}
			fv = parsed
		default:
			rv := reflect.ValueOf(rawVal)
			if rv.Type().ConvertibleTo(targetType) {
				return rv.Convert(targetType), nil
			}
			return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to %s", rawVal, targetType)
		}
		out := reflect.New(targetType).Elem()
		out.SetFloat(fv)
		return out, nil
	default:
		rv := reflect.ValueOf(rawVal)
		if rv.Type().ConvertibleTo(targetType) {
			return rv.Convert(targetType), nil
		}
		return reflect.Zero(targetType), fmt.Errorf("unsupported conversion %T -> %s", rawVal, targetType)
	}
}

// ---------------------------------------------------------------------------
// Marshal / toMap
// ---------------------------------------------------------------------------

var rawMessageType = reflect.TypeOf(json.RawMessage{})

func isGenericMap(t reflect.Type) bool {
	return t.Kind() == reflect.Map &&
		t.Key().Kind() == reflect.String &&
		t.Elem() == rawMessageType
}

func (r *Registry) Marshal(v interface{}, locale string) ([]byte, error) {
	if locale == "" {
		return nil, fmt.Errorf("locale não pode ser vazio")
	}
	m, err := r.toMap(reflect.ValueOf(v), locale)
	if err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

func (r *Registry) toMap(v reflect.Value, locale string) (interface{}, error) {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
	}

	if v.IsValid() && isGenericMap(v.Type()) {
		out := map[string]interface{}{}
		iter := v.MapRange()
		for iter.Next() {
			canonKey := iter.Key().String()
			raw := iter.Value().Interface().(json.RawMessage)

			var parsed interface{}
			if err := json.Unmarshal(raw, &parsed); err != nil {
				return nil, err
			}
			localizedVal, err := r.canonicalToGeneric(parsed, locale)
			if err != nil {
				return nil, fmt.Errorf("campo genérico %q: %w", canonKey, err)
			}
			out[r.localizedFieldName(canonKey, locale)] = localizedVal
		}
		return out, nil
	}

	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		outSlice := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			elem, err := r.toMap(v.Index(i), locale)
			if err != nil {
				return nil, err
			}
			outSlice[i] = elem
		}
		return outSlice, nil
	case reflect.Map:
		outMap := map[string]interface{}{}
		for _, k := range v.MapKeys() {
			if k.Kind() != reflect.String {
				continue // somente chaves string são suportadas para JSON
			}
			val := v.MapIndex(k)
			mapped, err := r.toMap(val, locale)
			if err != nil {
				return nil, err
			}
			outMap[k.String()] = mapped
		}
		return outMap, nil
	}

	if v.Kind() != reflect.Struct {
		return v.Interface(), nil
	}

	out := map[string]interface{}{}
	for _, fm := range getFieldMeta(v.Type()) {
		if fm.Hidden {
			continue
		}
		fv := v.Field(fm.Index)

		if fm.EmbedPromote {
			nested, err := r.toMap(fv, locale)
			if err != nil {
				return nil, err
			}
			if nestedMap, ok := nested.(map[string]interface{}); ok {
				for k, vv := range nestedMap {
					out[k] = vv
				}
			}
			continue
		}

		key, ok := resolveFieldKey(fm, locale)
		if !ok {
			continue
		}

		if fm.EnumName != "" {
			loc, err := r.toLocale(fm.EnumName, safeEnumString(fv), locale)
			if err != nil {
				return nil, fmt.Errorf("campo %q: %w", key, err)
			}
			out[key] = loc
			continue
		}

		nested, err := r.toMap(fv, locale)
		if err != nil {
			return nil, err
		}
		out[key] = nested
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Unmarshal / fromMap
// ---------------------------------------------------------------------------

func (r *Registry) Unmarshal(data []byte, v interface{}, locale string) error {
	if locale == "" {
		return fmt.Errorf("locale não pode ser vazio")
	}
	var raw map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return err
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("v deve ser ponteiro para struct")
	}
	return r.fromMap(raw, rv.Elem(), locale)
}

// ---------------------------------------------------------------------------
// Tradução de erros do go-playground/validator
//
// Ao contrário da primeira versão, isso NÃO mantém um catálogo de mensagens
// próprio. O validator já resolve isso via go-playground/universal-translator
// + os pacotes github.com/go-playground/validator/v10/translations/{locale},
// que cobrem dezenas de idiomas com mensagens revisadas — reinventar isso
// seria pior manutenção, não melhor.
//
// O que o Registry faz é só a ponte:
//  1. I18nTagNameFunc(locale) — plugado via v.RegisterTagNameFunc no validator
//     de cada locale — faz o validator usar a MESMA tag `i18n` do JSON como
//     nome de campo nas mensagens, em vez do nome do campo Go.
//  2. RegisterValidatorTranslator(locale, trans) guarda o ut.Translator já
//     configurado (com RegisterDefaultTranslations do pacote daquele idioma)
//     pra cada locale.
//  3. TranslateValidationErrors(errs, locale) usa fe.Translate(trans), com
//     fallback pro locale "canônico" (padrão: inglês) se o locale pedido não
//     tiver tradutor registrado.
//
// Quem monta o *validator.Validate e o ut.Translator por locale é o
// aplicativo (usando os pacotes de tradução que ele realmente precisa) — o
// Registry não importa translations/pt, translations/es etc. pra não forçar
// toda aplicação a carregar idiomas que não usa.
// ---------------------------------------------------------------------------

// ValidationError é a versão localizada de um validator.FieldError.
type ValidationError struct {
	Field   string // caminho do campo já no idioma pedido, ex: "situacao_pedido" ou "cliente.primeiro_nome"
	Tag     string // tag de validação que falhou, ex: "required", "email"
	Message string // mensagem humana já localizada (gerada pelo validator/translations)
}

func (v ValidationError) Error() string {
	return v.Message
}

// I18nTagNameFunc devolve uma função pronta pra plugar via
// v.RegisterTagNameFunc(reg.I18nTagNameFunc("pt")) num *validator.Validate
// específico de um locale. Ela faz o validator reportar (e traduzir) usando
// o nome definido na tag `i18n` daquele locale — a mesma tag já usada pelo
// Marshal/Unmarshal — em vez do nome do campo Go. Fallback: tag `json`, e por
// fim o nome do campo Go.
//
// É uma função pura (não depende de estado do Registry) — por isso não
// precisa de um *Registry específico pra ser chamada, mas fica como método
// por conveniência/descoberta (reg.I18nTagNameFunc(...)).
func (r *Registry) I18nTagNameFunc(locale string) validator.TagNameFunc {
	return func(fld reflect.StructField) string {
		if tag, ok := fld.Tag.Lookup("i18n"); ok {
			for _, part := range strings.Split(tag, ",") {
				kv := strings.SplitN(part, "=", 2)
				if len(kv) == 2 && kv[0] == locale {
					return kv[1]
				}
			}
		}
		name := strings.Split(fld.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			return fld.Name
		}
		return name
	}
}

// RegisterValidatorTranslator associa um ut.Translator já configurado
// (RegisterDefaultTranslations já chamado) a um locale. Chame uma vez por
// idioma suportado, no boot da aplicação.
func (r *Registry) RegisterValidatorTranslator(locale string, trans ut.Translator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.validatorTranslators == nil {
		r.validatorTranslators = map[string]ut.Translator{}
	}
	r.validatorTranslators[locale] = trans
}

// SetValidationFallbackLocale define o locale canônico usado quando
// TranslateValidationErrors recebe um locale sem tradutor registrado.
// Se nunca chamado, o padrão é "en".
func (r *Registry) SetValidationFallbackLocale(locale string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.validationFallbackLocale = locale
}

func (r *Registry) validatorTranslatorFor(locale string) (ut.Translator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if t, ok := r.validatorTranslators[locale]; ok {
		return t, true
	}
	fallback := r.validationFallbackLocale
	if fallback == "" {
		fallback = "en"
	}
	t, ok := r.validatorTranslators[fallback]
	return t, ok
}

// stripEmbeddedSegments remove do namespace do validator os segmentos que
// correspondem a campos embedded promovidos (EmbedPromote) — o validator não
// sabe que esses campos são "achatados" no JSON, então inclui o nome do tipo
// embedded como um segmento próprio (ex: "Order.Base.CreatedBy"). Aqui só
// removemos esse segmento; a tradução do NOME de cada segmento já foi feita
// pelo validator via I18nTagNameFunc — esta função não traduz nada, só decide
// o que manter no caminho.
func stripEmbeddedSegments(rootType reflect.Type, namespace, locale string) string {
	segments := strings.Split(namespace, ".")
	typ := rootType
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	var out []string
	for i, seg := range segments {
		if i == 0 {
			continue // nome do tipo raiz, não faz parte do caminho
		}

		name := seg
		suffix := ""
		if idx := strings.Index(seg, "["); idx >= 0 {
			name = seg[:idx]
			suffix = seg[idx:]
		}

		if typ.Kind() != reflect.Struct {
			out = append(out, name+suffix)
			continue
		}

		var matchedField reflect.StructField
		var matchedMeta fieldMeta
		found := false
		for _, fm := range getFieldMeta(typ) {
			f := typ.Field(fm.Index)
			// o segmento já vem traduzido pelo validator (via I18nTagNameFunc);
			// compara tanto contra o nome traduzido quanto contra o nome Go cru
			// (embedded sem tag cai nesse segundo caso).
			translated, _ := resolveFieldKey(fm, locale)
			if translated == name || f.Name == name {
				matchedField = f
				matchedMeta = fm
				found = true
				break
			}
		}
		if !found {
			out = append(out, name+suffix)
			continue
		}

		nextType := matchedField.Type
		for nextType.Kind() == reflect.Ptr {
			nextType = nextType.Elem()
		}
		if nextType.Kind() == reflect.Slice || nextType.Kind() == reflect.Array {
			nextType = nextType.Elem()
			for nextType.Kind() == reflect.Ptr {
				nextType = nextType.Elem()
			}
		}

		if matchedMeta.EmbedPromote {
			typ = nextType
			continue // não emite segmento: acham o JSON, o namespace acompanha
		}

		out = append(out, name+suffix)
		typ = nextType
	}
	return strings.Join(out, ".")
}

// TranslateValidationErrors converte validator.ValidationErrors (produzido
// por um *validator.Validate configurado com reg.I18nTagNameFunc(locale)) em
// erros com nome de campo e mensagem no idioma pedido. rootType é o tipo da
// struct raiz validada (necessário só pra reconhecer e remover segmentos de
// campos embedded do caminho — a tradução em si é feita pelo próprio
// validator). Faz fallback pro locale canônico se `locale` não tiver
// tradutor registrado.
func (r *Registry) TranslateValidationErrors(errs validator.ValidationErrors, rootType reflect.Type, locale string) ([]ValidationError, error) {
	trans, ok := r.validatorTranslatorFor(locale)
	if !ok {
		return nil, fmt.Errorf("i18n: nenhum ut.Translator registrado para o locale %q nem para o fallback", locale)
	}

	out := make([]ValidationError, 0, len(errs))
	for _, fe := range errs {
		field := stripEmbeddedSegments(rootType, fe.Namespace(), locale)
		out = append(out, ValidationError{
			Field:   field,
			Tag:     fe.Tag(),
			Message: fe.Translate(trans),
		})
	}
	return out, nil
}

func (r *Registry) fromMap(raw map[string]interface{}, v reflect.Value, locale string) error {
	for _, fm := range getFieldMeta(v.Type()) {
		if fm.Hidden {
			continue
		}
		fv := v.Field(fm.Index)

		if fm.EmbedPromote {
			target := fv
			if target.Kind() == reflect.Ptr {
				if target.IsNil() {
					if !target.CanSet() {
						continue
					}
					target.Set(reflect.New(target.Type().Elem()))
				}
				target = target.Elem()
			}
			if target.Kind() != reflect.Struct {
				continue
			}
			if err := r.fromMap(raw, target, locale); err != nil {
				return err
			}
			continue
		}

		key, ok := resolveFieldKey(fm, locale)
		if !ok {
			continue
		}
		rawVal, present := raw[key]
		if !present {
			continue
		}

		if fm.EnumName != "" {
			strVal, ok := rawVal.(string)
			if !ok {
				return fmt.Errorf("campo %q: esperado string para o enum %q, recebido %T", key, fm.EnumName, rawVal)
			}
			canon, err := r.toCanonical(fm.EnumName, strVal, locale)
			if err != nil {
				return fmt.Errorf("campo %q: %w", key, err)
			}
			if !fv.CanSet() {
				continue
			}
			if fv.Kind() == reflect.String {
				fv.SetString(canon)
			} else {
				rv := reflect.ValueOf(canon)
				if rv.Type().ConvertibleTo(fv.Type()) {
					fv.Set(rv.Convert(fv.Type()))
				} else {
					return fmt.Errorf("campo %q: não foi possível converter enum canônico %q para %s", key, canon, fv.Type())
				}
			}
			continue
		}

		if isGenericMap(fv.Type()) {
			rawGeneric, ok := rawVal.(map[string]interface{})
			if !ok {
				return fmt.Errorf("campo %q: esperado objeto JSON, recebido %T", key, rawVal)
			}
			out := reflect.MakeMap(fv.Type())
			for capabilityKey, payload := range rawGeneric {
				canonKey := r.canonicalFieldName(capabilityKey, locale)
				translated, err := r.genericToCanonical(payload, locale)
				if err != nil {
					return fmt.Errorf("campo genérico %q: %w", capabilityKey, err)
				}
				b, err := json.Marshal(translated)
				if err != nil {
					return err
				}
				out.SetMapIndex(reflect.ValueOf(canonKey), reflect.ValueOf(json.RawMessage(b)))
			}
			fv.Set(out)
			continue
		}

		switch fv.Kind() {
		case reflect.Struct:
			nestedMap, ok := rawVal.(map[string]interface{})
			if !ok {
				continue
			}
			if err := r.fromMap(nestedMap, fv, locale); err != nil {
				return err
			}
		case reflect.Ptr:
			if fv.Type().Elem().Kind() == reflect.Struct {
				nestedMap, ok := rawVal.(map[string]interface{})
				if !ok {
					continue
				}
				newVal := reflect.New(fv.Type().Elem())
				if err := r.fromMap(nestedMap, newVal.Elem(), locale); err != nil {
					return err
				}
				fv.Set(newVal)
			} else {
				converted, err := convertSimple(rawVal, fv.Type())
				if err != nil {
					return err
				}
				fv.Set(converted)
			}
		case reflect.Slice:
			rawSlice, ok := rawVal.([]interface{})
			if !ok {
				continue
			}
			elemType := fv.Type().Elem()
			out := reflect.MakeSlice(fv.Type(), len(rawSlice), len(rawSlice))
			for idx, item := range rawSlice {
				if elemType.Kind() == reflect.Struct {
					itemMap, ok := item.(map[string]interface{})
					if !ok {
						continue
					}
					if err := r.fromMap(itemMap, out.Index(idx), locale); err != nil {
						return err
					}
				} else {
					converted, err := convertSimple(item, elemType)
					if err != nil {
						return err
					}
					out.Index(idx).Set(converted)
				}
			}
			fv.Set(out)
		default:
			if !fv.CanSet() {
				continue
			}
			converted, err := convertSimple(rawVal, fv.Type())
			if err != nil {
				return err
			}
			fv.Set(converted)
		}
	}
	return nil
}
