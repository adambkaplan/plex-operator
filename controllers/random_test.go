package controllers

import (
	"fmt"
	"math/rand"
	"strconv"
)

func RandomName(baseName string) string {
	return fmt.Sprintf("%s-%s", baseName, strconv.Itoa(rand.Intn(10000)))
}
