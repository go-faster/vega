package cli

import (
	"os"
	"strconv"
)

func BoolEnv(k string) bool {
	v, _ := strconv.ParseBool(os.Getenv(k))
	return v
}
