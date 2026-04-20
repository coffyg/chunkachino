package chunkachino

import (
	"strings"
	"testing"
)

// --- Locale key acceptance -------------------------------------------------

func TestLocale_ExactlyTheAcceptedKeys(t *testing.T) {
	// The contract per spec: five literal locale strings + two legacy
	// short aliases. Anything else falls back to English.
	cases := []struct {
		in   string
		want string
	}{
		// Exact locale strings
		{"en-US", "en"},
		{"es-ES", "es"},
		{"fr-FR", "fr"},
		{"pt-PT", "pt"},
		{"de-DE", "de"},

		// Legacy short aliases
		{"en", "en"},
		{"fr", "fr"},

		// Case insensitivity on the accepted set
		{"EN-US", "en"},
		{"es-es", "es"},
		{"Pt-Pt", "pt"},
		{"DE-de", "de"},
		{"FR", "fr"},

		// Underscore separator tolerated (common in gettext/POSIX forms)
		{"en_US", "en"},
		{"pt_PT", "pt"},

		// NOT accepted → fall back to English
		{"", "en"},
		{"en-GB", "en"},
		{"fr-CA", "en"},
		{"es", "en"},
		{"pt", "en"},
		{"de", "en"},
		{"ja-JP", "en"},
		{"zh", "en"},
		{"garbage", "en"},
	}
	for _, tc := range cases {
		if got := resolveLanguage(tc.in); got != tc.want {
			t.Errorf("resolveLanguage(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLocale_AbbreviationsForLang_PointsAtRightCatalog(t *testing.T) {
	// Quick sanity: the correct map is returned for each accepted key, by
	// probing a language-unique abbreviation.
	cases := []struct {
		lang   string
		probe  string // lowercase word that exists ONLY in that lang's map
	}{
		{"en-US", "mrs"},
		{"es-ES", "srta"},
		{"fr-FR", "mme"},
		{"pt-PT", "lda"},
		{"de-DE", "bzw"},
	}
	for _, tc := range cases {
		abbrs := abbreviationsForLang(tc.lang)
		if _, ok := abbrs[tc.probe]; !ok {
			t.Errorf("lang %q: expected probe %q present, not found", tc.lang, tc.probe)
		}
	}
}

// --- Spanish ---------------------------------------------------------------

func TestAbbreviations_Spanish_CanonicalSentence(t *testing.T) {
	// Flo's canonical test: "El Sr. García vive aquí." must stay one chunk.
	text := "El Sr. García vive aquí."
	for _, lang := range []string{"es-ES", "ES-ES", "es_ES"} {
		for _, n := range []int{1, 2, 3, 5, 1000} {
			got := feed(Config{Mode: ModeSentence, MaxWords: 40, Language: lang}, splitEveryN(text, n))
			compareChunks(t, got, []string{text})
		}
	}
}

func TestAbbreviations_Spanish(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "Sra. in address",
			text: "La Sra. Pérez vino hoy. Muy amable.",
			want: []string{"La Sra. Pérez vino hoy.", "Muy amable."},
		},
		{
			name: "Srta.",
			text: "La Srta. López trabaja aquí. Bienvenida.",
			want: []string{"La Srta. López trabaja aquí.", "Bienvenida."},
		},
		{
			name: "Dr. and Dra.",
			text: "El Dr. Ruiz y la Dra. Gómez hablaron. Buen día.",
			want: []string{"El Dr. Ruiz y la Dra. Gómez hablaron.", "Buen día."},
		},
		{
			name: "D. and Dña.",
			text: "D. Juan y Dña. María asistieron. Gracias.",
			want: []string{"D. Juan y Dña. María asistieron.", "Gracias."},
		},
		{
			// "etc." is a known abbreviation, so it should NOT flush here.
			// The whole text has only one real sentence terminator (final
			// period), so the whole thing is a single chunk.
			name: "etc. does not falsely flush mid-phrase",
			text: "Compré manzanas, peras, etc. Luego rico.",
			want: []string{"Compré manzanas, peras, etc. Luego rico."},
		},
		{
			name: "p.ej. connector",
			text: "Usa fruta, p.ej. manzanas o peras. Rico.",
			want: []string{"Usa fruta, p.ej. manzanas o peras.", "Rico."},
		},
		{
			name: "Ud. / Uds.",
			text: "Ud. y Uds. deben firmar. Por favor.",
			want: []string{"Ud. y Uds. deben firmar.", "Por favor."},
		},
		{
			name: "S.A. corporate",
			text: "Telefónica S.A. cotiza en bolsa. Bien.",
			want: []string{"Telefónica S.A. cotiza en bolsa.", "Bien."},
		},
		{
			name: "EE.UU. country",
			text: "Viajó a EE.UU. la semana pasada. Bonito.",
			want: []string{"Viajó a EE.UU. la semana pasada.", "Bonito."},
		},
		{
			name: "pág. reference",
			text: "Mira la pág. 42 del libro. Interesante.",
			want: []string{"Mira la pág. 42 del libro.", "Interesante."},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, n := range []int{1, 2, 3, 5, 9, 1000} {
				got := feed(Config{Mode: ModeSentence, MaxWords: 40, Language: "es-ES"}, splitEveryN(tc.text, n))
				compareChunks(t, got, tc.want)
			}
		})
	}
}

// --- Portuguese ------------------------------------------------------------

func TestAbbreviations_Portuguese_CanonicalSentence(t *testing.T) {
	text := "O Prof. Dr. Santos chegou."
	for _, lang := range []string{"pt-PT", "PT-PT", "pt_PT"} {
		for _, n := range []int{1, 2, 3, 5, 1000} {
			got := feed(Config{Mode: ModeSentence, MaxWords: 40, Language: lang}, splitEveryN(text, n))
			compareChunks(t, got, []string{text})
		}
	}
}

func TestAbbreviations_Portuguese(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "Sr. / Sra.",
			text: "O Sr. e a Sra. Silva chegaram. Bem-vindos.",
			want: []string{"O Sr. e a Sra. Silva chegaram.", "Bem-vindos."},
		},
		{
			name: "Exmo. / Exma.",
			text: "Exmo. Senhor, Exma. Senhora. Boa tarde.",
			want: []string{"Exmo. Senhor, Exma. Senhora.", "Boa tarde."},
		},
		{
			name: "V. Exa.",
			text: "Cumprimentos a V. Exa. pela nomeação. Parabéns.",
			want: []string{"Cumprimentos a V. Exa. pela nomeação.", "Parabéns."},
		},
		{
			name: "Eng. / Arq.",
			text: "O Eng. Costa e a Arq. Lopes assinaram. Tudo bem.",
			want: []string{"O Eng. Costa e a Arq. Lopes assinaram.", "Tudo bem."},
		},
		{
			name: "Lda. corporate",
			text: "Chamei a ACME Lda. ontem. Nada.",
			want: []string{"Chamei a ACME Lda. ontem.", "Nada."},
		},
		{
			name: "p.ex. connector",
			text: "Traz fruta, p.ex. maçãs ou peras. Obrigado.",
			want: []string{"Traz fruta, p.ex. maçãs ou peras.", "Obrigado."},
		},
		{
			name: "séc. reference",
			text: "No séc. XIX foi diferente. Claro.",
			want: []string{"No séc. XIX foi diferente.", "Claro."},
		},
		{
			name: "pág. / vol.",
			text: "Ver pág. 15, vol. 2 da colecção. Boa leitura.",
			want: []string{"Ver pág. 15, vol. 2 da colecção.", "Boa leitura."},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, n := range []int{1, 2, 3, 5, 9, 1000} {
				got := feed(Config{Mode: ModeSentence, MaxWords: 40, Language: "pt-PT"}, splitEveryN(tc.text, n))
				compareChunks(t, got, tc.want)
			}
		})
	}
}

// --- German ----------------------------------------------------------------

func TestAbbreviations_German_CanonicalSentence(t *testing.T) {
	text := "z.B. der Dr. Müller ist hier."
	for _, lang := range []string{"de-DE", "DE-DE", "de_DE"} {
		for _, n := range []int{1, 2, 3, 5, 1000} {
			got := feed(Config{Mode: ModeSentence, MaxWords: 40, Language: lang}, splitEveryN(text, n))
			compareChunks(t, got, []string{text})
		}
	}
}

func TestAbbreviations_German(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "Hr. / Fr.",
			text: "Hr. Schmidt und Fr. Weber sind hier. Gut.",
			want: []string{"Hr. Schmidt und Fr. Weber sind hier.", "Gut."},
		},
		{
			name: "Dr. / Prof.",
			text: "Der Dr. Braun und Prof. Hahn kamen. Schön.",
			want: []string{"Der Dr. Braun und Prof. Hahn kamen.", "Schön."},
		},
		{
			name: "z.B. / d.h. / u.a.",
			text: "Das sind z.B. d.h. u.a. Gründe dafür. Alles klar.",
			want: []string{"Das sind z.B. d.h. u.a. Gründe dafür.", "Alles klar."},
		},
		{
			name: "bzw. connector",
			text: "Kinder bzw. Jugendliche dürfen rein. Willkommen.",
			want: []string{"Kinder bzw. Jugendliche dürfen rein.", "Willkommen."},
		},
		{
			name: "usw. / etc.",
			text: "Äpfel, Birnen usw. etc. kaufen. Okay.",
			want: []string{"Äpfel, Birnen usw. etc. kaufen.", "Okay."},
		},
		{
			name: "ggf. / evtl.",
			text: "Wir sehen uns ggf. evtl. morgen. Bis dann.",
			want: []string{"Wir sehen uns ggf. evtl. morgen.", "Bis dann."},
		},
		{
			name: "ca. / Nr.",
			text: "Haus Nr. 17 ist ca. 500 Meter weit. Komm mit.",
			want: []string{"Haus Nr. 17 ist ca. 500 Meter weit.", "Komm mit."},
		},
		{
			name: "v.Chr. / n.Chr.",
			text: "200 v.Chr. bis 300 n.Chr. passierte viel. Interessant.",
			want: []string{"200 v.Chr. bis 300 n.Chr. passierte viel.", "Interessant."},
		},
		{
			name: "Str. / Bd.",
			text: "Haupt Str. 5, Bd. 3 lesen. Gut.",
			want: []string{"Haupt Str. 5, Bd. 3 lesen.", "Gut."},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, n := range []int{1, 2, 3, 5, 9, 1000} {
				got := feed(Config{Mode: ModeSentence, MaxWords: 40, Language: "de-DE"}, splitEveryN(tc.text, n))
				compareChunks(t, got, tc.want)
			}
		})
	}
}

// --- Unknown locale falls back to English without error --------------------

func TestUnknownLocale_FallsBackToEnglish(t *testing.T) {
	// A text that relies on ENGLISH abbreviation rules but is fed with an
	// unsupported language tag. We expect English behaviour.
	text := "Mr. Smith went home. End."
	for _, lang := range []string{"ja-JP", "zh-CN", "garbage", "es" /* short-form NOT accepted */} {
		for _, n := range []int{1, 3, 100} {
			got := feed(Config{Mode: ModeSentence, MaxWords: 40, Language: lang}, splitEveryN(text, n))
			compareChunks(t, got, []string{"Mr. Smith went home.", "End."})
		}
	}
}

// --- Sanity: no locale list contains empty strings or trailing periods -----

func TestAllLocales_EntriesAreCleanLowercaseNoPeriod(t *testing.T) {
	catalogs := map[string]map[string]struct{}{
		"en": englishAbbreviations,
		"fr": frenchAbbreviations,
		"es": spanishAbbreviations,
		"pt": portugueseAbbreviations,
		"de": germanAbbreviations,
	}
	for name, m := range catalogs {
		if len(m) == 0 {
			t.Errorf("%s catalog is empty", name)
		}
		for k := range m {
			if k == "" {
				t.Errorf("%s catalog contains empty entry", name)
			}
			if strings.HasSuffix(k, ".") {
				t.Errorf("%s catalog entry %q has trailing period (should be stripped)", name, k)
			}
			if strings.ToLower(k) != k {
				t.Errorf("%s catalog entry %q is not lowercase", name, k)
			}
		}
	}
}
