package brat

// (c) Christian Maurer   v. 130115 - license see murus.go

import
  . "murus/obj"
type
  Rational interface {

  Editor
  Stringer
  Printer
  Adder
  Multiplier

// x = 1/x0, where x0 denotes x before.
  Invert ()

  RealVal () float64
  Set (a, b int) bool
  Integer () bool
  GeqNull () bool
}
