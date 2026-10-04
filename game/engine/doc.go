// Package engine implements the Dr Brain - Motor Programming robot-puzzle rules: map parsing,
// program validation and the tape VM. It has no dependencies, globals or I/O.
//
// Coordinates: x = column, y = row, row 0 is the top of the layout and y grows
// downward. Facing 0=up(y-1) 1=right 2=down 3=left.
package engine
