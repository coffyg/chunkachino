package chunkachino

// englishAbbreviations is a curated list of English abbreviations that end
// with a period but should NOT be treated as sentence boundaries.
//
// Entries are stored lowercase WITHOUT the trailing period. Lookup is
// case-insensitive on the token that immediately precedes the period.
//
// Keep this list tight and high-signal. A noisy list causes missed sentence
// boundaries (bad for TTS latency); a sparse list causes false splits mid-
// sentence (bad for prosody). When in doubt, leave it out.
var englishAbbreviations = map[string]struct{}{
	// Titles / honorifics
	"mr":    {},
	"mrs":   {},
	"ms":    {},
	"dr":    {},
	"prof":  {},
	"sr":    {},
	"jr":    {},
	"st":    {}, // Saint / Street
	"rev":   {},
	"hon":   {},
	"capt":  {},
	"cmdr":  {},
	"col":   {},
	"gen":   {},
	"gov":   {},
	"lt":    {},
	"maj":   {},
	"pres":  {},
	"sen":   {},
	"sgt":   {},

	// Latin / academic
	"e.g":  {},
	"i.e":  {},
	"etc":  {},
	"vs":   {},
	"viz":  {},
	"cf":   {},
	"al":   {}, // "et al."
	"ph.d": {},
	"m.d":  {},
	"b.a":  {},
	"b.s":  {},
	"m.a":  {},
	"m.s":  {},

	// Time / calendar
	"a.m": {},
	"p.m": {},
	"jan": {},
	"feb": {},
	"mar": {},
	"apr": {},
	"jun": {},
	"jul": {},
	"aug": {},
	"sep": {},
	"sept": {},
	"oct": {},
	"nov": {},
	"dec": {},
	"mon": {},
	"tue": {},
	"tues": {},
	"wed": {},
	"thu": {},
	"thur": {},
	"thurs": {},
	"fri": {},
	"sat": {},
	"sun": {},

	// Units / measurements
	"no":   {}, // number
	"vol":  {},
	"pg":   {},
	"pp":   {},
	"fig":  {},
	"ch":   {},
	"sec":  {},
	"min":  {},
	"hr":   {},
	"ft":   {},
	"in":   {},
	"lb":   {},
	"lbs":  {},
	"oz":   {},
	"kg":   {},
	"mg":   {},
	"ml":   {},
	"mm":   {},
	"cm":   {},
	"km":   {},
	"mph":  {},

	// Geo / org
	"u.s":   {},
	"u.k":   {},
	"u.s.a": {},
	"e.u":   {},
	"n.y":   {},
	"l.a":   {},
	"d.c":   {},
	"co":    {},
	"inc":   {},
	"ltd":   {},
	"corp":  {},
	"dept":  {},
	"univ":  {},
	"assn":  {},
	"bros":  {},
	"est":   {},
}
