package utils

import (
	"strconv"
	"strings"
)

type KernelVersion struct {
	Min int
	Max int
}

type Unit string

const ()

type Capacity struct {
	size float32
	unit Unit
}

func ParseCapacity(s string) (Capacity, error) {
	switch {
	case strings.HasSuffix(s, "Gi"), strings.HasSuffix(s, "GB"):
		size, err := strconv.ParseFloat(strings.TrimRight(s, "Gi"), 0)
		if err != nil {
			return Capacity{}, err
		}
		return Capacity{size: float32(size), unit: "Gi"}, nil
	}

	return Capacity{}, nil
}
