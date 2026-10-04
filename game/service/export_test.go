package service

// RunnerCount returns the number of registered runners (test hook).
func RunnerCount(g GameService) int {
	svc := g.(*gameService)
	svc.rmu.Lock()
	defer svc.rmu.Unlock()
	return len(svc.runners)
}
