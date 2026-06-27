package textutils

// IsValidUTF8Content checks if content is valid UTF-8 text (not binary garbage)
func IsValidUTF8Content(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	// Check first 1KB for binary indicators
	checkLen := len(data)
	if checkLen > 1024 {
		checkLen = 1024
	}

	nullCount := 0
	nonPrintable := 0
	for i := 0; i < checkLen; i++ {
		b := data[i]
		if b == 0 {
			nullCount++
		}
		// Count non-printable non-whitespace ASCII control chars
		if b < 32 && b != '\t' && b != '\n' && b != '\r' {
			nonPrintable++
		}
	}

	// If >5% null bytes or >20% non-printable, likely binary
	if float64(nullCount)/float64(checkLen) > 0.05 {
		return false
	}
	if float64(nonPrintable)/float64(checkLen) > 0.20 {
		return false
	}

	return true
}
