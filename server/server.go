package server

import "context"

type Server interface {
	Start(context.Context) error
	Stop(context.Context) error
}

type serverImpl struct {
	name string
	stop context.CancelFunc
}

func New(name string) Server {
	return &serverImpl{
		name: name,
	}
}

func (s *serverImpl) Start(ctx context.Context) error {
	return nil
}

func (s *serverImpl) Stop(ctx context.Context) error {
	return nil
}
