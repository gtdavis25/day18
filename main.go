/*
day18 is a solution to the day 18 puzzle from Advent of Code 2019 - see https://adventofcode.com/2019/day/18

It reads an ASCII maze and prints the shortest path which collects all of the keys in the maze (represented by lower-case characters).

If arguments are provided, the first argument is assumed to be the path of the input file. Otherwise, input is read from standard input.
*/
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

func main() {
	r := os.Stdin
	if len(os.Args) > 1 {
		f, err := os.Open(os.Args[1])
		defer f.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		r = f
	}
	m := readMaze(r)
	initial := state{cells: m.start(), keys: 0}
	result := shortestPath(m, initial, make(map[string]int))
	fmt.Printf("%d\n", result)
}

// maze represents the maze.
type maze struct {
	w, h int
	rows [][]*cell
	keys keyset
}

// readMaze reads a maze from r and returns it. The input is assumed to be a rectangular grid of characters.
func readMaze(r io.Reader) *maze {
	var rows []string
	for scanner := bufio.NewScanner(r); scanner.Scan(); {
		rows = append(rows, scanner.Text())
	}
	m := newMaze(len(rows[0]), len(rows))
	for i := range rows {
		for j := range rows[i] {
			if char := rows[i][j]; char != '#' {
				m.addCell(i, j, newCell(char))
			}
		}
	}
	m.buildPaths()
	return m
}

// newMaze initialises a new maze with width w and height h.
func newMaze(w, h int) *maze {
	cells := make([]*cell, w*h)
	rows := make([][]*cell, h)
	for i := range rows {
		rows[i] = cells[i*w : (i+1)*w]
	}
	return &maze{w, h, rows, 0}
}

// addCell adds c to m at row i and column j, joining c to any neighbours and, if c is a key, adds its value to m's keyset.
func (m *maze) addCell(i, j int, c *cell) {
	m.rows[i][j] = c
	if i > 0 && m.rows[i-1][j] != nil {
		c.join(m.rows[i-1][j])
	}
	if j > 0 && m.rows[i][j-1] != nil {
		c.join(m.rows[i][j-1])
	}
	if i+1 < m.h && m.rows[i+1][j] != nil {
		c.join(m.rows[i+1][j])
	}
	if j+1 < m.w && m.rows[i][j+1] != nil {
		c.join(m.rows[i][j+1])
	}
	if c.cellType == key {
		m.keys = m.keys.plus(c.char)
	}
}

// start returns a slice containing all start cells in m.
func (m *maze) start() []*cell {
	var startCells []*cell
	for i := range m.rows {
		for _, c := range m.rows[i] {
			if c != nil && c.cellType == start {
				startCells = append(startCells, c)
			}
		}
	}
	return startCells
}

// buildPaths populates the path list for each start and key cell in m.
func (m *maze) buildPaths() {
	for i := range m.rows {
		for _, c := range m.rows[i] {
			if c != nil && (c.cellType == key || c.cellType == start) {
				c.paths = findPaths(c)
			}
		}
	}
}

// cell represents a (non-wall) cell in the maze.
type cell struct {
	char     byte
	adj      []*cell
	paths    []path
	cellType cellType
}

// newCell returns a new cell with the value char and initialises its cellType.
func newCell(char byte) *cell {
	c := &cell{char: char}
	switch {
	case char == '@':
		c.cellType = start
	case 'a' <= char && char <= 'z':
		c.cellType = key
	case 'A' <= char && char <= 'Z':
		c.cellType = door
	}
	return c
}

// join adds c1 to c's adjacency list, and vice versa.
func (c *cell) join(c1 *cell) {
	c.adj = append(c.adj, c1)
	c1.adj = append(c1.adj, c)
}

// cellType represents the type of a cell: empty, start, key or door.
type cellType int

const (
	empty cellType = iota
	start
	key
	door
)

// keyset represents a set of maze keys (lower-case ASCII characters) as a bitmap.
type keyset uint

// contains returns true if k contains c, and false otherwise.
func (k keyset) contains(char byte) bool {
	return k>>(char-'a')&1 == 1
}

// plus returns a new keyset which is the result of adding char to k.
func (k keyset) plus(char byte) keyset {
	return k | 1<<(char-'a')
}

// containsAll returns true if keys is a subset of k, and false otherwise.
func (k keyset) containsAll(keys keyset) bool {
	return k&keys == keys
}

// path represents a path between two cells and includes the set of keys required to traverse it.
type path struct {
	len     int
	dest    *cell
	reqKeys keyset
}

// findPaths performs a breadth-first search of the cells reachable from c,
// and returns a slice containing the shortest paths to all reachable keys.
func findPaths(c *cell) []path {
	var paths []path
	start := path{len: 0, dest: c}
	seen := map[*cell]bool{c: true}
	for q := []path{start}; len(q) > 0; q = q[1:] {
		current := q[0]

		// If this path ends at a key, add it to the list of paths to return.
		if current.dest.cellType == key {
			paths = append(paths, current)
		}
		for _, adj := range current.dest.adj {
			if seen[adj] {
				continue
			}
			seen[adj] = true
			next := path{dest: adj, len: current.len + 1, reqKeys: current.reqKeys}

			// If adj is a door, then add its corresponding key to the path's required keys.
			if adj.cellType == door {
				next.reqKeys = next.reqKeys.plus(adj.char | 32)
			}
			q = append(q, next)
		}
	}
	return paths
}

// shortestPath returns the length of the shortest path from s to the end state where we have collected all of the keys in m.
// The table parameter massively reduces the number of recursive calls to shortestPath by memoizing partial results - pass an empty map.
func shortestPath(m *maze, s state, table map[string]int) int {
	stateKey := s.String()

	// If we've collected all the keys, we're done.
	if s.keys == m.keys {
		return 0
	}

	// If we've calculated this path before, return the memoized result.
	if d, ok := table[stateKey]; ok {
		return d
	}

	// Calculate the total weight of each possible path from s. The result is the smallest such weight.
	var min int
	for i, cell := range s.cells {
		for _, path := range cell.paths {

			// If this path leads to a key we've already collected, or if it passes through a door we can't open, ignore it.
			if s.keys.contains(path.dest.char) || !s.keys.containsAll(path.reqKeys) {
				continue
			}

			// Create the next state as a copy of the current state, replacing the current cell with the new cell and adding the new cell's key.
			nextState := s.copy()
			nextState.cells[i] = path.dest
			nextState.keys = s.keys.plus(path.dest.char)

			// The total weight of this path is the length of the path, plus the length of the shortest path from the next state to the end state.
			dist := path.len + shortestPath(m, nextState, table)
			if min == 0 || dist < min {
				min = dist
			}
		}
	}

	// Memoize the result so we don't have to calculate it again.
	table[stateKey] = min
	return min
}

// state represents the current state of a maze traversal, including the list of current positions and the set of collected keys.
type state struct {
	cells []*cell
	keys  keyset
}

// String returns a unique string representation of s. Used as a map key for memoization.
func (s state) String() string {
	cells := make([]byte, len(s.cells))
	for i := range cells {
		cells[i] = s.cells[i].char
	}
	return fmt.Sprintf("%s%d", cells, s.keys)
}

// copy returns a copy of s.
func (s state) copy() state {
	newState := state{cells: make([]*cell, len(s.cells)), keys: s.keys}
	copy(newState.cells, s.cells)
	return newState
}
