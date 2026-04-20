package chunkachino

// spanishAbbreviations is a curated list of Spanish (es-ES) abbreviations
// that end with a period but should NOT be treated as sentence boundaries.
//
// Entries are stored lowercase WITHOUT the trailing period. Multi-period
// abbreviations like "ee.uu" are matched as a single unit by the
// classifier's walk-backward-through-inner-dots logic.
//
// Scope is Peninsular Spanish. Kept tight and high-signal; a noisy list
// costs sentence-boundary latency, a sparse list costs false splits.
var spanishAbbreviations = map[string]struct{}{
	// Titles / honorifics
	"sr":    {}, // Señor
	"sra":   {}, // Señora
	"srta":  {}, // Señorita
	"sres":  {}, // Señores
	"sras":  {}, // Señoras
	"d":     {}, // Don
	"dña":   {}, // Doña
	"dna":   {}, // Doña (ASCII variant)
	"dr":    {}, // Doctor
	"dra":   {}, // Doctora
	"dres":  {}, // Doctores
	"lic":   {}, // Licenciado/a
	"ing":   {}, // Ingeniero/a
	"arq":   {}, // Arquitecto/a
	"prof":  {}, // Profesor/a
	"profa": {}, // Profesora
	"excmo": {}, // Excelentísimo
	"excma": {}, // Excelentísima
	"ilmo":  {}, // Ilustrísimo
	"ilma":  {}, // Ilustrísima
	"rvdo":  {}, // Reverendo
	"hno":   {}, // Hermano
	"hna":   {}, // Hermana
	"fr":    {}, // Fray
	"mons":  {}, // Monseñor
	"ud":    {}, // Usted
	"uds":   {}, // Ustedes
	"vd":    {}, // Usted (older form)
	"vds":   {}, // Ustedes (older form)

	// Latin / academic / connectors
	"etc":   {},
	"p.ej":  {}, // por ejemplo
	"pej":   {}, // variant
	"ej":    {}, // ejemplo
	"i.e":   {}, // i.e.
	"e.g":   {},
	"vs":    {},
	"cf":    {},
	"cfr":   {}, // confer
	"a.c":   {}, // antes de Cristo
	"d.c":   {}, // después de Cristo
	"a.m":   {},
	"p.m":   {},

	// Units / references
	"no":   {}, // número
	"núm":  {}, // número
	"num":  {},
	"pág":  {}, // página
	"pag":  {},
	"págs": {},
	"pags": {},
	"vol":  {},
	"vols": {},
	"cap":  {}, // capítulo
	"art":  {}, // artículo
	"fig":  {},
	"tomo": {},
	"ed":   {}, // edición / editor
	"trad": {}, // traducción
	"apdo": {}, // apartado
	"depto": {}, // departamento
	"dpto": {},

	// Time / calendar
	"ene": {}, // enero
	"feb": {},
	"mar": {}, // marzo
	"abr": {},
	"may": {},
	"jun": {},
	"jul": {},
	"ago": {},
	"sep": {},
	"sept": {},
	"oct": {},
	"nov": {},
	"dic": {}, // diciembre
	"lun": {}, // lunes
	"mie": {}, // miércoles (ASCII)
	"jue": {},
	"vie": {},
	"sáb": {},
	"sab": {},
	"dom": {},

	// Geo / org
	"ee.uu": {}, // Estados Unidos
	"eeuu":  {}, // variant without dots
	"rr.hh": {}, // recursos humanos
	"ss.aa": {}, // sus altezas
	"s.a":   {}, // sociedad anónima
	"s.l":   {}, // sociedad limitada
	"s.r.l": {}, // sociedad de responsabilidad limitada
	"cía":   {}, // compañía
	"cia":   {},
	"hnos":  {}, // hermanos
	"avda":  {}, // avenida
	"av":    {}, // avenida / avenue
	"c":     {}, // calle (disabled risk: too short - keep for "c/ Mayor 5, 2" style but only period form matches; worth it for addresses)
	"ctra":  {}, // carretera

	// Misc common
	"aprox":  {}, // aproximadamente
	"atte":   {}, // atentamente
	"tel":    {}, // teléfono
	"teléf":  {},
	"telef":  {},
	"fax":    {},
	"admón":  {}, // administración
	"admon":  {},
	"gral":   {}, // general
}
