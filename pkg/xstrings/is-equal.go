package xstrings

func IsEqual(target string, values ...string) bool {
	for _, value := range values {
		if target == value {
			return true
		}
	}

	return false
}
