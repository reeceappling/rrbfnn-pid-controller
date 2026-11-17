package coreSystem

import (
	"fmt"
	"math"
	"math/rand"
	"encoding/json"
)

// RecurrentRBFNetwork implements an adaptive recurrent RBF network for online learning
type RecurrentRBFNetwork struct {
	// Architecture
	NumInputs  int
	NumHidden  int // Variable
	MaxHidden  int
	NumOutputs int // Always 3: Kp, Ki, Kd
	MinSpread  float64

	// Parameters
	LearningRates LearningRates

	// Learnable weights
	Nodes []RBFNode // Hidden RBF Nodes // TODO: can this be optimized?
	/* TODO: node optimization:
	 	Centers   [][]float64 // [NumHidden][numTotalInputsIncludingLastHiddenActivities]
		Spreads   []float64 // [NumHidden]
		Activities []float64 // [NumHidden]
		OutputWeights [][]float64 // [NumOutputs][NumHidden]
		Biases        []float64   // [NumOutputs]
	*/
	OutputWeights [][]float64 // [NumOutputs][NumHidden]
	Biases        []float64   // [NumOutputs]

	// Recurrent state
	PrevActivations []float64 // Previous hidden layer activations

	// Structural adaptation
	ErrorThreshold    float64
	DistanceThreshold float64
	ActivityDecay     float64
	PruneEnabled      bool
	PruneThreshold    float64

	// Statistics
	grownNodes  int
	prunedNodes int
	totalLoss   float64

	// Gain bounds (constrain network output)
	GainBounds [3]FloatBound
}

// NewRecurrentRBFNetwork creates a new RBF network
func NewRecurrentRBFNetwork(NumHiddenNodes, MaxHiddenNodes int, LearningRates LearningRates, minSpread float64, outputWeightsInitialMaxMagnitude float64) *RecurrentRBFNetwork {
	// Construct a dummy input to get the actual expected dimension
	// In this simplified version, we'll use 41 as specified in the original code
	numInputs := 41

	net := &RecurrentRBFNetwork{
		NumInputs:         numInputs,
		NumHidden:         NumHiddenNodes,
		MaxHidden:         MaxHiddenNodes,
		NumOutputs:        3,
		LearningRates:     LearningRates,
		Nodes:             make([]RBFNode, NumHiddenNodes),
		OutputWeights:     make([][]float64, 3),
		Biases:            make([]float64, 3),
		PrevActivations:   make([]float64, NumHiddenNodes),
		ErrorThreshold:    1.0,  // Grow when error exceeds 1°F // TODO: adding new Nodes like this will happen every time there is a setpoint change.... TODO: should we make this only grow when the erorr has been greater than a degree for 5 mins?
		DistanceThreshold: 0.5,  // Grow when input is 0.5 units away from all centers // TODO: const?
		ActivityDecay:     0.99, // TODO: const?
		PruneEnabled:      false,
		PruneThreshold:    0.01, // TODO: const?
		GainBounds: [3]FloatBound{
			{Min: 0.0, Max: 1.0}, // Kp bounds // TODO: consts?
			{Min: 0.0, Max: 0.5}, // Ki bounds // TODO: consts?
			{Min: 0.0, Max: 0.3}, // Kd bounds // TODO: consts?
		},
		MinSpread: minSpread,
	}

	// Initialize output weights
	weightInitializationModifier := outputWeightsInitialMaxMagnitude * 2
	for i := 0; i < 3; i++ {
		net.OutputWeights[i] = make([]float64, NumHiddenNodes)
		for j := 0; j < NumHiddenNodes; j++ {
			randNegToPosHalf := rand.Float64() - 0.5
			net.OutputWeights[i][j] = randNegToPosHalf * weightInitializationModifier // Small random initialization
		}
		net.Biases[i] = 0.0 // Start at midpoint of bounds
	}

	// Initialize RBF Nodes with correct dimensions
	// Each center needs: NumInputs + NumHidden dimensions (for recurrent inputs)
	centerDim := numInputs + NumHiddenNodes
	baseSigma := math.Sqrt(2.0 * float64(centerDim) / float64(NumHiddenNodes))
	for j := 0; j < NumHiddenNodes; j++ {
		center := make([]float64, centerDim)
		// External input dimensions: random in [-1, 1]
		for k := 0; k < numInputs; k++ {
			center[k] = (rand.Float64() * 2.0) - 1.0
		}
		// Recurrent dimensions: random in [0, 1]
		for k := 0; k < NumHiddenNodes; k++ {
			center[numInputs+k] = rand.Float64()
		}

		net.Nodes[j] = RBFNode{
			Center:   center,
			Spread:   baseSigma * (0.8 + rand.Float64()*0.4), // TODO: what is 0.8? 0.4?
			Activity: 1.0,
		}
	}

	return net
}

// TODO: use next line for importing pre-trained RRBFNNs
func NewRRBFNNFromJson(jsonBs []byte) (*RecurrentRBFNetwork, error) {
	out := &RecurrentRBFNetwork{}
	err := json.Unmarshal(jsonBs, out)
	out.grownNodes = 0
	out.prunedNodes = 0
	out.totalLoss = 0.0
	return out, err
}

// Forward computes network output given current inputs
func (net *RecurrentRBFNetwork) Forward(inputs []float64) RBFOutput {
	// Construct augmented input: [external_inputs, previous_recurrent_activations]
	augmented := make([]float64, len(inputs)+len(net.PrevActivations))
	copy(augmented, inputs)
	copy(augmented[len(inputs):], net.PrevActivations)

	if len(augmented) != len(net.Nodes[0].Center) {
		panic(fmt.Sprintf("augmented len %d doesn't match center len %d", len(augmented), len(net.Nodes[0].Center)))
	}

	// ========== Compute RBF activations ==========
	activations := make([]float64, net.NumHidden)
	for j := 0; j < net.NumHidden; j++ {
		// Sanity check
		if len(net.Nodes[j].Center) != len(augmented) {
			panic(fmt.Sprintf("node %d center len %d doesn't match augmented len %d", j, len(net.Nodes[j].Center), len(augmented)))
		}

		// Compute Gaussian basis function: phi_j = exp(-||z - c_j||^2 / (2*sigma_j^2))
		distSq := 0.0
		for k := 0; k < len(augmented); k++ {
			diff := augmented[k] - net.Nodes[j].Center[k]
			distSq += diff * diff
		}

		sigma := math.Max(net.Nodes[j].Spread, net.MinSpread*2) // Enforce minimum spread // TODO: why is this *2? should the other be /2 instead?
		sigma2 := sigma * sigma
		activations[j] = math.Exp(-distSq / (2.0 * sigma2))

		// Update activity tracking (EMA)
		net.Nodes[j].Activity = net.ActivityDecay*net.Nodes[j].Activity + (1.0-net.ActivityDecay)*activations[j]
	}

	// ========== Compute output layer ==========
	rawOutputs := make([]float64, 3)
	gains := make([]float64, 3)

	for i := 0; i < 3; i++ {
		sum := net.Biases[i]
		for j := 0; j < net.NumHidden; j++ {
			sum += net.OutputWeights[i][j] * activations[j]
		}
		rawOutputs[i] = sum

		// Apply sigmoid and scale to gain bounds
		s := sigmoid(sum)
		gains[i] = net.GainBounds[i].Min + (net.GainBounds[i].DeltaMinMax() * s)

		// Clamp to bounds
		net.GainBounds[i].Clamp(gains[i])
	}

	// Update recurrent state
	copy(net.PrevActivations, activations)

	return RBFOutput{
		Kp:                gains[0],
		Ki:                gains[1],
		Kd:                gains[2],
		PreSigmoidRaw:     rawOutputs,
		HiddenActivations: activations,
		AugmentedInputs:   augmented,
	}
}

// BackwardAndUpdate performs one step of gradient descent learning
func (net *RecurrentRBFNetwork) BackwardAndUpdate(error float64, phi, z []float64, rawOutputs []float64) {
	// Plant Jacobian: heater reduces positive temperature error
	dE_dU := -1.0

	// ========== Compute output layer gradients ==========
	deltas := make([]float64, 3)
	for i := 0; i < 3; i++ {
		s := sigmoid(rawOutputs[i])
		dK_dO := net.GainBounds[i].DeltaMinMax() * s * (1.0 - s) // TODO: how does this work?

		// Error signal scaled by Sigmoid derivative
		deltas[i] = error * dE_dU * dK_dO

		// Update bias
		net.Biases[i] -= net.LearningRates.Weights * deltas[i]
	}

	// ========== Update hidden layer parameters ==========
	for j := 0; j < net.NumHidden; j++ {
		// Backprop error signal to this node
		backErr := 0.0
		for i := 0; i < 3; i++ {
			backErr += deltas[i] * net.OutputWeights[i][j]

			// Update output weight
			net.OutputWeights[i][j] -= net.LearningRates.Weights * deltas[i] * phi[j]
		}

		// Compute distance and derivatives for center/spread updates
		sigma := math.Max(net.Nodes[j].Spread, 0.1)
		sigma2 := sigma * sigma
		sigma3 := sigma2 * sigma

		distSq := 0.0
		for k := 0; k < len(z); k++ {
			diff := z[k] - net.Nodes[j].Center[k]
			distSq += diff * diff

			// Update center
			dJ_dC := backErr * phi[j] * diff / sigma2
			net.Nodes[j].Center[k] -= net.LearningRates.Centers * dJ_dC
		}

		// Update spread
		dJ_dSigma := backErr * phi[j] * distSq / sigma3
		net.Nodes[j].Spread -= net.LearningRates.Spreads * dJ_dSigma

		// Enforce minimum spread
		if net.Nodes[j].Spread < net.MinSpread {
			net.Nodes[j].Spread = net.MinSpread
		}
	}

	// Accumulate loss
	net.totalLoss += error * error
}

// AdaptStructure grows/prunes Nodes based on error and input novelty
func (net *RecurrentRBFNetwork) AdaptStructure(inputs []float64, error float64) {
	if net.NumHidden >= net.MaxHidden {
		return
	}

	// Check growth criterion: high error in novel input region
	if math.Abs(error) > net.ErrorThreshold { // TODO: should this happen only if the error stays high for a certain period? (not just a setpoint change?)
		z := make([]float64, len(inputs)+len(net.PrevActivations))
		copy(z, inputs)
		copy(z[len(inputs):], net.PrevActivations)

		// Find minimum distance to existing centers
		minDist := math.MaxFloat64
		for j := 0; j < net.NumHidden; j++ {
			dist := 0.0
			for k := 0; k < len(z); k++ {
				diff := z[k] - net.Nodes[j].Center[k]
				dist += diff * diff
			}
			dist = math.Sqrt(dist)
			if dist < minDist {
				minDist = dist
			}
		}

		// Grow if sufficiently novel
		if minDist > net.DistanceThreshold {
			net.growNode(z)
		}
	}

	// Prune inactive Nodes // TODO: do we even want pruning in cases where something might only happen once per year?
	if net.PruneEnabled {
		for j := net.NumHidden - 1; j >= 0; j-- {
			if net.Nodes[j].Activity < net.PruneThreshold && net.NumHidden > 1 {
				net.removeNode(j)
			}
		}
	}
}

// growNode adds a new RBF center
func (net *RecurrentRBFNetwork) growNode(newNodeCenterWithoutRecurrentDim []float64) {
	// IMPORTANT: When growing a node, we're adding a new recurrent dimension.
	// The new node's activation becomes a new input for future timesteps.
	// So we must:
	// 1. Expand all existing centers by 1 dimension (for new node's recurrent coord)
	// 2. Create the new center with full augmented dimensionality
	// 3. Expand PrevActivations to match

	newCenter := make([]float64, len(newNodeCenterWithoutRecurrentDim)+1) // +1 for the new node's own recurrent dim
	copy(newCenter, newNodeCenterWithoutRecurrentDim)
	newCenter[len(newNodeCenterWithoutRecurrentDim)] = 1.0 // New node's own activation coordinate

	// Expand all existing centers by one dimension (for the new node's recurrent activation)
	newDimValue := 0.5 // Neutral initialization for existing Nodes' view of new recurrent dim
	for j := range net.Nodes {
		net.Nodes[j].Center = append(net.Nodes[j].Center, newDimValue)
	}

	// Set spread based on minimum distance to existing centers
	baseSigma := math.Sqrt(2.0 * float64(len(newCenter)) / float64(net.NumHidden+1))
	newSpread := baseSigma

	if net.NumHidden > 0 {
		minDist := math.MaxFloat64
		for j := 0; j < net.NumHidden; j++ {
			dist := 0.0
			// Compare only up to current newNodeCenterWithoutRecurrentDim dimensionality
			for k := 0; k < len(newNodeCenterWithoutRecurrentDim); k++ {
				diff := newNodeCenterWithoutRecurrentDim[k] - net.Nodes[j].Center[k]
				dist += diff * diff
			}
			dist = math.Sqrt(dist)
			if dist < minDist {
				minDist = dist
			}
		}
		newSpread = math.Max(minDist*0.7, 0.3*baseSigma) // TODO: where do 0.7 and 0.3 come from?
	}

	net.Nodes = append(net.Nodes, RBFNode{
		Center:   newCenter,
		Spread:   newSpread,
		Activity: 1.0,
	})

	// Expand PrevActivations to include the new node's activation
	net.PrevActivations = append(net.PrevActivations, 1.0)

	// Add new output weights
	for i := 0; i < 3; i++ {
		net.OutputWeights[i] = append(net.OutputWeights[i], 0.0) // Start neutral
	}

	net.NumHidden++
	net.grownNodes++
}

// removeNode deletes a hidden node
func (net *RecurrentRBFNetwork) removeNode(j int) {
	net.Nodes = append(net.Nodes[:j], net.Nodes[j+1:]...)
	net.PrevActivations = append(net.PrevActivations[:j], net.PrevActivations[j+1:]...)

	for i := 0; i < 3; i++ {
		net.OutputWeights[i] = append(net.OutputWeights[i][:j], net.OutputWeights[i][j+1:]...)
	}

	net.NumHidden--
	net.prunedNodes++
}

// GetNumNodes returns the current number of hidden Nodes
func (net *RecurrentRBFNetwork) GetNumNodes() int {
	return net.NumHidden
}

// GetTotalLoss returns accumulated loss
func (net *RecurrentRBFNetwork) GetTotalLoss() float64 {
	return net.totalLoss
}

// sigmoid is the logistic activation function
func sigmoid(x float64) float64 {
	// Clamp to prevent overflow
	if x > 100 {
		return 1.0
	}
	if x < -100 {
		return 0.0
	}
	return 1.0 / (1.0 + math.Exp(-x))
}

// Reset clears recurrent memory
func (net *RecurrentRBFNetwork) Reset() {
	for i := range net.PrevActivations {
		net.PrevActivations[i] = 0.0
	}
}

// SetGrowthEnabled enables or disables network node growth
func (net *RecurrentRBFNetwork) SetGrowthEnabled(enabled bool) {
	// Growth is controlled by checking NumHidden >= MaxHidden
	// To disable growth, we can effectively freeze it by setting MaxHidden = NumHidden
	// To enable growth, we restore MaxHidden to the configured value
	if !enabled {
		net.MaxHidden = net.NumHidden
	}
}

// SetPruningEnabled enables or disables network node pruning
func (net *RecurrentRBFNetwork) SetPruningEnabled(enabled bool) {
	net.PruneEnabled = enabled
}
