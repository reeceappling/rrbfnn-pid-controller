package pid

import (
	"testing"
	"gonum.org/v1/gonum/mat"
	"math/rand"
)

func BenchmarkVectorReplacement(b *testing.B) {
	existingVec := mat.NewVecDense(3, []float64{1.0, 2.0, 3.0})
	rnd := rand.New(rand.NewSource(42))
	createNewValues := func() [3]float64 {
		return [3]float64{rnd.Float64(), rnd.Float64(), rnd.Float64()}
	}
	b.Run("newVec Copy", func(b *testing.B) {
		newVals := createNewValues()
		newVec := mat.NewVecDense(3, newVals[:])
		existingVec.CopyVec(newVec)
	})
	b.Run("newVec SetUnderlying", func(b *testing.B) {
		newVals := createNewValues()
		newVec := mat.NewVecDense(3, newVals[:])
		*existingVec = *newVec
	})
	b.Run("newVec Set Pointer", func(b *testing.B) {
		newVals := createNewValues()
		newVec := mat.NewVecDense(3, newVals[:])
		existingVec = newVec
	})
	b.Run("replaceValues", func(b *testing.B) {
		for i, val := range createNewValues() {
			existingVec.SetVec(i, val)
		}
	})
}
