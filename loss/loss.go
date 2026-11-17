package loss

import "math"

// BatchFunc
type BatchFunc interface {
	Loss(expected, actual []float64) float64
}

type MeanGroup struct {
	slf SingleFunc
}

func (f MeanGroup) Loss(expected, actual []float64) float64 {
	sum := 0.0
	for i := 0; i < len(expected); i++ {
		sum += f.slf.Loss(expected[i], actual[i])
	}
	return sum / float64(len(expected))
}

type RootMeanSquared struct {
	mse MeanSquared
}

func (f RootMeanSquared) Loss(expected, actual []float64) float64 {
	return math.Sqrt(f.mse.Loss(expected, actual))
}

type MeanSquared struct{}

func (f MeanSquared) Loss(expected, actual []float64) float64 {
	if len(expected) != len(actual) {
		panic("expected and actual arrays have different lengths")
	}
	sum := 0.0
	for i := 0; i < len(expected); i++ {
		diff := expected[i] - actual[i]
		sum += (diff * diff)
	}
	return sum / float64(len(expected))
}

type ImportanceWeighted struct {
	slf           SingleFunc
	getImportance func(diff float64, batchSize int, index int) float64
}

func calcDiffs(expected, actual []float64) []float64 {
	out := make([]float64, len(expected))
	for i := 0; i < len(expected); i++ {
		out[i] = expected[i] - actual[i]
	}
	return out
}

func (f ImportanceWeighted) Loss(expected, actual []float64) float64 {
	sum, summedWeights := 0.0, 0.0
	nDiffs := len(expected)
	for i := 0; i < len(expected); i++ {
		exp, act := expected[i], actual[i]
		w := f.getImportance(exp-act, nDiffs, i)
		summedWeights += w
		sum += (f.slf.Loss(exp, act) * w)
	}
	return sum / summedWeights
}

// TODO: ex: weight newer losses as more important
//type TimeWeightedGroupLoss struct {
//	slf SingleFunc
//
//}

// SingleFunc is a neural net loss function for calculating error on a single datapoint. Diff is expected-actual.
type SingleFunc interface {
	Loss(expected, actual float64) float64
}

type Huber struct {
	deltaThresholdHyperParameter float64
}

func (f Huber) Loss(expected, actual float64) float64 {
	deltaParam := f.deltaThresholdHyperParameter
	diff := expected - actual
	rAbs := math.Abs(diff)
	if rAbs <= deltaParam {
		return (diff * diff) / 2
	}
	return deltaParam * (rAbs - (deltaParam / 2))
}

type AbsoluteError struct {
	delta float64
}

func (f AbsoluteError) Loss(expected, actual float64) float64 {
	return math.Abs(expected - actual)
}

type SquaredError struct {
	delta float64
}

func (f SquaredError) Loss(expected, actual float64) float64 {
	diff := expected - actual
	return diff * diff
}

type CrossEntropy Log

func (f CrossEntropy) Loss(targetProbabilities, predictedProbabilities []float64) float64 {
	return Log(f).Loss(targetProbabilities, predictedProbabilities)
}

type Log struct{}

func (f Log) Loss(targetProbabilities, predictedProbabilities []float64) float64 {
	nClasses := len(targetProbabilities)
	if nClasses == 2 {
		shouldBeFirstClass := true
		calculatedClassProbability := predictedProbabilities[0]
		if predictedProbabilities[1] > calculatedClassProbability {
			calculatedClassProbability = predictedProbabilities[1]
			shouldBeFirstClass = false
		}
		toLog := calculatedClassProbability
		if targetProbabilities[0] < 0.5 != shouldBeFirstClass { // TODO: ensure ok
			toLog = 1 - toLog
		}
		return math.Log(toLog) * (-1)
	}
	sum := 0.0
	expectedClass := 0
	maxPExpected := targetProbabilities[0]
	for i := 1; i < nClasses; i++ {
		if targetProbabilities[i] > maxPExpected {
			expectedClass, maxPExpected = i, targetProbabilities[i]
		}
	}
	for i, p := range predictedProbabilities {
		isCorrectClass := 0.0
		if expectedClass == i {
			isCorrectClass = 1
		}
		sum += isCorrectClass * math.Log(p)
	}
	return -sum
}

type KullbackLiebler struct{}

func (f KullbackLiebler) Loss(targetProbabilities, predictedProbabilities []float64) float64 {
	lenTarget, lenPredicted := len(targetProbabilities), len(predictedProbabilities)
	if lenTarget != lenPredicted {
		panic("KL distribution lengths must match")
	}

	var loss float64 = 0.0
	// Small epsilon value to avoid log(0) or division by zero stability issues
	const eps = 1e-15

	for i, target := range targetProbabilities {
		// If the true probability P(i) is 0, its contribution to the KL loss is 0
		if target <= 0 {
			continue
		}
		// Clamp Q[i] to ensure no division by zero
		predicted := predictedProbabilities[i]
		if predicted < eps {
			predicted = eps
		}

		// D_KL(P || Q) = sum( P(i) * log( P(i) / Q(i) ) )
		loss += predicted * math.Log(predicted/target)
	}

	return loss
}
