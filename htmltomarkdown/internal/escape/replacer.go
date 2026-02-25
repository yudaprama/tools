package escape

import "github.com/getkawai/tools/htmltomarkdown/marker"

var placeholderRune rune = marker.MarkerEscaping

// IMPORTANT: Only internally we assume it is only byte
var placeholderByte byte = marker.BytesMarkerEscaping[0]
