package graph

import (
	"github.com/wricardo/drbrain-motor-programming/game/engine"
	"github.com/wricardo/drbrain-motor-programming/game/service"
	"github.com/wricardo/drbrain-motor-programming/graph/model"
)

var facingNames = [...]model.Facing{
	engine.FacingUp:    model.FacingUp,
	engine.FacingRight: model.FacingRight,
	engine.FacingDown:  model.FacingDown,
	engine.FacingLeft:  model.FacingLeft,
}

func toFacing(f engine.Facing) model.Facing {
	if f < 0 || int(f) >= len(facingNames) {
		return model.FacingUp
	}
	return facingNames[f]
}

func toPosition(p engine.Position) *model.Position {
	return &model.Position{X: p.X, Y: p.Y}
}

func toPositions(ps []engine.Position) []*model.Position {
	out := make([]*model.Position, len(ps))
	for i, p := range ps {
		out[i] = toPosition(p)
	}
	return out
}

func toFrames(fs []engine.Frame) []*model.Frame {
	out := make([]*model.Frame, len(fs))
	for i, f := range fs {
		out[i] = &model.Frame{Tape: f.Tape, Pc: f.PC}
	}
	return out
}

func toLossReason(r engine.LossReason) *model.LossReason {
	if r == engine.LossNone {
		return nil
	}
	lr := model.LossReason(r)
	return &lr
}

func toInstructions(in []engine.Instruction) []model.Instruction {
	out := make([]model.Instruction, len(in))
	for i, v := range in {
		out[i] = model.Instruction(v)
	}
	return out
}

func toMap(m *engine.Map) *model.Map {
	if m == nil {
		return nil
	}
	return &model.Map{
		ID: m.ID, Name: m.Name, Description: m.Description, Difficulty: m.Difficulty,
		Width: m.Width, Height: m.Height,
		Layout:         append([]string{}, m.Layout...),
		Start:          toPosition(m.Start),
		StartFacing:    toFacing(m.StartFacing),
		Rocks:          toPositions(m.Rocks),
		Treats:         toPositions(m.Treats),
		MainTapeLength: m.MainTapeLength,
		SubTapeLengths: append([]int{}, m.SubTapeLengths...),
		MaxSteps:       m.MaxSteps,
		MaxCallDepth:   m.MaxCallDepth,
		AllowRecursion: m.AllowRecursion,
	}
}

func toMaps(ms []*engine.Map) []*model.Map {
	out := make([]*model.Map, len(ms))
	for i, m := range ms {
		out[i] = toMap(m)
	}
	return out
}

func toProgram(p engine.Program) *model.Program {
	subs := make([][]model.Instruction, len(p.Subs))
	for i, s := range p.Subs {
		subs[i] = toInstructions(s)
	}
	return &model.Program{Main: toInstructions(p.Main), Subs: subs}
}

// fromProgramInput converts the GraphQL input to an engine program. Semantic
// validation (tape lengths, sub indices) is the service's job.
func fromProgramInput(in model.ProgramInput) engine.Program {
	p := engine.Program{
		Main: make([]engine.Instruction, len(in.Main)),
		Subs: make([][]engine.Instruction, len(in.Subs)),
	}
	for i, v := range in.Main {
		p.Main[i] = engine.Instruction(v)
	}
	for i, s := range in.Subs {
		p.Subs[i] = make([]engine.Instruction, len(s))
		for j, v := range s {
			p.Subs[i][j] = engine.Instruction(v)
		}
	}
	return p
}

func fromMapInput(in model.MapInput) engine.MapConfig {
	cfg := engine.MapConfig{
		ID: in.ID, Name: in.Name,
		Layout: append([]string(nil), in.Layout...),
	}
	// nil = omitted (engine default); a non-nil empty list means zero subs.
	if in.SubTapeLengths != nil {
		cfg.SubTapeLengths = append([]int{}, in.SubTapeLengths...)
	}
	if in.Description != nil {
		cfg.Description = *in.Description
	}
	if in.Difficulty != nil {
		cfg.Difficulty = *in.Difficulty
	}
	if in.MainTapeLength != nil {
		cfg.MainTapeLength = *in.MainTapeLength
	}
	if in.MaxSteps != nil {
		cfg.MaxSteps = *in.MaxSteps
	}
	if in.MaxCallDepth != nil {
		cfg.MaxCallDepth = *in.MaxCallDepth
	}
	if in.AllowRecursion != nil {
		cfg.AllowRecursion = *in.AllowRecursion
	}
	return cfg
}

// toVMState converts a VM state; m supplies the treat positions behind the
// collected bitmap so clients never see it.
func toVMState(vm *engine.VMState, m *engine.Map) *model.VMState {
	remaining := []engine.Position{}
	if m != nil && len(vm.Collected) == len(m.Treats) {
		remaining = vm.TreatsRemaining(m)
	}
	return &model.VMState{
		Pos:             toPosition(vm.Pos),
		Facing:          toFacing(vm.Facing),
		TreatsRemaining: toPositions(remaining),
		CallStack:       toFrames(vm.CallStack),
		Steps:           vm.Steps,
		Status:          model.Status(vm.Status),
		LossReason:      toLossReason(vm.LossReason),
		Visited:         toPositions(vm.Visited),
	}
}

func toStepEvent(e *engine.StepEvent) *model.StepEvent {
	if e == nil {
		return nil
	}
	ev := &model.StepEvent{
		Step: e.Step, Instruction: model.Instruction(e.Instruction),
		TapeIndex: e.TapeIndex, SlotIndex: e.SlotIndex,
		From: toPosition(e.From), To: toPosition(e.To),
		FacingBefore: toFacing(e.FacingBefore), FacingAfter: toFacing(e.FacingAfter),
		Blocked:    e.Blocked,
		CallStack:  toFrames(e.CallStack),
		Status:     model.Status(e.Status),
		LossReason: toLossReason(e.LossReason),
	}
	if e.Collected != nil {
		ev.Collected = toPosition(*e.Collected)
	}
	return ev
}

// toSession converts an immutable session snapshot (a Clone).
func toSession(s *service.Session) *model.Session {
	if s == nil {
		return nil
	}
	out := &model.Session{
		ID: s.ID, DisplayName: s.DisplayName, MapID: s.MapID,
		Map:     toMap(s.Map),
		Program: toProgram(s.Program),
		VM:      toVMState(&s.VM, s.Map),
		Playing: s.Playing, SpeedMs: s.SpeedMs,
		Seq:       int(s.Seq),
		CreatedAt: s.CreatedAt, LastActionAt: s.LastActionAt,
		Attempts:  s.Attempts,
		LastEvent: toStepEvent(s.LastEvent),
	}
	if s.BestSteps > 0 {
		b := s.BestSteps
		out.BestSteps = &b
	}
	return out
}

func toSessionUpdate(s *service.Session) *model.SessionUpdate {
	return &model.SessionUpdate{Seq: int(s.Seq), Session: toSession(s), Event: toStepEvent(s.LastEvent)}
}

func toSimulationResult(r *service.SimulationResult) *model.SimulationResult {
	out := &model.SimulationResult{
		Status:     model.Status(r.Status),
		LossReason: toLossReason(r.LossReason),
		Steps:      r.Steps,
		FinalState: toVMState(&r.FinalState, r.Map),
	}
	if r.Events != nil {
		out.Events = make([]*model.StepEvent, len(r.Events))
		for i := range r.Events {
			out.Events[i] = toStepEvent(&r.Events[i])
		}
	}
	return out
}
