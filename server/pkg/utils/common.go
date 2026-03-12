package utils

func ContainsAny(slice []string, values []string) bool {
	for _, v := range slice {
		for _, val := range values {
			if v == val {
				return true
			}
		}
	}
	return false
}
