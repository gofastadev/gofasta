package whatsapp

import "encoding/base64"

// stdBase64 is the standard base64 encoder. Aliased so the call sites
// in ultramsg.go read more naturally without dragging encoding/base64
// into every provider file's import block.
var stdBase64 = base64.StdEncoding
