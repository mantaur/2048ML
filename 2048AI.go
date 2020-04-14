package main

import (
	"fmt"
	"math/rand"
	"ml/2048ML/grid"
	"sort"
	"sync"
	"time"
)

// Create a board struct to represent 2048 board and hold values in each tile
type board struct {
}

type neuralNet struct {
	Layers    []*layer
	MovesLeft int
	Score     int
}

//A layer holds multiple nodes stacked and their connections to the previous layer
type layer struct {
	Nodes []node //maybe []*node
	Type  int    //0 for input, 1 for hidden and 2 for output

	// Nodes []*grid.Cell
	//Connections *layer //recursively define the connections
}

//Lowest level of neural net, individual nodes. Each holds a value, bias and a list of weights
type node struct {
	Value   float32
	Bias    float32
	Weights []float32
}

//byScore implements sort.Interface for []neuralNet based on Score field
type byScore []*neuralNet

func (a byScore) Len() int           { return len(a) }
func (a byScore) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byScore) Less(i, j int) bool { return a[i].Score > a[j].Score }

func main() {
	fmt.Println("Starting...")

	// popSize := 60000
	popSize := 30000
	mutationRate := 0.05 //5%
	generations := 30

	population := make([]*neuralNet, popSize)
	boards := make([]*grid.Grid, popSize)

	for i := 0; i < popSize; i++ {
		population[i] = createNeuralNet(16, 1, 4)
		boards[i] = createGrid(4, 2, 0, 4096)
	}

	evolve(population, boards, generations, mutationRate)

}

func evolve(population []*neuralNet, boards []*grid.Grid, generations int, mutationR float64) {
	for g := 0; g < generations; g++ {
		size := len(population)
		wg := new(sync.WaitGroup)
		wg.Add(size)
		for i := 0; i < size; i++ {
			go population[i].Play(boards[i], wg)
			// population[i].Play(boards[i]) // Leaving here becuase debugging mutlithreaded is hard
		}
		wg.Wait()
		sort.Sort(byScore(population))
		// fmt.Println("Generation:", g, ".   Best score:", population[0].Score, ".   Median:", population[size/2].Score)
		fmt.Printf("Generation: %3d, best score: %5d, median: %5d\n", g, population[0].Score, population[size/2].Score)
		if population[size/2].Score < 2000 {
			mutationR = 0.2
		} else if population[size/2].Score > 2200 {
			mutationR = 0.05
		}

		for i := 0; i < (size*5)/6; i++ {
			population[size-i-1] = reproduce(population[0], population[i], mutationR)
		}
	}
}

//reproduce creates a new neuralNet from two supplied
func reproduce(nn1, nn2 *neuralNet, mutationR float64) *neuralNet {
	nrLays := len(nn1.Layers)
	nnb := neuralNet{
		MovesLeft: 3,
		Score:     0,
		Layers:    make([]*layer, nrLays),
	}

	inputSize := len(nn1.Layers[0].Nodes)

	inputLayer := layer{
		Type: 0,
	}
	inputLayer.Nodes = make([]node, inputSize)

	hiddenLayers := make([]layer, nrLays-2)
	for i := 0; i < nrLays-2; i++ {
		if i == 0 {
			hiddenLayers[i].Type = 1
			hiddenLayers[i].Nodes = make([]node, 8)

			mergeBias(&hiddenLayers[i], nn1.Layers[1+i], nn2.Layers[1+i])

			mergeWeights(&hiddenLayers[i], nn1.Layers[1+i], nn2.Layers[1+i])
		} else {
			hiddenLayers[i].Type = 1
			hiddenLayers[i].Nodes = make([]node, 8)

			mergeBias(&hiddenLayers[i], nn1.Layers[1+i], nn2.Layers[1+i])

			mergeWeights(&hiddenLayers[i], nn1.Layers[1+i], nn2.Layers[1+i])
		}
	}

	outputLayer := layer{
		Type:  2,
		Nodes: make([]node, len(nn1.Layers[nrLays-1].Nodes)),
	}

	mergeBias(&outputLayer, nn1.Layers[nrLays-1], nn2.Layers[nrLays-1])

	//init weigths of output layer after appending all layers
	nnb.Layers[0] = &inputLayer
	// append(nn.Layers, &inputLayer)
	for i, hidLay := range hiddenLayers {
		nnb.Layers[1+i] = &hidLay
	}
	mergeWeights(&outputLayer, nn1.Layers[nrLays-1], nn2.Layers[nrLays-1])
	nnb.Layers[nrLays-1] = &outputLayer

	nnb.mutate(mutationR)

	return &nnb
}

func (nn *neuralNet) mutate(mutationR float64) {
	chance := rand.Float64()
	if chance < mutationR {
		for l := 1; l < len(nn.Layers); l++ { //for every non input layer
			for n := 0; n < len(nn.Layers[l].Nodes); n++ { //for every node
				for w := 0; w < len(nn.Layers[l].Nodes[n].Weights); w++ { //for every weight
					mutAmount := rand.Float32()*3 - 1.5 //how much to alter [-1.5 , 1.5)
					nn.Layers[l].Nodes[n].Weights[w] *= mutAmount
				}
			}
		}
	}
}

//Play let's the neural network play till finished and records it's score
//Chooses between the nn's top three moves if up to two moves have no effect
func (nn *neuralNet) Play(g *grid.Grid, wg *sync.WaitGroup) {
	g.Reset()
	g.Build()
	nn.Score = g.Score //New game, reset nn's score

	for !g.GameOver && nn.MovesLeft > 0 { //keep playing while game isn't over and have moves left

		nn.scanInput(g)      //Look
		nn.feedForward()     //Think
		g.Shift(nn.output()) //Act

		nn.MovesLeft--
		if nn.Score == g.Score {
			g.Shift(nn.sortedOut()[1]) //nothing happened?, try second best move!
		}
		if nn.Score == g.Score {
			g.Shift(nn.sortedOut()[2]) //still nothing?, try third and last resort move before deadlock and game over!
		}

		if nn.Score != g.Score { //if a move is done and score affected, reset MovesLeft
			nn.MovesLeft = 3
		}
		nn.Score = g.Score
	}
	if nn.Score > 15000 {
		printGrid(g)
	}
	wg.Done()
}

func createGrid(size, start, score, max int) *grid.Grid {
	g := grid.Grid{
		Size:       size,
		StartCells: start,
		Score:      score,
		MaxScore:   max,
		GameOver:   false,
	}
	g.Build()

	return &g
}

//Creates a neural net with random biases and weights between layers, and with input layer
//of size: ´inputSize´, ´hidden´ number of hidden layers and ´output´ outputs
func createNeuralNet(inputSize, hidden, output int) *neuralNet {
	nn := neuralNet{}
	nn.MovesLeft = 3
	nn.Score = 10
	nn.Layers = make([]*layer, 1+hidden+1)
	inputLayer := layer{
		Type: 0,
	}
	inputLayer.Nodes = make([]node, inputSize)

	// initBias(&inputLayer, inputSize

	hiddenLayers := make([]layer, hidden)
	for i := 0; i < hidden; i++ {
		if i == 0 {
			hiddenLayers[i].Type = 1
			hiddenLayers[i].Nodes = make([]node, 8)

			initBias(&hiddenLayers[i], inputSize)

			for j := range hiddenLayers[i].Nodes {
				length := len(inputLayer.Nodes)
				hiddenLayers[i].Nodes[j].Weights = make([]float32, length)
				for w := 0; w < length; w++ {
					hiddenLayers[i].Nodes[j].Weights[w] = rand.Float32()*2 - 1 //create random weights for each node in each hidden layer
				}
			}
		} else {
			hiddenLayers[i].Type = 1
			hiddenLayers[i].Nodes = make([]node, 8)

			initBias(&hiddenLayers[i], inputSize)

			for j := range hiddenLayers[i].Nodes {
				length := len(hiddenLayers[i-1].Nodes)
				hiddenLayers[i].Nodes[j].Weights = make([]float32, length)
				for w := 0; w < length; w++ {
					hiddenLayers[i].Nodes[j].Weights[w] = rand.Float32()*2 - 1 //create random weights for each node in each hidden layer
				}
			}
		}
	}

	outputLayer := layer{
		Type:  2,
		Nodes: make([]node, output),
	}

	initBias(&outputLayer, inputSize)
	//init weigths of output layer after appending all layers
	nn.Layers[0] = &inputLayer
	// append(nn.Layers, &inputLayer)
	for i, _ := range hiddenLayers {
		// nn.Layers[1+i] = &hidLay
		nn.Layers[1+i] = &hiddenLayers[i]
	}
	nn.Layers[hidden+1] = &outputLayer

	length := len(nn.Layers[hidden].Nodes)
	for i := range nn.Layers[hidden+1].Nodes {
		nn.Layers[hidden+1].Nodes[i].Weights = make([]float32, length)
		for w := 0; w < length; w++ {
			nn.Layers[hidden+1].Nodes[i].Weights[w] = rand.Float32() //create random weights for each node in output layer
		}
	}

	return &nn
}

func initBias(nnl *layer, inputSize int) {
	length := len(nnl.Nodes)
	for i := 0; i < length; i++ {
		nnl.Nodes[i].Bias = rand.Float32() * float32(inputSize)
	}
}

func mergeBias(nnlb, nnl1, nnl2 *layer) {
	length := len(nnl1.Nodes)
	for i := 0; i < length; i++ {
		nnlb.Nodes[i].Bias = (nnl1.Nodes[i].Bias + nnl2.Nodes[i].Bias) / 2
	}
}

func mergeWeights(nnlb, nnl1, nnl2 *layer) {
	length := len(nnl1.Nodes)
	for i := 0; i < length; i++ {
		nnlb.Nodes[i].Weights = make([]float32, len(nnl1.Nodes[i].Weights))
		for w := 0; w < len(nnl1.Nodes[i].Weights); w++ {
			nnlb.Nodes[i].Weights[w] = (nnl1.Nodes[i].Weights[w] + nnl2.Nodes[i].Weights[w]) / 2
		}
	}
}

func (nn *neuralNet) scanInput(g *grid.Grid) {
	for x := 0; x < g.Size; x++ {
		for y := 0; y < g.Size; y++ {
			if g.Cells[x][y].Tile != nil {
				nn.Layers[0].Nodes[x+y*4].Value = float32(g.Cells[x][y].Tile.Value) - nn.Layers[0].Nodes[x+y*4].Bias
			}
		}
	}
}

func (nn *neuralNet) feedForward() { //TODO REVISIT RANGE AND USE i, "val" := range
	for i := 0; i < len(nn.Layers); i++ {
		//for i := range nn.Layers { //for every layer
		if nn.Layers[i].Type != 0 { //Feedforward when current layer isnt an input layer
			for j := 0; j < len(nn.Layers[i].Nodes); j++ { //for every node in this layer
				for k := 0; k < len(nn.Layers[i-1].Nodes); k++ { //for every node in previous layer for this node in this layer
					nn.Layers[i].Nodes[j].Value += nn.Layers[i-1].Nodes[k].Value * nn.Layers[i].Nodes[j].Weights[k]
				}
				nn.Layers[i].Nodes[j].Value -= nn.Layers[i].Nodes[j].Bias //Remove bias from sum
				//Squishify value with ReLU MAYBE TODO
			}
		}
	}
}

func (nn *neuralNet) output() int {
	end := len(nn.Layers) - 1
	highest := float32(nn.Layers[end].Nodes[0].Value)
	direction := 0
	for i := 1; i < 4; i++ {
		if highest < float32(nn.Layers[end].Nodes[i].Value) {
			highest = float32(nn.Layers[end].Nodes[i].Value)
			direction = i
		}
	}
	return direction
}

type dir struct {
	Value   float32
	moveDir int
}

func (nn *neuralNet) sortedOut() []int {
	end := len(nn.Layers) - 1
	outputSize := len(nn.Layers[end].Nodes)

	outPutDir := make([]dir, outputSize)
	res := make([]int, outputSize)

	for i := 0; i < outputSize; i++ {
		outPutDir[i].Value = nn.Layers[end].Nodes[i].Value
		outPutDir[i].moveDir = i
	}

	sort.Slice(outPutDir, func(i, j int) bool {
		return outPutDir[i].Value > outPutDir[j].Value
	})

	for i := 0; i < outputSize; i++ {
		res[i] = outPutDir[i].moveDir
	}

	return res
}

func printGrid(g *grid.Grid) {
	fmt.Println()
	for y := g.Size - 1; y >= 0; y-- {
		for x := 0; x < g.Size; x++ {
			if g.Cells[x][y].Tile != nil {
				fmt.Printf("[ ")
				fmt.Printf("%4d ", g.Cells[x][y].Tile.Value)
				fmt.Printf("]")
			} else {
				fmt.Printf("[   ]")
			}
		}
		fmt.Println()
	}
	fmt.Println()
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
