package chunkachino

// portugueseAbbreviations is a curated list of European Portuguese (pt-PT)
// abbreviations that end with a period but should NOT be treated as
// sentence boundaries.
//
// Entries are stored lowercase WITHOUT the trailing period. Multi-period
// abbreviations are matched as a unit by the classifier's walk-backward
// logic (e.g. "v.exa", "p.ex").
//
// Scope is Portugal (not Brazilian variants where the two diverge, e.g.
// "Lda." is Portugal, "Ltda." is Brazil — we include Lda. and still
// accept ltda as a bonus for noise tolerance).
var portugueseAbbreviations = map[string]struct{}{
	// Titles / honorifics
	"sr":    {}, // Senhor
	"sra":   {}, // Senhora
	"srs":   {},
	"sras":  {},
	"srta":  {}, // Senhorita
	"d":     {}, // Dom
	"dona":  {}, // (lowercased "dona")
	"dr":    {}, // Doutor
	"dra":   {}, // Doutora
	"drs":   {},
	"dras":  {},
	"prof":  {}, // Professor
	"profa": {}, // Professora
	"profª": {}, // Professora (feminine ordinal form)
	"exmo":  {}, // Excelentíssimo
	"exma":  {}, // Excelentíssima
	"ilmo":  {}, // Ilustríssimo
	"ilma":  {},
	"v.exa": {}, // Vossa Excelência
	"vexa":  {},
	"v.sa":  {}, // Vossa Senhoria
	"vsa":   {},
	"rev":   {}, // Reverendo
	"revmo": {},
	"eng":   {}, // Engenheiro
	"engº":  {},
	"eng.º": {},
	"arq":   {}, // Arquitecto
	"arqº":  {},
	"fr":    {}, // Frei
	"mons":  {}, // Monsenhor
	"mad":   {}, // Madre

	// Latin / academic / connectors
	"etc":   {},
	"p.ex":  {}, // por exemplo
	"pex":   {},
	"ex":    {}, // exemplo / exa.
	"i.e":   {}, // isto é
	"e.g":   {},
	"vs":    {},
	"cf":    {},
	"cfr":   {},
	"a.c":   {}, // antes de Cristo
	"d.c":   {}, // depois de Cristo
	"a.m":   {},
	"p.m":   {},

	// References / units
	"pág":  {}, // página
	"pag":  {},
	"págs": {},
	"pags": {},
	"p":    {}, // página (short)
	"pp":   {},
	"vol":  {},
	"vols": {},
	"cap":  {}, // capítulo
	"caps": {},
	"art":  {}, // artigo
	"arts": {},
	"artº": {},
	"fig":  {},
	"tab":  {}, // tabela
	"séc":  {}, // século
	"sec":  {},
	"n.º":  {}, // número
	"nº":   {},
	"no":   {}, // numero (ASCII)
	"núm":  {},
	"num":  {},
	"ed":   {}, // edição / editor
	"trad": {}, // tradução
	"org":  {}, // organização / organizado por
	"coord": {}, // coordenação

	// Time / calendar
	"jan": {}, // janeiro
	"fev": {},
	"mar": {}, // março
	"abr": {},
	"mai": {}, // maio
	"jun": {},
	"jul": {},
	"ago": {},
	"set": {}, // setembro (PT uses "set"; "sep" is BR)
	"sep": {},
	"out": {},
	"nov": {},
	"dez": {},
	"seg": {}, // segunda-feira
	"ter": {},
	"qua": {},
	"qui": {},
	"sex": {},
	"sáb": {},
	"sab": {},
	"dom": {},

	// Geo / org / address
	"s.a":   {}, // sociedade anónima
	"lda":   {}, // sociedade por quotas (Portugal)
	"ltda":  {}, // Brazilian variant, still tolerated
	"c.ª":   {}, // companhia
	"cia":   {},
	"co":    {},
	"av":    {}, // avenida
	"r":     {}, // rua
	"tv":    {}, // travessa
	"est":   {}, // estrada
	"lg":    {}, // largo
	"pç":    {}, // praça
	"pc":    {},

	// Misc common
	"aprox": {}, // aproximadamente
	"tel":   {}, // telefone
	"telef": {},
	"telm":  {}, // telemóvel
	"fax":   {},
	"obs":   {}, // observação
	"ref":   {}, // referência
	"adm":   {}, // administração
	"gen":   {}, // general
	"cel":   {}, // coronel
}
