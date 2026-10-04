package engine

import (
	"errors"
	"fmt"
)

// ErrTerminal is returned by Step when the VM already finished.
var ErrTerminal = errors.New("engine: vm is in a terminal state")

// ProgramError describes an invalid program.
type ProgramError struct{ Issues []string }

func (e *ProgramError) Error() string {
	s := "invalid program"
	for i, is := range e.Issues {
		if i == 0 {
			s += ": "
		} else {
			s += "; "
		}
		s += is
	}
	return s
}

// ValidateProgram checks tape lengths and instruction legality against m.
func ValidateProgram(m *Map, p Program) error {
	var issues []string
	if len(p.Main) != m.MainTapeLength {
		issues = append(issues, fmt.Sprintf("main tape has %d slots, want %d", len(p.Main), m.MainTapeLength))
	}
	if len(p.Subs) != len(m.SubTapeLengths) {
		issues = append(issues, fmt.Sprintf("program has %d subs, want %d", len(p.Subs), len(m.SubTapeLengths)))
	}
	check := func(name string, tape []Instruction) {
		for i, ins := range tape {
			if !ins.Valid() {
				issues = append(issues, fmt.Sprintf("%s[%d]: unknown instruction %q", name, i, ins))
				continue
			}
			if k, ok := ins.SubIndex(); ok && k >= len(m.SubTapeLengths) {
				issues = append(issues, fmt.Sprintf("%s[%d]: %s but map has %d subs", name, i, ins, len(m.SubTapeLengths)))
			}
		}
	}
	check("main", p.Main)
	for i, s := range p.Subs {
		if i < len(m.SubTapeLengths) && len(s) != m.SubTapeLengths[i] {
			issues = append(issues, fmt.Sprintf("sub%d has %d slots, want %d", i+1, len(s), m.SubTapeLengths[i]))
		}
		check(fmt.Sprintf("sub%d", i+1), s)
	}
	if len(issues) > 0 {
		return &ProgramError{Issues: issues}
	}
	return nil
}

// EmptyProgram returns an all-EMPTY program sized for m.
func EmptyProgram(m *Map) Program {
	p := Program{Main: make([]Instruction, m.MainTapeLength), Subs: make([][]Instruction, len(m.SubTapeLengths))}
	for i := range p.Main {
		p.Main[i] = Empty
	}
	for i, l := range m.SubTapeLengths {
		p.Subs[i] = make([]Instruction, l)
		for j := range p.Subs[i] {
			p.Subs[i][j] = Empty
		}
	}
	return p
}

func tapeOf(p *Program, t int) []Instruction {
	if t < 0 {
		return p.Main
	}
	return p.Subs[t]
}

// NewVMState returns the initial state for m. The program is needed only to
// settle past leading EMPTY slots, so it is taken separately via Settle.
func NewVMState(m *Map) VMState {
	return VMState{
		Pos:        m.Start,
		Facing:     m.StartFacing,
		Collected:  make([]bool, len(m.Treats)),
		TreatsLeft: len(m.Treats),
		CallStack:  append(make([]Frame, 0, 8), Frame{Tape: -1, PC: 0}),
		Status:     StatusReady,
		Visited:    []Position{m.Start},
	}
}

// Settle skips uncounted EMPTY slots and pops finished frames so the top
// frame points at a real instruction. If the main tape is exhausted the VM is
// marked LOST/PROGRAM_ENDED (unless already terminal). Call once after
// NewVMState and after any state change made outside Step.
func (s *VMState) Settle(p *Program) {
	if s.Status.Terminal() {
		return
	}
	for len(s.CallStack) > 0 {
		top := &s.CallStack[len(s.CallStack)-1]
		tape := tapeOf(p, top.Tape)
		if top.PC >= len(tape) {
			if len(s.CallStack) == 1 {
				s.Status = StatusLost
				s.LossReason = LossProgramEnded
				return
			}
			s.CallStack = s.CallStack[:len(s.CallStack)-1]
			continue
		}
		if tape[top.PC] == Empty {
			top.PC++
			continue
		}
		return
	}
}

// Step executes one counted instruction. The caller owns s. It returns
// ErrTerminal if the VM already finished. The program must have been
// validated and s settled (NewVMState+Settle, or a previous Step).
func (s *VMState) Step(m *Map, p *Program) (StepEvent, error) {
	var ev StepEvent
	if err := s.step(m, p, &ev); err != nil {
		return StepEvent{}, err
	}
	return ev, nil
}

// step is Step with an optional event sink; ev==nil skips event construction
// (no per-step allocations for solver/validator loops).
func (s *VMState) step(m *Map, p *Program, evp *StepEvent) error {
	if s.Status.Terminal() {
		return ErrTerminal
	}
	s.Settle(p)
	if s.Status.Terminal() { // program was empty / already finished
		return ErrTerminal
	}
	top := &s.CallStack[len(s.CallStack)-1]
	tape, slot := top.Tape, top.PC
	ins := tapeOf(p, tape)[slot]
	top.PC++
	s.Steps++
	s.Status = StatusRunning

	from, facingBefore := s.Pos, s.Facing
	blocked := false
	var collected *Position

	switch ins {
	case TurnLeft:
		s.Facing = (s.Facing + 3) % 4
	case TurnRight:
		s.Facing = (s.Facing + 1) % 4
	case MoveForward:
		d := dirs[s.Facing]
		nx, ny := s.Pos.X+d[0], s.Pos.Y+d[1]
		if m.IsRock(nx, ny) {
			blocked = true
		} else {
			s.Pos = Position{nx, ny}
			s.Visited = append(s.Visited, s.Pos)
			if ti := m.treatIdx[ny*m.Width+nx]; ti >= 0 && !s.Collected[ti] {
				s.Collected[ti] = true
				s.TreatsLeft--
				if evp != nil {
					c := s.Pos
					collected = &c
				}
			}
		}
	default:
		if k, ok := ins.SubIndex(); ok {
			if !m.AllowRecursion && s.onStack(k) {
				break // no-op, still a step
			}
			s.CallStack = append(s.CallStack, Frame{Tape: k, PC: 0})
			if len(s.CallStack)-1 > m.MaxCallDepth {
				s.Status = StatusLost
				s.LossReason = LossCallDepth
			}
		}
	}

	if !s.Status.Terminal() {
		switch {
		case s.TreatsLeft == 0:
			s.Status = StatusWon
		case s.Steps >= m.MaxSteps:
			s.Status = StatusLost
			s.LossReason = LossStepLimit
		default:
			s.Settle(p)
		}
	}

	if evp != nil {
		*evp = StepEvent{
			Step: s.Steps, Instruction: ins, TapeIndex: tape, SlotIndex: slot,
			From: from, To: s.Pos, FacingBefore: facingBefore, FacingAfter: s.Facing,
			Blocked: blocked, Collected: collected,
			CallStack: append([]Frame(nil), s.CallStack...),
			Status:    s.Status, LossReason: s.LossReason,
		}
	}
	return nil
}

func (s *VMState) onStack(sub int) bool {
	for _, f := range s.CallStack {
		if f.Tape == sub {
			return true
		}
	}
	return false
}

// Simulate runs from s until terminal or limit steps (limit<=0 means
// m.MaxSteps). It returns every event and the final state. s is not mutated.
func Simulate(m *Map, p Program, s VMState, limit int) ([]StepEvent, VMState) {
	if limit <= 0 {
		limit = m.MaxSteps
	}
	st := s.Clone()
	st.Settle(&p)
	var events []StepEvent
	for i := 0; i < limit && !st.Status.Terminal(); i++ {
		ev, err := st.Step(m, &p)
		if err != nil {
			break
		}
		events = append(events, ev)
	}
	return events, st
}

// Run is a convenience: fresh state, full simulation, no events allocated.
func Run(m *Map, p Program) VMState {
	st := NewVMState(m)
	st.Settle(&p)
	for !st.Status.Terminal() {
		if err := st.step(m, &p, nil); err != nil {
			break
		}
	}
	return st
}
