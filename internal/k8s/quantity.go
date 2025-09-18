package k8s

import (
	"k8s.io/apimachinery/pkg/api/resource"
)

// Binary size.
type Binary int

// Quantity returns Binary as SI binary.
func (b Binary) Quantity() resource.Quantity {
	return *resource.NewQuantity(int64(b), resource.BinarySI)
}

// Binary constants.
const (
	Byte Binary = 1

	KB = Byte * 1024
	MB = KB * 1024
	GB = MB * 1024
)

type CPUs int

const (
	MilliCPU CPUs = 1
	CPU           = MilliCPU * 1_000
)

func (b CPUs) Quantity() resource.Quantity {
	return *resource.NewMilliQuantity(int64(b), resource.DecimalSI)
}

// Quantity returns resource.Quantity pointer.
func Quantity(quantity resource.Quantity) *resource.Quantity {
	return &quantity
}
