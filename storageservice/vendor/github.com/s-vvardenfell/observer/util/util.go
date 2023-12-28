package util

import "os"

func CheckEnv(variableName, defaultValue string) string {
	if cn, ok := os.LookupEnv(variableName); ok {
		return cn
	}

	return defaultValue
}
