package chunkachino

// frenchAbbreviations is a curated list of French abbreviations that end
// with a period but should NOT be treated as sentence boundaries.
//
// Entries are stored lowercase WITHOUT the trailing period.
var frenchAbbreviations = map[string]struct{}{
	// Titles / honorifics
	"m":    {}, // Monsieur
	"mm":   {}, // Messieurs
	"mme":  {}, // Madame
	"mmes": {}, // Mesdames
	"mlle": {}, // Mademoiselle
	"mgr":  {}, // Monseigneur
	"me":   {}, // Maître
	"dr":   {},
	"pr":   {}, // Professeur
	"st":   {}, // Saint
	"ste":  {}, // Sainte
	"sts":  {},
	"stes": {},

	// Latin (used in French too)
	"etc":  {},
	"e.g":  {}, // sometimes used
	"p.ex": {}, // par exemple
	"c.-à-d": {},
	"cad":  {}, // c-à-d variant
	"vs":   {},
	"cf":   {},
	"ex":   {},
	"env":  {}, // environ
	"av":   {}, // avant / avenue
	"boul": {}, // boulevard
	"bd":   {},
	"cap":  {}, // capitaine / capitale
	"qté":  {},

	// Time / calendar
	"janv": {},
	"févr": {},
	"fev":  {},
	"avr":  {},
	"juil": {},
	"sept": {},
	"déc":  {},
	"dec":  {},
	// Jours
	"lun": {},
	"mar": {},
	"mer": {},
	"jeu": {},
	"ven": {},
	"sam": {},
	"dim": {},

	// Units / misc
	"no":   {}, // numéro
	"nos":  {},
	"p":    {}, // page
	"pp":   {},
	"vol":  {},
	"chap": {},
	"fig":  {},
	"min":  {},
	"sec":  {},
	"h":    {}, // heure(s) — "14 h."
	"réf":  {},
	"tél":  {},
	"fax":  {},

	// Geo / org
	"sté":   {}, // société
	"cie":   {},
	"ets":   {}, // établissements
	"sarl":  {},
	"sa":    {},
	"ass":   {},
	"dép":   {},
	"arr":   {},
}
