package chunkachino

// germanAbbreviations is a curated list of German (de-DE) abbreviations
// that end with a period but should NOT be treated as sentence boundaries.
//
// Entries are stored lowercase WITHOUT the trailing period. Multi-period
// abbreviations like "z.b" (z. B.), "u.a" (u. a.), "d.h" (d. h.) are
// matched as a unit by the classifier's walk-backward logic, which
// matters a lot in German where dotted multi-letter abbreviations are
// especially common.
//
// Note: German also uses spaced forms like "z. B." with a space between
// the letters. That form naturally gets handled because the first "z."
// is a single-letter-initial that only triggers the initial-lookahead
// rule when followed by another uppercase letter + period — which "B."
// is. The explicit "z.b" entry covers the unspaced form.
var germanAbbreviations = map[string]struct{}{
	// Titles / honorifics
	"hr":      {}, // Herr
	"hrn":     {}, // Herrn
	"fr":      {}, // Frau
	"frl":     {}, // Fräulein
	"dr":      {},
	"prof":    {},
	"dipl":    {}, // Diplom
	"dipl.-ing": {}, // Dipl.-Ing.
	"dipl-ing":  {}, // variant
	"ing":     {}, // Ingenieur
	"mag":     {}, // Magister
	"med":     {}, // Medizin
	"jur":     {}, // Jurist
	"sen":     {}, // Senior
	"jun":     {}, // Junior

	// Common sentence-internal abbreviations (HIGH SIGNAL)
	"bzw":   {}, // beziehungsweise
	"z.b":   {}, // zum Beispiel
	"zb":    {},
	"d.h":   {}, // das heißt
	"dh":    {},
	"u.a":   {}, // unter anderem / und andere
	"ua":    {},
	"u.ä":   {}, // und Ähnliches
	"uä":    {},
	"o.ä":   {}, // oder Ähnliches
	"oä":    {},
	"u.v.a": {}, // und viele(s) andere
	"u.v.m": {}, // und vieles mehr
	"usw":   {}, // und so weiter
	"usf":   {}, // und so fort
	"ggf":   {}, // gegebenenfalls
	"evtl":  {}, // eventuell
	"bspw":  {}, // beispielsweise
	"v.a":   {}, // vor allem
	"va":    {},
	"i.d.r": {}, // in der Regel
	"idr":   {},
	"s.o":   {}, // siehe oben
	"s.u":   {}, // siehe unten
	"u.u":   {}, // unter Umständen
	"z.t":   {}, // zum Teil
	"zt":    {},
	"z.zt":  {}, // zur Zeit
	"zz":    {},
	"vgl":   {}, // vergleiche
	"ca":    {}, // circa
	"bzgl":  {}, // bezüglich
	"etc":   {},
	"inkl":  {}, // inklusive
	"exkl":  {}, // exklusive
	"max":   {},
	"min":   {},

	// Time / calendar
	"v.chr": {}, // vor Christus
	"n.chr": {}, // nach Christus
	"jh":    {}, // Jahrhundert
	"jhd":   {},
	"jan":   {},
	"feb":   {},
	"mär":   {}, // März
	"mar":   {},
	"apr":   {},
	"mai":   {},
	"jul":   {},
	"aug":   {},
	"sep":   {},
	"sept":  {},
	"okt":   {},
	"nov":   {},
	"dez":   {},
	"mo":    {}, // Montag
	"di":    {},
	"mi":    {},
	"do":    {},
	"so":    {}, // Sonntag (NB: "s.o." = siehe oben; "so." = Sonntag — overlap accepted; both stay non-terminating)

	// References / units
	"nr":    {}, // Nummer
	"nrn":   {},
	"s":     {}, // Seite
	"bd":    {}, // Band
	"bde":   {},
	"bdn":   {},
	"abb":   {}, // Abbildung
	"abs":   {}, // Absatz / Abschnitt
	"art":   {}, // Artikel
	"kap":   {}, // Kapitel
	"anm":   {}, // Anmerkung
	"hrsg":  {}, // Herausgeber
	"übers": {}, // Übersetzer
	"uebers": {},
	"aufl":  {}, // Auflage
	"ausg":  {}, // Ausgabe
	"ff":    {}, // folgende (Seiten)
	"geb":   {}, // geboren
	"gest":  {}, // gestorben
	"verst": {}, // verstorben

	// Geo / address / org
	"str":   {}, // Straße
	"pl":    {}, // Platz
	"plz":   {}, // Postleitzahl
	"ort":   {},
	"b":     {}, // bei
	"d":     {}, // der/die/das (address use is rare; kept conservative)
	"gmbh":  {},
	"ag":    {}, // Aktiengesellschaft
	"kg":    {}, // Kommanditgesellschaft (also kilogram — acceptable)
	"e.v":   {}, // eingetragener Verein
	"ev":    {},
	"co":    {},
	"mio":   {}, // Million
	"mrd":   {}, // Milliarde
	"tsd":   {}, // Tausend

	// Misc
	"tel":   {},
	"fax":   {},
	"zzgl":  {}, // zuzüglich
	"abzgl": {}, // abzüglich
	"geg":   {}, // gegebenenfalls (variant)
}
