package utils

type Semaphore struct {
	limit int
	ch    chan struct{}
}

func NewSemaphore(limit int) *Semaphore {
	return &Semaphore{
		limit: limit,
		ch:    make(chan struct{}, limit),
	}
}

func (s *Semaphore) Acquire() {
	s.ch <- struct{}{}
}

func (s *Semaphore) Release() {
	<-s.ch
}
