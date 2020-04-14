package grid

import (
	"fmt"
	"math/rand"
	"time"
)

var idSeed uint = 0

//Position structure
type pos struct {
	X int
	Y int
}

//Tile tracks value and position on the grid
type Tile struct {
	ID           uint
	Value        int
	MergeHistory TileList
	Current      pos
	Prev         pos
	New          bool
	Merged       bool
}

//Move a tile to destination cell
func (t *Tile) Move(dest *Cell) {
	t.Prev = t.Current
	t.Current = dest.Pos
}

//Merge tile tn to this tile and add tile tn's mergehistory to this tile's history
func (t *Tile) Merge(tn *Tile) {
	t.Merged = true
	t.Value += tn.Value //Future proofs for other merge rules. Fibonacci game??? Yes please. TODO
	t.MergeHistory = append(t.MergeHistory, tn)
}

func randVal() *rand.Rand {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	return r
}

//randTileVal returns value of new tile = 2 90% of times, otherwise 4
func randTileVal() int {
	// if randVal().Float64() > 0.9 {
	// 	return 4
	// }
	if rand.Float64() > 0.9 {
		return 4
	}
	return 2
}

//TileList holds a list of Tile s...
type TileList []*Tile

func (t TileList) remove(tr *Tile) TileList {
	for i, tl := range t {
		if tl == tr {
			return append(t[:i], t[i+1:]...)
		}
	}
	return t
}

//Grid tracks the Tiles
type Grid struct {
	Size       int
	MaxScore   int
	StartCells int
	Tiles      TileList
	Cells      [][]*Cell
	Score      int
	GameOver   bool
}

//Cell extends Tile and Pos
type Cell struct {
	Tile *Tile
	Pos  pos
}

//Build creates the grid
func (g *Grid) Build() {
	g.Cells = make([][]*Cell, g.Size)
	g.Tiles = make(TileList, 0)

	for i := range g.Cells {
		g.Cells[i] = make([]*Cell, g.Size)
		for j := range g.Cells[i] {
			cell := new(Cell)

			cell.Pos = pos{
				X: i,
				Y: j,
			}

			g.Cells[i][j] = cell
		}
	}

	// fill the grid with StartCells # of random new tiles
	start := g.StartCells

	for start != 0 {
		start = start - 1
		g.newTile()
	}
}

//Reset s the board
func (g *Grid) Reset() {
	for i := range g.Cells {
		for j := range g.Cells[i] {
			g.Cells[i][j].Tile = nil //memory management hehe...
		}
	}
	g.Score = 0
	g.Tiles = make(TileList, 0)
	g.GameOver = false
}

// create a new tile in a random available/empty cell
func (g *Grid) newTile() {

	avail := g.EmptyCells()

	if len(avail) == 0 {
		return
	}
	//Two random values between 0 and Grid.Size
	// i := randVal().Int() % len(avail)
	i := rand.Int() % len(avail)
	cell := avail[i]

	id := idSeed //Will concurrency screw with this?
	idSeed++

	newTile := Tile{
		ID:           id,
		Value:        randTileVal(),
		MergeHistory: make(TileList, 0),
		Current:      cell.Pos,
		New:          true,
	}

	g.Tiles = append(g.Tiles, &newTile)
	cell.Tile = &newTile

	return
}

type vector struct {
	X     int
	Y     int
	Label string
}

// dMap is a direction map
var dMap = map[int]vector{
	3: vector{ //Up
		X:     0,
		Y:     1,
		Label: "Up",
	},
	2: vector{ //Right
		X:     1,
		Y:     0,
		Label: "Right",
	},
	1: vector{ //Down
		X:     0,
		Y:     -1,
		Label: "Down",
	},
	4: vector{ //Left
		X:     -1,
		Y:     0,
		Label: "Left",
	},
}

func (g *Grid) getEdge(v *vector) (start, end, delta int) {
	if v.X == 1 || v.Y == 1 {
		end = -1
		start = g.Size - 1
		delta = -1
	} else {
		end = g.Size
		start = 0
		delta = 1
	}

	return start, end, delta
}

//Shift does:
//Get direction
//For table slice (vector perpendicular to direction)
//For row cell (vector opposite to direction)
//If cell has Tile
//Add to 'new row' slice
//If Tile value matches prev Tile value
//Merge Tile to prev
//Remove Tile ([2, 2, 4] .. [2 <-- 2, 4] .. [4, 4])
func (g *Grid) Shift(d int) *Grid {
	if g.GameOver {
		return g
	}

	v := dMap[d]

	moved := false

	start, end, delta := g.getEdge(&v)

	for i := start; i != end; i += delta {
		f2 := start
		for j := start; j != end; j += delta {

			var cell, dest *Cell

			if v.X == 0 { //Horizontal motion is zero. Direction is in Y
				cell = g.Cells[i][j]
				dest = g.Cells[i][f2]
			} else {
				cell = g.Cells[j][i]
				dest = g.Cells[f2][i]
			}

			if cell.Tile != nil { //If there's something here and somewhere to move it. Otherwise do nothing this iteration
				cell.Tile.New = false
				cell.Tile.Merged = false
				if dest.Pos != cell.Pos { //If they're not the same cells
					if dest.Tile != nil {
						if dest.Tile.Value == cell.Tile.Value { //If the value at the second finger matches the value at the current finger, merge.
							//Do a merge.
							dest.Tile.Merge(cell.Tile)
							g.Score += dest.Tile.Value
							if dest.Tile.Value == g.MaxScore {
								fmt.Printf("YOU WIN! Score: %d", g.Score)
								printGrid(g)
								fmt.Println()
								g.Reset()
							}
							//Now remove the old
							moved = true
							g.Tiles = g.Tiles.remove(cell.Tile)
							//Always increment finger2 after a merge
							f2 += delta
							cell.Tile = nil
						} else {
							f2 += delta
							if v.X == 0 {
								//Now reevaluate the cells changing.
								dest = g.Cells[i][f2]
							} else {
								dest = g.Cells[f2][i]
							}

							if dest.Pos != cell.Pos {
								cell.Tile.Move(dest)
								moved = true
								dest.Tile = cell.Tile
								cell.Tile = nil
							}
						}
					} else {
						cell.Tile.Move(dest)
						moved = true
						dest.Tile = cell.Tile
						cell.Tile = nil
					}
				}
			}
		}
	}

	if len(g.Tiles) == g.Size*g.Size {
		if !g.matchesRemaining() {
			g.GameOver = true
		}
	}

	if moved {
		g.newTile()
	}

	return g
}

func printGrid(g *Grid) {
	fmt.Println()
	for y := g.Size - 1; y >= 0; y-- {
		for x := 0; x < g.Size; x++ {
			if g.Cells[x][y].Tile != nil {
				fmt.Printf("[ ")
				fmt.Printf("%d ", g.Cells[x][y].Tile.Value)
				fmt.Printf("]")
			} else {
				fmt.Printf("[   ]")
			}
		}
		fmt.Println()
	}
}

// Determine if that are more possible moves
func (g *Grid) matchesRemaining() bool {
	matchFound := false
	for i := 0; i < g.Size; i++ {
		for j := 0 + i%2; j < g.Size; j = j + 2 {
			for k := 1; k <= 4; k++ {
				v := dMap[k]
				cell := g.Cells[i][j]

				x := i + v.X
				y := j + v.Y

				if y >= 0 && y < g.Size && x >= 0 && x < g.Size {
					cmp := g.Cells[x][y]
					//fmt.Printf("vector: %v, cell: %v, compare: %v, values: %v, %v", v, cell.Pos, cmp.Pos, cell.Tile.Value, cmp.Tile.Value)
					//fmt.Println()

					if cell.Tile.Value == cmp.Tile.Value {
						matchFound = true
						return matchFound
					}
				}
			}
		}
	}
	return matchFound
}

//EmptyCells return an array pointing to empty/available cells
func (g *Grid) EmptyCells() (ret []*Cell) {
	for i := 0; i < g.Size; i++ {
		for j := 0; j < g.Size; j++ {
			c := g.Cells[i][j]
			if c.Tile == nil {
				ret = append(ret, c)
			}
		}
	}

	return ret
}

//NewGrid returns a new grid. Negative moves sent to the m channel will terminate the grid.
func NewGrid(s, c, m int) (gr *Grid, ch chan *Grid, mv chan int) {
	ch = make(chan *Grid)
	mv = make(chan int)

	g := Grid{
		Size:       s,
		StartCells: c,
		Score:      0,
		MaxScore:   m,
	}

	g.Build()

	go func() {
		defer close(ch)
		defer close(mv)

		for {
			select {
			case move := <-mv:
				if move == 0 {
					g.Reset()
					ch <- &g
				} else {
					//Make a move and send the result
					ch <- g.Shift(move)
				}
			}
		}
	}()

	gr = &g

	return gr, ch, mv
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
