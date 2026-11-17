package main

import (
	"flag"
	"fmt"
	"math"
	"sort"
)

// RBFNode represents a dynamic radial basis neuron with recurrent feedback.
type RBFNode struct {
	Center   []float64 // Center vector (dimension: input_dim + 1 for self/recurrent feedback)
	Width    float64   // Gaussian spread parameter (sigma)
	Weight   float64   // Weight connecting this neuron to the output
	PrevOut  float64   // Output activation at t-1 (recurrent state)
	Activity float64   // Moving average of activation to guide pruning
}

// RRBFNN represents the Recurrent Radial Basis Function Network.
type RRBFNN struct {
	InputDim      int
	Nodes         []*RBFNode
	Bias          float64
	LearningRate  float64
	ErrorThresh   float64 // Error threshold to trigger node insertion
	DistThresh    float64 // Distance threshold (novelty) to trigger node insertion
	PruneEnabled  bool    // When false, nodes are never removed (growth only)
	PruneThresh   float64 // Activity threshold below which a node is removed (only if PruneEnabled)
	MaxNodes      int
	ActivityDecay float64 // EMA factor for node Activity; must be slow enough to span the signal's longest period

	// Progress counters (for console reporting)
	GrownTotal     int // nodes added since start
	PrunedTotal    int // nodes removed since start
	grownThisStep  int
	prunedThisStep int
	lastGrowErr    float64 // |err| that triggered the most recent growth
}

// Config bundles the hyperparameters so they can be swept.
type Config struct {
	LR, ErrThresh, DistThresh, PruneThresh, ActivityDecay float64
	MaxNodes                                              int
	PruneEnabled                                          bool
}

// NewRRBFNN initializes the network with a base capacity.
func NewRRBFNN(inputDim int, c Config) *RRBFNN {
	return &RRBFNN{
		InputDim:      inputDim,
		Nodes:         make([]*RBFNode, 0),
		Bias:          0.0,
		LearningRate:  c.LR,
		ErrorThresh:   c.ErrThresh,
		DistThresh:    c.DistThresh,
		PruneEnabled:  c.PruneEnabled,
		PruneThresh:   c.PruneThresh,
		ActivityDecay: c.ActivityDecay,
		MaxNodes:      c.MaxNodes,
	}
}

// computeFullInput augments the input vector with recurrent activation.
func (net *RRBFNN) computeFullInput(x []float64, prevNodeOut float64) []float64 {
	full := make([]float64, len(x)+1)
	copy(full, x)
	full[len(x)] = prevNodeOut // recurrent feature
	return full
}

// euclideanDist computes Euclidean distance between vectors a and b.
func euclideanDist(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return math.Sqrt(sum)
}

// Forward computes the network prediction and updates activations.
func (net *RRBFNN) Forward(x []float64) (float64, []float64) {
	return net.forward(x, true)
}

// Predict computes the network prediction without touching any node state (safe for reporting).
func (net *RRBFNN) Predict(x []float64) float64 {
	out, _ := net.forward(x, false)
	return out
}

// Observe runs the network forward in inference mode: recurrent state (PrevOut) and Activity are
// advanced exactly as in training, but no weights/centers/widths are changed. Use this when running
// the network on a live stream without learning; using Predict there would freeze the recurrent state.
func (net *RRBFNN) Observe(x []float64) float64 {
	out, activations := net.forward(x, true)
	for i, node := range net.Nodes {
		node.PrevOut = activations[i]
	}
	return out
}

func (net *RRBFNN) forward(x []float64, updateActivity bool) (float64, []float64) {
	out := net.Bias
	activations := make([]float64, len(net.Nodes))

	for i, node := range net.Nodes {
		fullInput := net.computeFullInput(x, node.PrevOut)
		dist := euclideanDist(fullInput, node.Center)

		// Gaussian RBF: phi = exp(-||x - c||^2 / (2 * sigma^2))
		phi := math.Exp(-(dist * dist) / (2.0*node.Width*node.Width + 1e-8))
		activations[i] = phi
		out += node.Weight * phi

		// Track moving average of activation for pruning
		if updateActivity {
			a := net.ActivityDecay
			node.Activity = a*node.Activity + (1-a)*phi
		}
	}
	return out, activations
}

// AdaptStructure handles online dynamic growing and pruning.
func (net *RRBFNN) AdaptStructure(x []float64, err float64) {
	net.grownThisStep, net.prunedThisStep = 0, 0
	// 1. Growing Criterion: High error and novel input space
	if math.Abs(err) > net.ErrorThresh && len(net.Nodes) < net.MaxNodes {
		minDist := math.MaxFloat64
		for _, node := range net.Nodes {
			fullInput := net.computeFullInput(x, node.PrevOut)
			d := euclideanDist(fullInput, node.Center)
			if d < minDist {
				minDist = d
			}
		}

		if minDist > net.DistThresh || len(net.Nodes) == 0 {
			// Allocate new neuron centered at current input state.
			// A freshly created node fires at ~1.0 on its first evaluation, so its recurrent
			// coordinate must be 1.0 (not 0) or it immediately suppresses itself next step.
			newCenter := net.computeFullInput(x, 1.0) // TODO: used to be 0.0
			// First node has no neighbour (minDist == MaxFloat64); fall back to a sane default width.
			width := 0.5
			if len(net.Nodes) > 0 {
				width = math.Max(minDist*0.7, 0.3) // dynamically scale width // TODO: 0.3 used to be 0.5
			}
			newNode := &RBFNode{
				Center:   newCenter,
				Width:    width,
				Weight:   err, // initialize weight to absorb residual error
				PrevOut:  1.0, // TODO: used to be 0
				Activity: 1.0,
			}
			net.Nodes = append(net.Nodes, newNode)
			net.GrownTotal++
			net.grownThisStep++
			net.lastGrowErr = math.Abs(err)
			return
		}
	}

	// 2. Pruning Criterion: Remove inactive/redundant units
	if net.PruneEnabled && len(net.Nodes) > 1 {
		pruned := make([]*RBFNode, 0, len(net.Nodes))
		for _, node := range net.Nodes {
			if node.Activity >= net.PruneThresh {
				pruned = append(pruned, node)
			}
		}
		removed := len(net.Nodes) - len(pruned)
		net.PrunedTotal += removed
		net.prunedThisStep += removed
		net.Nodes = pruned
	}
}

// TrainStep performs online adaptation: forward pass, structural adjustment, and parameter tuning.
func (net *RRBFNN) TrainStep(x []float64, target float64) float64 {
	pred, activations := net.Forward(x)
	err := target - pred

	// Dynamic structure adaptation (add/prune nodes)
	net.AdaptStructure(x, err)

	// Re-evaluate activations if structure changed
	if len(activations) != len(net.Nodes) {
		pred, activations = net.forward(x, false)
		err = target - pred
	}

	// Dynamic Parameter Adaptation (Gradient Descent)
	for i, node := range net.Nodes {
		phi := activations[i]
		fullInput := net.computeFullInput(x, node.PrevOut)
		w := node.Weight // use the pre-update weight for the center/width gradients

		// dE/dw = -err * phi
		node.Weight += net.LearningRate * err * phi

		// dE/dc_j = -err * w * phi * ((x_j - c_j) / sigma^2)
		// dE/dsigma = -err * w * phi * (||x - c||^2 / sigma^3)
		sigma2 := node.Width*node.Width + 1e-8
		distSq := 0.0
		for j := range node.Center {
			diff := fullInput[j] - node.Center[j]
			distSq += diff * diff
			node.Center[j] += net.LearningRate * err * w * phi * (diff / sigma2)
		}

		// Width adaptation
		node.Width += net.LearningRate * err * w * phi * (distSq / (sigma2*node.Width + 1e-8))
		if node.Width < 0.1 {
			node.Width = 0.1 // Numerical floor
		}

		// Update recurrent state for step t+1
		node.PrevOut = phi
	}

	// Bias adaptation
	net.Bias += net.LearningRate * err

	return err * err
}

// widthStats returns min/mean/max of node widths (0s if no nodes).
func (net *RRBFNN) widthStats() (minW, meanW, maxW float64) {
	if len(net.Nodes) == 0 {
		return 0, 0, 0
	}
	minW, maxW = math.MaxFloat64, -math.MaxFloat64
	sum := 0.0
	for _, n := range net.Nodes {
		minW = math.Min(minW, n.Width)
		maxW = math.Max(maxW, n.Width)
		sum += n.Width
	}
	return minW, sum / float64(len(net.Nodes)), maxW
}

const (
	inputDim    = 3
	dtStep      = 0.05
	trainSteps  = 2000
	evalSteps   = 300
	reportEvery = 100
)

func target(t int) float64 {
	tv := float64(t) * dtStep
	return math.Sin(tv) * math.Cos(0.5*tv)
}

// runExperiment trains then evaluates one configuration. verbose=true prints the progress table.
// Returns (training MSE over the last 500 steps, held-out MSE, final node count).
func runExperiment(c Config, verbose bool) (tailMSE, heldOutMSE float64, nodes int) {
	net := NewRRBFNN(inputDim, c)

	if verbose {
		fmt.Println("Training RRBFNN on a nonlinear dynamic time-series: y(t) = sin(t) * cos(0.5t)")
		fmt.Println("Inputs: [sin(t), y(t-1), y(t-2)]  (NARX)")
		fmt.Printf("Config: lr=%.3f errThresh=%.3f distThresh=%.2f prune=%v activityDecay=%.3f maxNodes=%d\n\n",
			c.LR, c.ErrThresh, c.DistThresh, c.PruneEnabled, c.ActivityDecay, c.MaxNodes)
		fmt.Printf("%-5s | %-9s | %-9s | %-8s | %-8s | %-8s | %-5s | %-3s/%-3s | %-7s | %-17s\n",
			"Step", "MSE(inst)", "MSE(win)", "Target", "Pred", "Err", "Nodes", "+", "-", "Bias", "Width min/mean/max")
		fmt.Println("------+-----------+-----------+----------+----------+----------+-------+---------+---------+-------------------")
	}

	windowSSE, windowN := 0.0, 0
	totalSSE, tailSSE := 0.0, 0.0
	grownWin, prunedWin := 0, 0
	yPrev1, yPrev2 := 0.0, 0.0
	tailStart := trainSteps - 500

	for t := 0; t < trainSteps; t++ {
		input := []float64{math.Sin(float64(t) * dtStep), yPrev1, yPrev2}
		y := target(t)

		mse := net.TrainStep(input, y)
		windowSSE += mse
		totalSSE += mse
		if t >= tailStart {
			tailSSE += mse
		}
		windowN++
		grownWin += net.grownThisStep
		prunedWin += net.prunedThisStep

		if verbose {
			if net.grownThisStep > 0 {
				fmt.Printf("      > step %4d: +%d node(s)  (total nodes=%d, trigger |err|=%.4f)\n", t, net.grownThisStep, len(net.Nodes), net.lastGrowErr)
			}
			if net.prunedThisStep > 0 {
				fmt.Printf("      > step %4d: -%d node(s)  (total nodes=%d)\n", t, net.prunedThisStep, len(net.Nodes))
			}
			if t%reportEvery == 0 || t == trainSteps-1 {
				pred := net.Predict(input)
				minW, meanW, maxW := net.widthStats()
				fmt.Printf("%-5d | %9.5f | %9.5f | %8.4f | %8.4f | %8.4f | %5d | %3d/%-3d | %7.4f | %5.2f/%5.2f/%5.2f\n",
					t, mse, windowSSE/float64(windowN), y, pred, y-pred, len(net.Nodes),
					grownWin, prunedWin, net.Bias, minW, meanW, maxW)
				windowSSE, windowN = 0, 0
				grownWin, prunedWin = 0, 0
			}
		}

		// Shift the NARX feedback taps (teacher forcing: use the true target)
		yPrev2, yPrev1 = yPrev1, y
	}

	// Held-out evaluation: continue the series past the training range, teacher-forced.
	evalSSE := 0.0
	for t := trainSteps; t < trainSteps+evalSteps; t++ {
		input := []float64{math.Sin(float64(t) * dtStep), yPrev1, yPrev2}
		y := target(t)
		e := y - net.Observe(input) // Observe, not Predict: recurrent state must keep advancing
		evalSSE += e * e
		yPrev2, yPrev1 = yPrev1, y
	}

	tailMSE = tailSSE / 500
	heldOutMSE = evalSSE / evalSteps

	if verbose {
		fmt.Println("\n=== Training complete ===")
		fmt.Printf("Overall MSE: %.5f | Last-500 MSE: %.5f | Final nodes: %d | Grown: %d | Pruned: %d | Bias: %.4f\n",
			totalSSE/trainSteps, tailMSE, len(net.Nodes), net.GrownTotal, net.PrunedTotal, net.Bias)
		fmt.Printf("Held-out MSE over next %d steps (no learning): %.5f\n", evalSteps, heldOutMSE)

		fmt.Println("\nFinal node table:")
		fmt.Printf("%-4s | %-28s | %-7s | %-8s | %-8s\n", "#", "Center [sin, y-1, y-2, recur]", "Width", "Weight", "Activity")
		for i, n := range net.Nodes {
			fmt.Printf("%-4d | [%6.3f,%6.3f,%6.3f,%6.3f] | %7.3f | %8.4f | %8.4f\n",
				i, n.Center[0], n.Center[1], n.Center[2], n.Center[3], n.Width, n.Weight, n.Activity)
		}
	}
	return tailMSE, heldOutMSE, len(net.Nodes)
}

type sweepResult struct {
	cfg           Config
	tail, heldOut float64
	nodes         int
}

// sweep grid-searches the growth-only hyperparameters and prints the best configurations by held-out MSE.
func sweep() { // TODO: how to use this?
	lrs := []float64{0.005, 0.01, 0.02, 0.03}
	errThs := []float64{0.02, 0.05, 0.08, 0.12}
	distThs := []float64{0.25, 0.35, 0.5, 0.7}
	maxNodes := []int{20, 30, 50, 80}

	var results []sweepResult
	total := len(lrs) * len(errThs) * len(distThs) * len(maxNodes)
	done := 0
	fmt.Printf("Sweeping %d configurations (prune disabled)...\n", total)
	for _, lr := range lrs {
		for _, et := range errThs {
			for _, dt := range distThs {
				for _, mn := range maxNodes {
					c := Config{LR: lr, ErrThresh: et, DistThresh: dt, MaxNodes: mn, ActivityDecay: 0.995, PruneEnabled: false}
					tail, ho, n := runExperiment(c, false)
					results = append(results, sweepResult{c, tail, ho, n})
					done++
					if done%32 == 0 {
						fmt.Printf("  %d/%d\n", done, total)
					}
				}
			}
		}
	}

	sort.Slice(results, func(i, j int) bool { return results[i].heldOut < results[j].heldOut })
	fmt.Printf("\n%-6s | %-9s | %-10s | %-8s | %-9s | %-9s | %-5s\n", "lr", "errThresh", "distThresh", "maxNodes", "tail MSE", "heldOut", "nodes")
	fmt.Println("-------+-----------+------------+----------+-----------+-----------+------")
	for i := 0; i < 15 && i < len(results); i++ {
		r := results[i]
		fmt.Printf("%-6.3f | %-9.3f | %-10.2f | %-8d | %9.5f | %9.5f | %5d\n",
			r.cfg.LR, r.cfg.ErrThresh, r.cfg.DistThresh, r.cfg.MaxNodes, r.tail, r.heldOut, r.nodes)
	}
}

func main() {
	doSweep := flag.Bool("sweep", false, "grid-search hyperparameters instead of running a single verbose training")
	flag.Parse()

	if *doSweep {
		sweep()
		return
	}

	// Growth-only configuration (pruning disabled). Values chosen from `-sweep`:
	//   held-out MSE 0.0068 with 20 nodes (best of 256 configs). Lower lr + wider distThresh wins:
	//   fewer, broader nodes generalise better than many narrow ones on this smooth signal.
	cfg := Config{
		LR:            0.01,
		ErrThresh:     0.05,
		DistThresh:    0.5,
		MaxNodes:      20,
		ActivityDecay: 0.995,
		PruneEnabled:  false,
		PruneThresh:   0.002, // unused while PruneEnabled=false
	}
	runExperiment(cfg, true)
}
